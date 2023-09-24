package importer

import (
	"context"
	"errors"
	"fmt"
	"goseg/config"
	"goseg/logger"
	"goseg/structs"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
)

var (
	uploadSessions = make(map[string]structs.WsTokenStruct)
	uploadDir      = filepath.Join(config.BasePath, "uploads")
	tempDir        = filepath.Join(config.BasePath, "temp")
	uploadMu       sync.Mutex
)

func init() {
	os.MkdirAll(uploadDir, 0755)
	os.MkdirAll(tempDir, 0755)
}

func SetUploadSession(uploadPayload structs.WsUploadPayload) error {
	uploadMu.Lock()
	defer uploadMu.Unlock()
	endpoint := uploadPayload.Payload.Endpoint
	token := uploadPayload.Token
	// Check if endpoint exists in uploadSessions
	if existingToken, exists := uploadSessions[endpoint]; exists {
		if existingToken == token {
			// If tokens are the same, do nothing
			return nil
		}
		// Tokens are different, return error
		return errors.New("token mismatch")
	}
	// If endpoint not in uploadSessions, add it
	uploadSessions[endpoint] = token
	return nil
}

func ClearUploadSession(session string) {
	uploadMu.Lock()
	defer uploadMu.Unlock()
	delete(uploadSessions, session)
}

func VerifySession(session string) bool {
	_, exists := uploadSessions[session]
	// todo: check token for user agent info
	return exists
}

func HTTPUploadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cache-Control, X-Requested-With")

	// Handle pre-flight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	session := vars["uploadSession"]
	validSession := VerifySession(session)
	patp := vars["patp"]

	if validSession {
		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status": "failure", "message": "Unable to read uploaded file"}`))
			return
		}
		defer file.Close()

		filename := fileHeader.Filename
		chunkIndex := r.FormValue("dzchunkindex")
		totalChunks := r.FormValue("dztotalchunkcount")
		index, err := strconv.Atoi(chunkIndex)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status": "failure", "message": "Invalid chunk index"}`))
			return
		}
		total, err := strconv.Atoi(totalChunks)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status": "failure", "message": "Invalid total chunk count"}`))
			return
		}
		tempFilePath := filepath.Join(tempDir, fmt.Sprintf("%s-part-%d", filename, index))
		tempFile, err := os.Create(tempFilePath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "failure", "message": "Failed to create temp file"}`))
			return
		}
		defer tempFile.Close()
		io.Copy(tempFile, file)
		if allChunksReceived(filename, total) {
			if err := combineChunks(filename, total); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"status": "failure", "message": "Failed to combine chunks"}`))
				return
			}
			ClearUploadSession(session)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success", "message": "Upload successful"}`))
			go configureUploadedPier(filename, patp)
			return
		}
	} else {
		logger.Logger.Error(fmt.Sprintf("Invalid upload session request %v", session))
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "failure", "message": "Invalid upload session"}`))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success", "message": "Chunk received successfully"}`))
	return
}

func allChunksReceived(filename string, total int) bool {
	for i := 0; i < total; i++ {
		if _, err := os.Stat(filepath.Join(tempDir, fmt.Sprintf("%s-part-%d", filename, i))); os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func combineChunks(filename string, total int) error {

	destFile, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		return err
	}
	defer destFile.Close()
	for i := 0; i < total; i++ {
		partFilePath := filepath.Join(tempDir, fmt.Sprintf("%s-part-%d", filename, i))
		partFile, err := os.Open(partFilePath)
		if err != nil {
			return err
		}
		io.Copy(destFile, partFile)
		partFile.Close()
		os.Remove(partFilePath)
	}
	return nil
}

func restructureDirectory(patp string) error {
	// get docker volume path for patp
	volDir := ""
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	volumes, err := cli.VolumeList(context.Background(), volume.ListOptions{})
	if err != nil {
		return err
	}
	for _, vol := range volumes.Volumes {
		if vol.Name == patp {
			volDir = vol.Mountpoint
			break
		}
	}
	if volDir == "" {
		return fmt.Errorf("No docker volume for %d!", patp)
	}
	dataDir := filepath.Join(volDir, "_data")
	logger.Logger.Debug("%v volume path: %v", patp, dataDir)
	// find .urb
	var urbLoc []string
	_ = filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && filepath.Base(path) == ".urb" && !strings.Contains(path, "__MACOSX") {
			urbLoc = append(urbLoc, filepath.Dir(path))
		}
		return nil
	})
	// there can only be one
	if len(urbLoc) > 1 {
		return fmt.Errorf("%v ships detected in pier directory", len(urbLoc))
	}
	if len(urbLoc) < 1 {
		return fmt.Errorf("No ship found in pier directory")
	}
	logger.Logger.Debug(fmt.Sprintf(".urb subdirectory in %v", urbLoc[0]))
	pierDir := filepath.Join(dataDir, patp)
	tempDir := filepath.Join(dataDir, "temp_dir")
	unusedDir := filepath.Join(dataDir, "unused")
	// move it into the right place
	if filepath.Join(pierDir, ".urb") != filepath.Join(urbLoc[0], ".urb") {
		logger.Logger.Info(".urb location incorrect! Restructuring directory structure")
		logger.Logger.Debug(fmt.Sprintf(".urb found in %v", urbLoc[0]))
		logger.Logger.Debug(fmt.Sprintf("Moving to %v", tempDir))
		if dataDir == urbLoc[0] { // .urb in root
			_ = os.MkdirAll(tempDir, 0755)
			items, _ := ioutil.ReadDir(urbLoc[0])
			for _, item := range items {
				if item.Name() != patp {
					os.Rename(filepath.Join(urbLoc[0], item.Name()), filepath.Join(tempDir, item.Name()))
				}
			}
		} else {
			os.Rename(urbLoc[0], tempDir)
		}
		unused := []string{}
		dirs, _ := ioutil.ReadDir(dataDir)
		for _, dir := range dirs {
			dirName := dir.Name()
			if dirName != "temp_dir" && dirName != "unused" {
				unused = append(unused, dirName)
			}
		}
		if len(unused) > 0 {
			_ = os.MkdirAll(unusedDir, 0755)
			for _, u := range unused {
				os.Rename(filepath.Join(dataDir, u), filepath.Join(unusedDir, u))
			}
		}
		os.Rename(tempDir, pierDir)
		logger.Logger.Info(fmt.Sprintf("%v restructuring done", patp))
	} else {
		logger.Logger.Debug("No restructuring needed")
	}
	return nil
}

func configureUploadedPier(filename, patp string) {
	// temp
	logger.Logger.Warn(fmt.Sprintf("configure uploaded pier called: %s", patp))

	// extract file to volume directory
	// run restructure
	// configs
	// StartContainer
	// errorCleanup
}
