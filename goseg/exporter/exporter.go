package exporter

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"goseg/docker"
	"goseg/logger"
	"goseg/structs"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

var (
	whitelist = make(map[string]structs.WsTokenStruct)
	exportMu  sync.Mutex
)

func WhitelistContainer(container string, token structs.WsTokenStruct) error {
	exportMu.Lock()
	defer exportMu.Unlock()
	whitelist[container] = token
	logger.Logger.Info(fmt.Sprintf("Whitelisted %v for export", container))
	return nil
}

func RemoveContainerFromWhitelist(container string) error {
	exportMu.Lock()
	defer exportMu.Unlock()
	delete(whitelist, container)
	logger.Logger.Info(fmt.Sprintf("Removed %v from export whitelist", container))
	return nil
}

func ExportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	// Handle pre-flight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	exportError := func(err error) {
		logger.Logger.Error(fmt.Sprintf("Unable to export: %v", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	cleanup := func(c string) {
		RemoveContainerFromWhitelist(c)
		docker.UTransBus <- structs.UrbitTransition{Patp: c, Type: "exportShip", Event: ""}
		docker.UTransBus <- structs.UrbitTransition{Patp: c, Type: "shipCompressed", Value: 0}
	}
	// check if container is whitelisted
	vars := mux.Vars(r)
	container := vars["container"]
	exportMu.Lock()
	whitelistToken, exists := whitelist[container]
	exportMu.Unlock()
	if !exists {
		err := fmt.Errorf("Container %v is not in whitelist!", container)
		logger.Logger.Error(fmt.Sprintf("Rejecting Export request: %v", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// check if token match
	var tokenData structs.WsTokenStruct

	err := json.NewDecoder(r.Body).Decode(&tokenData)
	if err != nil {
		err := fmt.Errorf("Export failed to decode token: %v", err)
		logger.Logger.Error(fmt.Sprintf("Rejecting Export request: %v", err))
		docker.UTransBus <- structs.UrbitTransition{Patp: container, Type: "exportShip", Event: ""}
		docker.UTransBus <- structs.UrbitTransition{Patp: container, Type: "shipCompressed", Value: 0}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if whitelistToken != tokenData {
		err := fmt.Errorf("Token for exporting %v is not valid", container)
		err = fmt.Errorf("Export failed to decode token: %v", err)
		logger.Logger.Error(fmt.Sprintf("Rejecting Export request: %v", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		docker.UTransBus <- structs.UrbitTransition{Patp: container, Type: "exportShip", Event: ""}
		docker.UTransBus <- structs.UrbitTransition{Patp: container, Type: "shipCompressed", Value: 0}
		return
	}
	// session is now valid. Defer cleanup
	defer cleanup(container)
	// make sure container is stopped
	statusTicker := time.NewTicker(500 * time.Millisecond)
loopLabel:
	for {
		select {
		case <-statusTicker.C:
			pierStatus, err := docker.GetShipStatus([]string{container})
			if err != nil {
				exportError(err)
				return
			}
			status, exists := pierStatus[container]
			if !exists {
				exportError(fmt.Errorf("Unable to export nonexistent container: %v", container))
			}
			if strings.Contains(status, "Exited") {
				break loopLabel
			}
		}
	}
	// compress volume transition compressed %
	volumeDirectory := "/var/lib/docker/volumes"
	var memoryFile bytes.Buffer
	filePath := filepath.Join(volumeDirectory, container, "_data")
	// Create new zip archive in memory
	zipWriter := zip.NewWriter(&memoryFile)
	// Walk through the file path
	var walkErr error
	// Count total files
	totalFiles := 0
	filepath.Walk(filePath, func(_ string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() {
			totalFiles++
		}
		return nil
	})
	completedFiles := 0
	filepath.Walk(filePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			walkErr = err
			return err
		}
		arcDir := strings.TrimPrefix(path, filePath+"/")
		if filepath.Base(path) != "conn.sock" {
			f, _ := zipWriter.Create(arcDir)
			fileReader, _ := os.Open(path)
			defer fileReader.Close()
			io.Copy(f, fileReader)
			completedFiles++
		}
		progress := int(float64(completedFiles) / float64(totalFiles) * 100)
		docker.UTransBus <- structs.UrbitTransition{Patp: container, Type: "shipCompressed", Value: progress}
		return nil
	})
	if walkErr != nil {
		exportError(walkErr)
		return
	}
	// Close the zip archive
	zipWriter.Close()
	/*
		ticker := time.NewTicker(500 * time.Millisecond)
		count := 0
		for {
			if count > 100 {
				break
			}
			select {
			case <-ticker.C:
				count = count + 5
				docker.UTransBus <- structs.UrbitTransition{Patp: container, Type: "shipCompressed", Value: count}
			}
		}
	*/
	// send file
	reader := bytes.NewReader(memoryFile.Bytes())
	http.ServeContent(w, r, container+".zip", time.Now(), reader)
}
