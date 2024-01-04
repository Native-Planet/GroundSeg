package handler

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/logger"
	"groundseg/structs"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	bugEndpoint = "https://bugs.groundseg.app"
)

// handle bug report requests
func SupportHandler(msg []byte) error {
	docker.SysTransBus <- structs.SystemTransition{Type: "bugReport", Event: "loading"}
	handleError := func(err error) error {
		docker.SysTransBus <- structs.SystemTransition{Type: "bugReportError", Event: fmt.Sprintf("%v", err)}
		return err
	}
	var supportPayload structs.WsSupportPayload
	if err := json.Unmarshal(msg, &supportPayload); err != nil {
		return handleError(fmt.Errorf("Couldn't unmarshal support payload: %v", err))
	}
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	contact := supportPayload.Payload.Contact
	description := supportPayload.Payload.Description
	ships := supportPayload.Payload.Ships
	cpuProfile := supportPayload.Payload.CPUProfile
	penpai := supportPayload.Payload.Penpai
	bugReportDir := filepath.Join(config.BasePath, "bug-reports", timestamp)
	if err := os.MkdirAll(bugReportDir, 0755); err != nil {
		return handleError(fmt.Errorf("Error creating bug-report dir: %v", err))
	}
	if err := dumpBugReport(timestamp, contact, description, ships, penpai); err != nil {
		return handleError(fmt.Errorf("Failed to dump logs: %v", err))
	}
	if cpuProfile {
		profilePath := filepath.Join(config.BasePath, "bug-reports", timestamp, "cpu.profile")
		if err := captureCPUProfile(profilePath); err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't collect CPU profile: %v", err))
		}
	}
	zipPath := filepath.Join(config.BasePath, "bug-reports", timestamp+".zip")
	if err := zipDir(bugReportDir, zipPath); err != nil {
		return handleError(fmt.Errorf("Error zipping bug report: %v", err))
	}
	if err := os.RemoveAll(bugReportDir); err != nil {
		logger.Logger.Warn(fmt.Sprintf("Couldn't remove report dir: %v", err))
	}
	if err := sendBugReport(zipPath, contact, description); err != nil {
		return handleError(fmt.Errorf("Couldn't submit bug report: %v", err))
	}
	docker.SysTransBus <- structs.SystemTransition{Type: "bugReport", Event: "success"}
	time.Sleep(3 * time.Second)
	docker.SysTransBus <- structs.SystemTransition{Type: "bugReport", Event: "done"}
	time.Sleep(1 * time.Second)
	docker.SysTransBus <- structs.SystemTransition{Type: "bugReport", Event: ""}
	return nil
}

// dump docker logs to path
func dumpDockerLogs(containerID string, path string) error {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("Error creating Docker client: %v", err)
	}
	defer dockerClient.Close()
	// get previous 1k logs
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Tail:       "1000",
	}
	ctx := context.Background()
	existingLogs, err := dockerClient.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return fmt.Errorf("Error dumping %v logs: %v", containerID, err)
	}
	defer existingLogs.Close()
	allLogs, err := ioutil.ReadAll(existingLogs)
	if err != nil {
		return fmt.Errorf("Error reading logs: %v", err)
	}
	var cleanedLogs strings.Builder
	reader := bytes.NewReader(allLogs)
	bufReader := bufio.NewReader(reader)
	for {
		header := make([]byte, 8)
		_, err := bufReader.Read(header)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Error reading log header: %v", err)
		}
		line, err := bufReader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("Error reading log line: %v", err)
		}
		cleanedLogs.WriteString(string(line))
	}
	err = ioutil.WriteFile(path, []byte(cleanedLogs.String()), 0644)
	if err != nil {
		return fmt.Errorf("Error writing logs to file: %v", err)
	}
	return nil
}

