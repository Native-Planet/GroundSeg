package handler

import (
	"encoding/json"
	"fmt"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/logger"
	"groundseg/structs"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

func PenpaiHandler(msg []byte) error {
	logger.Logger.Info("Penpai")
	var penpaiPayload structs.WsPenpaiPayload
	err := json.Unmarshal(msg, &penpaiPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal penpai payload: %v", err)
	}
	conf := config.Conf()
	switch penpaiPayload.Payload.Action {
	case "toggle":
		running := false
		if conf.PenpaiRunning {
			// stop container
			err := docker.StopContainerByName("llama-gpt-api")
			if err != nil {
				return fmt.Errorf(fmt.Sprintf("Failed to stop Llama API: %v", err))
			}
		} else {
			// start container
			info, err := docker.StartContainer("llama-gpt-api", "llama-api")
			if err != nil {
				return fmt.Errorf(fmt.Sprintf("Error starting Llama API: %v", err))
			}
			config.UpdateContainerState("llama-api", info)
			running = true
		}
		if err = config.UpdateConf(map[string]interface{}{
			"penpaiRunning": running,
		}); err != nil {
			return fmt.Errorf(fmt.Sprintf("%v", err))
		}
		return nil
	case "download-model":
		model := penpaiPayload.Payload.Model
		for _, m := range conf.PenpaiModels {
			if model == m.ModelTitle {
				go downloadModel(m)
			}
		}
		return nil
	case "set-model":
		// update config
		model := penpaiPayload.Payload.Model
		if err = config.UpdateConf(map[string]interface{}{
			"penpaiActive": model,
		}); err != nil {
			return err
		}
		if err := docker.DeleteContainer("llama-gpt-api"); err != nil {
			return fmt.Errorf("Failed to delete container: %v", err)
		}
		// if running, restart container
		if conf.PenpaiRunning {
			if _, err := docker.StartContainer("llama-gpt-api", "llama-api"); err != nil {
				return fmt.Errorf("Couldn't start Llama API: %v", err)
			}
		}
	case "set-cores":
		cores := penpaiPayload.Payload.Cores
		// check if core count is valid
		if cores < 1 {
			return fmt.Errorf("Penpai unable to set 0 cores!")
		}
		if cores >= runtime.NumCPU() {
			return fmt.Errorf(fmt.Sprintf("Penpai unable to set %v cores!", cores))
		}
		// update config
		if err = config.UpdateConf(map[string]interface{}{
			"penpaiCores": cores,
		}); err != nil {
			return fmt.Errorf(fmt.Sprintf("%v", err))
		}
		if err := docker.DeleteContainer("llama-gpt-api"); err != nil {
			return fmt.Errorf("Failed to delete container: %v", err)
		}
		// if running, restart container
		if conf.PenpaiRunning {
			if _, err := docker.StartContainer("llama-gpt-api", "llama-api"); err != nil {
				return fmt.Errorf("Couldn't start Llama API: %v", err)
			}
		}
		return nil
	case "remove":
		// check if container exists
		// remove container, delete volume
		logger.Logger.Debug(fmt.Sprintf("Todo: remove penpai"))
	}
	return nil
}

func downloadModel(m structs.Penpai) error {
	conf := config.Conf()
	// retrieve file name
	filename, err := extractLlamafileName(m.ModelUrl)
	if err != nil {
		return penpaiHandleError(err)
	}

	// clear old file if exists
	modelFile := filepath.Join(conf.DockerData, "volumes", "llama-gpt-api", "_data", filename)
	os.Remove(modelFile)
	if err != nil && !os.IsNotExist(err) {
		return penpaiHandleError(err)
	}

	// Create the file
	out, err := os.Create(modelFile)
	if err != nil {
		return penpaiHandleError(err)
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(m.ModelUrl)
	if err != nil {
		return penpaiHandleError(err)
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return penpaiHandleError(fmt.Errorf("bad status: %s", resp.Status))
	}

	// Get the content length (total size)
	total := resp.ContentLength

	// Create a progress reader
	progressReader := &ProgressReader{
		Reader: resp.Body,
		Total:  total,
	}

	// Copy data from the response to the file, while tracking progress
	_, err = io.Copy(out, progressReader)
	if err != nil {
		return penpaiHandleError(err)
	}

	return nil
}

// ProgressReader is a custom reader that tracks progress
type ProgressReader struct {
	Reader io.Reader
	Total  int64
	Done   int64
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	if n > 0 {
		pr.Done += int64(n)
		logger.Logger.Warn(fmt.Sprintf("done: %v, total: %v", pr.Done, pr.Total))
	}
	return n, err
}

func extractLlamafileName(url string) (string, error) {
	re := regexp.MustCompile(`([^/]+\.llamafile)`)
	match := re.FindStringSubmatch(url)
	if len(match) > 1 {
		return match[1], nil
	}
	return "", fmt.Errorf("invalid download link: %v", url)
}

func penpaiHandleError(err error) error {
	logger.Logger.Error(fmt.Sprintf("Failed to download penpai model: %v", err))
	return nil
}
