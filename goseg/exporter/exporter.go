package exporter

import (
	"archive/zip"
	"bufio"
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

const (
	maxArchiveSize   = 10 * 1024 * 1024 * 1024 // 10GB max archive size
	bufferSize       = 1024 * 1024             // 1MB buffer for writing
	progressInterval = 100                     // Update progress every 100 files
)

var (
	whitelist = make(map[string]structs.WsTokenStruct)
	exportMu  sync.Mutex
)

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

	ctx := r.Context()

	exportError := func(w http.ResponseWriter, err error) {
		zap.L().Error(fmt.Sprintf("Unable to export: %v", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	cleanup := func(patp, containerName, exportTrans, compressedTrans string) {
		RemoveContainerFromWhitelist(containerName)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: exportTrans, Event: ""}
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: compressedTrans, Value: 0}
	}

	// check if container is whitelisted
	vars := mux.Vars(r)
	container := vars["container"]

	exportMu.Lock()
	whitelistToken, exists := whitelist[container]
	exportMu.Unlock()

	if !exists {
		exportError(w, fmt.Errorf("container %v is not in whitelist", container))
		return
	}

	// get patp if minio container
	patp := strings.TrimPrefix(container, "minio_")
	exportTrans := "exportShip"
	compressedTrans := "shipCompressed"
	isMinIO := strings.Contains(container, "minio_")
	if isMinIO {
		exportTrans = "exportBucket"
		compressedTrans = "bucketCompressed"
	}

	// validate token
	var tokenData structs.WsTokenStruct
	if err := json.NewDecoder(r.Body).Decode(&tokenData); err != nil {
		exportError(w, fmt.Errorf("failed to decode token: %w", err))
		cleanup(patp, container, exportTrans, compressedTrans)
		return
	}

	if whitelistToken != tokenData {
		exportError(w, fmt.Errorf("invalid token for container %v", container))
		cleanup(patp, container, exportTrans, compressedTrans)
		return
	}

	// wait for container to stop if not MinIO
	if !isMinIO {
		statusTicker := time.NewTicker(500 * time.Millisecond)
		defer statusTicker.Stop()

	waitLoop:
		for {
			select {
			case <-ctx.Done():
				exportError(w, fmt.Errorf("client disconnected while waiting for container to stop"))
				cleanup(patp, container, exportTrans, compressedTrans)
				return
			case <-statusTicker.C:
				pierStatus, err := docker.GetShipStatus([]string{container})
				if err != nil {
					exportError(w, err)
					cleanup(patp, container, exportTrans, compressedTrans)
					return
				}

				status, exists := pierStatus[container]
				if !exists {
					exportError(w, fmt.Errorf("container %v does not exist", container))
					cleanup(patp, container, exportTrans, compressedTrans)
					return
				}

				if strings.Contains(status, "Exited") {
					break waitLoop
				}
			}
		}
	}

	// get file path
	volumeDirectory := defaults.DockerData("volumes")
	filePath := filepath.Join(volumeDirectory, container, "_data")

	shipConf := config.UrbitConf(container)
	if customLoc, ok := shipConf.CustomPierLocation.(string); ok {
		filePath = customLoc
	}

	// check total size before starting
	var totalSize int64
	if err := filepath.Walk(filePath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.HasSuffix(info.Name(), ".sock") {
			totalSize += info.Size()
		}
		return nil
	}); err != nil {
		exportError(w, fmt.Errorf("failed to calculate total size: %w", err))
		cleanup(patp, container, exportTrans, compressedTrans)
		return
	}

	if totalSize > maxArchiveSize {
		exportError(w, fmt.Errorf("total size %d exceeds maximum allowed size %d", totalSize, maxArchiveSize))
		cleanup(patp, container, exportTrans, compressedTrans)
		return
	}

	// count total files for progress reporting
	totalFiles := 0
	if err := filepath.Walk(filePath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.HasSuffix(info.Name(), ".sock") {
			totalFiles++
		}
		return nil
	}); err != nil {
		exportError(w, fmt.Errorf("failed to count files: %w", err))
		cleanup(patp, container, exportTrans, compressedTrans)
		return
	}

	// set response headers
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", container))

	// create buffered writer
	bufferedWriter := bufio.NewWriterSize(w, bufferSize)
	zipWriter := zip.NewWriter(bufferedWriter)

	// ensure everything is properly closed
	defer func() {
		if err := zipWriter.Close(); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to close zip writer: %v", err))
		}
		if err := bufferedWriter.Flush(); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to flush buffer: %v", err))
		}
		cleanup(patp, container, exportTrans, compressedTrans)
	}()

	completedFiles := 0
	err := filepath.Walk(filePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk error at %s: %w", path, err)
		}

		// check for context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("client disconnected during export")
		default:
		}

		if info.IsDir() || strings.HasSuffix(info.Name(), ".sock") {
			return nil
		}

		// create relative path for zip entry
		arcDir := strings.TrimPrefix(path, filePath+"/")

		// create new file in zip
		f, err := zipWriter.Create(arcDir)
		if err != nil {
			return fmt.Errorf("failed to create zip entry for %s: %w", arcDir, err)
		}

		// open and copy file
		fileReader, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer fileReader.Close()

		// copy file contents
		if _, err := io.Copy(f, fileReader); err != nil {
			return fmt.Errorf("failed to copy %s: %w", path, err)
		}

		completedFiles++

		// update progress periodically
		if completedFiles%progressInterval == 0 {
			progress := int(float64(completedFiles) / float64(totalFiles) * 100)
			docker.UTransBus <- structs.UrbitTransition{
				Patp:  patp,
				Type:  compressedTrans,
				Value: progress,
			}
		}

		return nil
	})

	if err != nil {
		// log error but can't send error response as we've already started sending file data
		zap.L().Error(fmt.Sprintf("Error during export: %v", err))
		return
	}

	// send final progress update
	docker.UTransBus <- structs.UrbitTransition{
		Patp:  patp,
		Type:  compressedTrans,
		Value: 100,
	}
}