func dumpBugReport(timestamp, contact, description string, piers []string, llama bool) error {
	bugReportDir := filepath.Join(config.BasePath, "bug-reports", timestamp)
	descPath := filepath.Join(bugReportDir, "description.txt")
	// description.txt
	if err := ioutil.WriteFile(descPath, []byte(fmt.Sprintf("Contact:\n%s\nDetails:\n%s", contact, description)), 0644); err != nil {
		logger.Logger.Error(fmt.Sprintf("Couldn't write details.txt"))
		return err
	}
	if llama {
		if err := dumpDockerLogs("llama-gpt-api", bugReportDir+"/"+"llama.log"); err != nil {
			logger.Logger.Warn(fmt.Sprintf("Couldn't dump llama logs: %v", err))
		}
	}
	// selected pier logs
	for _, pier := range piers {
		if err := dumpDockerLogs(pier, bugReportDir+"/"+pier+".log"); err != nil {
			logger.Logger.Warn(fmt.Sprintf("Couldn't dump pier logs: %v", err))
		}
		if err := dumpDockerLogs("minio_"+pier, bugReportDir+"/minio_"+pier+".log"); err != nil {
			logger.Logger.Warn(fmt.Sprintf("Couldn't dump minio logs: %v", err))
		}
	}
	// service logs
	if err := dumpDockerLogs("wireguard", bugReportDir+"/wireguard.log"); err != nil {
		logger.Logger.Warn(fmt.Sprintf("Couldn't dump pier logs: %v", err))
	}
	// system.json
	srcPath := filepath.Join(config.BasePath, "settings", "system.json")
	destPath := filepath.Join(bugReportDir, "system.json")
	if err := copyFile(srcPath, destPath); err != nil {
		logger.Logger.Warn(fmt.Sprintf("Couldn't copy service configs: %v", err))
	}
	if err := sanitizeJSON(destPath, "sessions", "privkey", "salt", "pwHash"); err != nil {
		logger.Logger.Error(fmt.Sprintf("Couldn't sanitize system.json! Removing from report"))
		if err := os.Remove(destPath); err != nil {
			return fmt.Errorf("Error removing unsanitized system log: %v", err)
		}
	}
	// pier configs
	for _, pier := range piers {
		srcPath = filepath.Join(config.BasePath, "settings", "pier", pier+".json")
		destPath = filepath.Join(bugReportDir, pier+".json")
		if err := copyFile(srcPath, destPath); err != nil {
			logger.Logger.Warn(fmt.Sprintf("Couldn't copy service configs: %v", err))
		}
		if err := sanitizeJSON(destPath, "minio_password"); err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't sanitize %v.json! Removing from report", pier))
			if err := os.Remove(destPath); err != nil {
				return fmt.Errorf("Error removing unsanitized pier log: %v", err)
			}
		}
	}
	// service config jsons
	configFiles := []string{"mc.json", "netdata.json", "wireguard.json"}
	for _, configFile := range configFiles {
		srcPath = filepath.Join(config.BasePath, "settings", configFile)
		destPath = filepath.Join(bugReportDir, configFile)
		if err := copyFile(srcPath, destPath); err != nil {
			logger.Logger.Warn(fmt.Sprintf("Couldn't copy service configs: %v", err))
		}
	}
	// current and previous syslogs
	sysLogs := []string{logger.SysLogfile(), logger.PrevSysLogfile()}
	for _, sysLog := range sysLogs {
		srcPath := sysLog
		destPath := filepath.Join(bugReportDir, filepath.Base(sysLog))
		if err := copyFile(srcPath, destPath); err != nil {
			logger.Logger.Warn(fmt.Sprintf("Couldn't copy system logs: %v", err))
		}
	}
	/*
		if err := sanitizeJSON("sample.json", "key1", "key3"); err != nil {
		}
	*/
	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	return dstFile.Sync()
}

func zipDir(directory, dest string) error {
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()
	zipWriter := zip.NewWriter(destFile)
	defer zipWriter.Close()
	filepath.Walk(directory, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(directory, filePath)
		if err != nil {
			return err
		}
		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}
		fsFile, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer fsFile.Close()
		_, err = io.Copy(zipFile, fsFile)
		return err
	})
	return nil
}

func sanitizeJSON(filePath string, keysToRemove ...string) error {
	// Read the file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Unmarshal into a map
	var jsonData map[string]interface{}
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		return err
	}

	// Remove the keys
	for _, key := range keysToRemove {
		delete(jsonData, key)
	}

	// Marshal back to JSON
	sanitizedData, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return err
	}

	// Write back to the file
	err = ioutil.WriteFile(filePath, sanitizedData, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func sendBugReport(bugReportPath, contact, description string) error {
	uploadedFile, err := os.Open(bugReportPath)
	if err != nil {
		return err
	}
	defer uploadedFile.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("contact", contact)
	writer.WriteField("string", description)
	part, err := writer.CreateFormFile("zip_file", bugReportPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, uploadedFile)
	if err != nil {
		return err
	}
	writer.Close()
	req, err := http.NewRequest("POST", bugEndpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", resp.Status, resp.Body)
	}
	logger.Logger.Info("Bug: Report submitted")
	return nil
}

func captureCPUProfile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	logger.Logger.Info("Profiling CPU for 30 seconds")
	if err := pprof.StartCPUProfile(f); err != nil {
		return err
	}
	time.Sleep(30 * time.Second)
	pprof.StopCPUProfile()
	return nil
}
