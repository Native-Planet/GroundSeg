package exporter

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"groundseg/config"
	"groundseg/defaults"
	"groundseg/docker"
	"groundseg/structs"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

var (
	whitelist = make(map[string]structs.WsTokenStruct)
	exportMu  sync.Mutex
	exportDir string
)

func init() {
	exportDir = config.GetStoragePath("export")
	os.MkdirAll(exportDir, 0755)
	zap.L().Info(fmt.Sprintf("Using export directory: %s", exportDir))
}

func WhitelistContainer(container string, token structs.WsTokenStruct) error {
	exportMu.Lock()
	defer exportMu.Unlock()
	whitelist[container] = token
	zap.L().Info(fmt.Sprintf("Whitelisted %v for export", container))
	return nil
}

func RemoveContainerFromWhitelist(container string) error {
	exportMu.Lock()
	defer exportMu.Unlock()
	delete(whitelist, container)
	zap.L().Info(fmt.Sprintf("Removed %v from export whitelist", container))
	return nil
}

func ExportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	exportError := func(err error) {
		zap.L().Error(fmt.Sprintf("Unable to export: %v", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	vars := mux.Vars(r)
	container := vars["container"]

	exportMu.Lock()
	whitelistToken, exists := whitelist[container]
	exportMu.Unlock()

	if !exists {
		err := fmt.Errorf("container %v is not in whitelist!", container)
		zap.L().Error(fmt.Sprintf("Rejecting Export request: %v", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	patp := strings.TrimPrefix(container, "minio_")
	exportTrans := "exportShip"
	compressedTrans := "shipCompressed"
	isMinIO := strings.Contains(container, "minio_")
	if isMinIO {
		exportTrans = "exportBucket"
		compressedTrans = "bucketCompressed"
	}

	cleanup := func() {
		RemoveContainerFromWhitelist(container)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: exportTrans, Event: ""}
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: compressedTrans, Value: 0}
	}

	var tokenData structs.WsTokenStruct
	err := json.NewDecoder(r.Body).Decode(&tokenData)
	if err != nil {
		err := fmt.Errorf("export failed to decode token: %v", err)
		zap.L().Error(fmt.Sprintf("rejecting Export request: %v", err))
		cleanup()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if whitelistToken != tokenData {
		err := fmt.Errorf("token for exporting %v is not valid", container)
		zap.L().Error(fmt.Sprintf("Rejecting Export request: %v", err))
		cleanup()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !isMinIO {
		statusTicker := time.NewTicker(500 * time.Millisecond)
		defer statusTicker.Stop()
		timeout := time.After(30 * time.Second)

	loopLabel:
		for {
			select {
			case <-statusTicker.C:
				pierStatus, err := docker.GetShipStatus([]string{container})
				if err != nil {
					exportError(err)
					cleanup()
					return
				}
				status, exists := pierStatus[container]
				if !exists {
					exportError(fmt.Errorf("unable to export nonexistent container: %v", container))
					cleanup()
					return
				}
				if strings.Contains(status, "Exited") {
					break loopLabel
				}
			case <-timeout:
				exportError(fmt.Errorf("timeout waiting for container to stop"))
				cleanup()
				return
			}
		}
	}

	volumeDirectory := defaults.DockerData("volumes")
	filePath := filepath.Join(volumeDirectory, container, "_data")
	shipConf := config.UrbitConf(container)

	if customLoc, ok := shipConf.CustomPierLocation.(string); ok {
		filePath = customLoc
	}

	// Create a temporary file for the zip
	tempFile, err := os.CreateTemp(exportDir, container+"-*.zip")
	if err != nil {
		exportError(fmt.Errorf("failed to create temp file: %v", err))
		cleanup()
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Create zip writer that writes to the temp file
	zipWriter := zip.NewWriter(tempFile)

	// Count total files for progress reporting
	totalFiles := 0
	err = filepath.Walk(filePath, func(_ string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() {
			totalFiles++
		}
		return nil
	})
	if err != nil {
		exportError(fmt.Errorf("failed to count files: %v", err))
		cleanup()
		return
	}

	// Process files with progress reporting
	completedFiles := 0
	err = filepath.Walk(filePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		// Skip socket files
		if filepath.Base(path) == "conn.sock" {
			return nil
		}

		arcDir := strings.TrimPrefix(path, filePath+"/")
		f, err := zipWriter.Create(arcDir)
		if err != nil {
			return err
		}

		fileReader, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fileReader.Close()

		_, err = io.Copy(f, fileReader)
		if err != nil {
			return err
		}

		completedFiles++
		progress := int(float64(completedFiles) / float64(totalFiles) * 100)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: compressedTrans, Value: progress}
		return nil
	})
	if err != nil {
		exportError(fmt.Errorf("Failed during file archiving: %v", err))
		cleanup()
		return
	}

	// Close the zip writer to finalize the archive
	err = zipWriter.Close()
	if err != nil {
		exportError(fmt.Errorf("Failed to finalize zip: %v", err))
		cleanup()
		return
	}

	// Reset file pointer to beginning
	_, err = tempFile.Seek(0, 0)
	if err != nil {
		exportError(fmt.Errorf("Failed to reset file pointer: %v", err))
		cleanup()
		return
	}

	// Get file info for Content-Length header
	fileInfo, err := tempFile.Stat()
	if err != nil {
		exportError(fmt.Errorf("Failed to get file info: %v", err))
		cleanup()
		return
	}

	// Set appropriate headers
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", container))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Stream the file to the client
	_, err = io.Copy(w, tempFile)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Error streaming file to client: %v", err))
	}

	// Cleanup at the end
	cleanup()
}
