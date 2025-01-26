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
	// start timer
	start := time.Now()

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := r.Context()
	vars := mux.Vars(r)
	container := vars["container"]

	// Shared error handler
	exportError := func(err error, status int) {
		zap.L().Error("Export failed: " + err.Error())
		http.Error(w, err.Error(), status)
	}

	// Check container whitelist
	exportMu.Lock()
	whitelistToken, exists := whitelist[container]
	exportMu.Unlock()

	if !exists {
		exportError(fmt.Errorf("container %v not whitelisted", container), http.StatusForbidden)
		return
	}

	// Validate authorization token
	var tokenData structs.WsTokenStruct
	if err := json.NewDecoder(r.Body).Decode(&tokenData); err != nil {
		exportError(fmt.Errorf("invalid token format"), http.StatusBadRequest)
		return
	}

	if whitelistToken != tokenData {
		exportError(fmt.Errorf("invalid authentication token"), http.StatusUnauthorized)
		return
	}

	// Configure paths and cleanup
	patp := strings.TrimPrefix(container, "minio_")
	isMinIO := strings.Contains(container, "minio_")
	filePath := getFilePath(container) // Extract path resolution to helper function

	// Configure streaming response
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", container))

	// Setup streaming pipeline
	zw := zip.NewWriter(w)
	defer func() {
		if closeErr := zw.Close(); closeErr != nil {
			zap.L().Error("Failed closing zip writer: " + closeErr.Error())
		}
		cleanupResources(container, patp, isMinIO)
	}()

	// Stream files to zip
	err := filepath.Walk(filePath, func(path string, info os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return fmt.Errorf("client disconnected")
		default:
		}

		if err != nil || info.IsDir() || strings.HasSuffix(info.Name(), ".sock") {
			return err
		}

		relPath, _ := filepath.Rel(filePath, path)
		entry, err := zw.Create(filepath.ToSlash(relPath))
		if err != nil {
			return fmt.Errorf("failed creating zip entry: %w", err)
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed opening file: %w", err)
		}
		defer file.Close()

		if _, err := io.Copy(entry, file); err != nil {
			return fmt.Errorf("failed streaming file: %w", err)
		}

		return nil
	})

	if err != nil {
		zap.L().Error("Streaming interrupted: " + err.Error())
	}

	// log time taken
	zap.L().Info(fmt.Sprintf("Exported %v in %v", container, time.Since(start)))
}

// Helper functions
func getFilePath(container string) string {
	if customLoc, ok := config.UrbitConf(container).CustomPierLocation.(string); ok {
		return customLoc
	}
	return filepath.Join(defaults.DockerData("volumes"), container, "_data")
}

func cleanupResources(container, patp string, isMinIO bool) {
	RemoveContainerFromWhitelist(container)
	transType := "exportShip"
	if isMinIO {
		transType = "exportBucket"
	}
	docker.UTransBus <- structs.UrbitTransition{
		Patp: patp,
		Type: transType,
	}
}
