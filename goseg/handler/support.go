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
	"path"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/disk"
	"go.uber.org/zap"
)

const (
	bugEndpoint = "https://bugs.groundseg.app"
)

var (
	bugReportPath string
)

func init() {
	bugReportPath = makeBugReportPath()
}

// handle bug report requests
func SupportHandler(msg []byte) error {
	defer func() {
		time.Sleep(1 * time.Second)
		docker.SysTransBus <- structs.SystemTransition{Type: "bugReport", Event: ""}
		time.Sleep(2 * time.Second)
		docker.SysTransBus <- structs.SystemTransition{Type: "bugReportError", Event: ""}
	}()
	// transition to loading
	docker.SysTransBus <- structs.SystemTransition{Type: "bugReport", Event: "loading"}
	// error handling
	handleError := func(err error) error {
		docker.SysTransBus <- structs.SystemTransition{Type: "bugReportError", Event: fmt.Sprintf("%v", err)}
		return err
	}

	// initialize payload
	var supportPayload structs.WsSupportPayload
	if err := json.Unmarshal(msg, &supportPayload); err != nil {
		return handleError(fmt.Errorf("Couldn't unmarshal support payload: %v", err))
	}
	// unix timestamp
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	// extract payload data
	contact := supportPayload.Payload.Contact
	description := supportPayload.Payload.Description
	ships := supportPayload.Payload.Ships
	cpuProfile := supportPayload.Payload.CPUProfile
	penpai := supportPayload.Payload.Penpai

	// set bug report dir
	bugReportDir := filepath.Join(bugReportPath, timestamp)

	// create the directory
	if err := os.MkdirAll(bugReportDir, 0755); err != nil {
		return handleError(fmt.Errorf("Error creating bug-report dir: %v", err))
	}

	// write bug report to disk
	if err := dumpBugReport(bugReportDir, timestamp, contact, description, ships, penpai); err != nil {
		return handleError(fmt.Errorf("Failed to dump logs: %v", err))
	}

	// collect cpu profile
	if cpuProfile {
		profilePath := filepath.Join(bugReportDir, "cpu.profile")
		if err := captureCPUProfile(profilePath); err != nil {
			return handleError(fmt.Errorf("Couldn't collect CPU profile: %v", err))
		}
	}

	// set zipfile location on disk
	zipPath := filepath.Join(bugReportPath, timestamp+".zip")
	if err := zipDir(bugReportDir, zipPath); err != nil {
		return handleError(fmt.Errorf("Error zipping bug report: %v", err))
	}

	// remove the bug report since we already have the zipped version
	if err := os.RemoveAll(bugReportDir); err != nil {
		zap.L().Warn(fmt.Sprintf("Couldn't remove report dir: %v", err))
	}

	// send bug report
	if err := sendBugReport(zipPath, contact, description); err != nil {
		return handleError(fmt.Errorf("Couldn't submit bug report: %v", err))
	}

	// completion transitions
	docker.SysTransBus <- structs.SystemTransition{Type: "bugReport", Event: "success"}
	time.Sleep(3 * time.Second)
	docker.SysTransBus <- structs.SystemTransition{Type: "bugReport", Event: "done"}
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

func dumpBugReport(bugReportDir, timestamp, contact, description string, piers []string, llama bool) error {

	// description.txt
	descPath := filepath.Join(bugReportDir, "description.txt")
	if err := ioutil.WriteFile(descPath, []byte(fmt.Sprintf("Contact:\n%s\nDetails:\n%s", contact, description)), 0644); err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't write details.txt"))
		return err
	}

	// llama bug dump
	if llama {
		if err := dumpDockerLogs("llama-gpt-api", bugReportDir+"/"+"llama.log"); err != nil {
			zap.L().Warn(fmt.Sprintf("Couldn't dump llama logs: %v", err))
		}
	}

	// selected pier logs
	for _, pier := range piers {
		if err := dumpDockerLogs(pier, bugReportDir+"/"+pier+".log"); err != nil {
			zap.L().Warn(fmt.Sprintf("Couldn't dump pier logs: %v", err))
		}
		if err := dumpDockerLogs("minio_"+pier, bugReportDir+"/minio_"+pier+".log"); err != nil {
			zap.L().Warn(fmt.Sprintf("Couldn't dump minio logs: %v", err))
		}
	}

	// service logs
	if err := dumpDockerLogs("wireguard", bugReportDir+"/wireguard.log"); err != nil {
		zap.L().Warn(fmt.Sprintf("Couldn't dump pier logs: %v", err))
	}

	// system.json
	srcPath := filepath.Join(config.BasePath, "settings", "system.json")
	destPath := filepath.Join(bugReportDir, "system.json")
	if err := copyFile(srcPath, destPath); err != nil {
		zap.L().Warn(fmt.Sprintf("Couldn't copy service configs: %v", err))
	}

	// remove private information
	if err := sanitizeJSON(destPath, "sessions", "privkey", "salt", "pwHash"); err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't sanitize system.json! Removing from report"))
		if err := os.Remove(destPath); err != nil {
			return fmt.Errorf("Error removing unsanitized system log: %v", err)
		}
	}

	// pier configs
	for _, pier := range piers {
		srcPath = filepath.Join(config.BasePath, "settings", "pier", pier+".json")
		destPath = filepath.Join(bugReportDir, pier+".json")
		if err := copyFile(srcPath, destPath); err != nil {
			zap.L().Warn(fmt.Sprintf("Couldn't copy service configs: %v", err))
		}
		if err := sanitizeJSON(destPath, "minio_password"); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't sanitize %v.json! Removing from report", pier))
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
			zap.L().Warn(fmt.Sprintf("Couldn't copy service configs: %v", err))
		}
	}

	// current and previous syslogs
	sysLogs := lastTwoLogs()
	for _, sysLog := range sysLogs {
		srcPath := sysLog
		destPath := filepath.Join(bugReportDir, filepath.Base(sysLog))
		if err := copyFile(srcPath, destPath); err != nil {
			zap.L().Warn(fmt.Sprintf("Couldn't copy system logs: %v", err))
		}
	}
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
	zap.L().Info("Bug: Report submitted")
	return nil
}

