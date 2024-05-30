package handler

import (
	"encoding/json"
	"fmt"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/logger"
	"groundseg/penpai"
	"groundseg/structs"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
			return fmt.Errorf("Failed to start penpai: %v", err)
		}
		return nil
	case "delete-model":
		model := penpaiPayload.Payload.Model
		for _, m := range conf.PenpaiModels {
			if model == m.ModelTitle {
				if err := penpaiDeleteModel(m); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("Failed to delete penpai model: %v", err)
				}
			}
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

func penpaiDeleteModel(m structs.Penpai) error {
	logger.Logger.Info(fmt.Sprintf("Requested to delete penpai model %v", m.ModelName))
	filename, err := penpai.ExtractLlamafileName(m.ModelUrl)
	if err != nil {
		return err
	}
	conf := config.Conf()
	modelFile := filepath.Join(conf.DockerData, "volumes", "llama-gpt-api", "_data", filename)
	err = os.Remove(modelFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var val *int
	for i, model := range conf.PenpaiModels {
		if model.ModelName == m.ModelName {
			val = &i
			break
		}
	}

	if val == nil {
		return fmt.Errorf("invalid index: %v", m.ModelName)
	}

	if err := penpai.SetExistence(m, false, *val); err != nil {
		return fmt.Errorf("Penpai routine failed to update config: %v", err)
	}
	logger.Logger.Info(fmt.Sprintf("Penpai model deleted %v", m.ModelName))
	return nil
}

func downloadModel(m structs.Penpai) error {
	defer penpai.Unlock(m.ModelHash)
	penpai.Lock(m.ModelHash)
	logger.Logger.Info(fmt.Sprintf("Requested to download penpai model %v", m.ModelTitle))
	conf := config.Conf()
	// retrieve file name
	filename, err := penpai.ExtractLlamafileName(m.ModelUrl)
	if err != nil {
		return penpaiHandleError(err)
	}

	// clear old file if exists
	logger.Logger.Info("Preparing the environment")
	modelFile := filepath.Join(conf.DockerData, "volumes", "llama-gpt-api", "_data", filename)
	err = os.Remove(modelFile)
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
	logger.Logger.Info("Fetching model from remote endpoint")
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

	// get hash
	logger.Logger.Info("Verifying llamafile hash")
	hash, err := penpai.GetSHA256(modelFile)
	if err != nil {
		return penpaiHandleError(err)
	}
	if hash != m.ModelHash {
		return penpaiHandleError(fmt.Errorf("Corrupted file download"))
	}

	var val *int
	for i, model := range conf.PenpaiModels {
		if model.ModelHash == hash {
			val = &i
			break
		}
	}

	if val == nil {
		return penpaiHandleError(fmt.Errorf("Invalid index"))
	}

	// update config
	logger.Logger.Info("Updating config")
	// update config
	if err := penpai.SetExistence(m, true, *val); err != nil {
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
		logger.Logger.Warn(fmt.Sprintf("%v%% done: %v, total: %v", float64(pr.Done)/float64(pr.Total)*100, pr.Done, pr.Total)) // dev
	}
	return n, err
}

func penpaiHandleError(err error) error {
	logger.Logger.Error(fmt.Sprintf("Failed to download penpai model: %v", err))
	return nil
}