func captureCPUProfile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	zap.L().Info("Profiling CPU for 30 seconds")
	if err := pprof.StartCPUProfile(f); err != nil {
		return err
	}
	time.Sleep(30 * time.Second)
	pprof.StopCPUProfile()
	return nil
}

func makeBugReportPath() string {
	// check if the basePath (or its parents) is a mountpoint with gopsutil
	bpCopy := config.BasePath

	partitions, err := disk.Partitions(true)
	if err != nil {
		zap.L().Error(fmt.Sprintf("failed to get list of partitions! Defaulting to BasePath"))
		return filepath.Join(config.BasePath, "bug-reports")
	}

	/*
		the outer loop loops from child up the unix path
		until a mountpoint is found
	*/
	//var mountpoint string
OuterLoop:
	for {
		for _, p := range partitions {
			if p.Mountpoint == bpCopy {
				devType := "mmc"
				if strings.Contains(p.Device, devType) {
					return "/media/data/bug-reports/"
				} else {
					break OuterLoop
				}
			}
		}
		if bpCopy == "/" {
			break
		}
		bpCopy = path.Dir(bpCopy) // Reduce the path by one level
	}
	return filepath.Join(config.BasePath, "/bug-reports/")
}

// LogFile represents a structured log file with a date and part number.
type LogFile struct {
	Name string
	Date string
	Part int
}

// lastTwoLogs returns the two most recent log files from the directory.
func lastTwoLogs() []string {
	// Read the directory
	files, err := ioutil.ReadDir(logger.LogPath)
	if err != nil {
		fmt.Printf("Failed to read directory: %v\n", err)
		return nil
	}

	// Create a slice to store LogFile structs
	var logFiles []LogFile

	// Filter and parse log files
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".log") {
			parts := strings.Split(file.Name(), "-part-")
			if len(parts) == 2 {
				date := parts[0]
				partNumber, err := strconv.Atoi(strings.TrimSuffix(parts[1], ".log"))
				if err != nil {
					fmt.Printf("Failed to parse part number from file: %s\n", file.Name())
					continue
				}
				logFiles = append(logFiles, LogFile{
					Name: file.Name(),
					Date: date,
					Part: partNumber,
				})
			}
		}
	}

	// Sort log files first by Date, then by Part (both in descending order)
	sort.Slice(logFiles, func(i, j int) bool {
		if logFiles[i].Date == logFiles[j].Date {
			return logFiles[i].Part > logFiles[j].Part
		}
		return logFiles[i].Date > logFiles[j].Date
	})

	// Get the two most recent log files
	var recentLogs []string
	for i := 0; i < len(logFiles) && i < 2; i++ {
		recentLogs = append(recentLogs, filepath.Join(logger.LogPath, logFiles[i].Name))
	}

	return recentLogs
}
