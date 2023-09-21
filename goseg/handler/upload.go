package handler

import (
	"context"
	"fmt"
	"goseg/config"
	"goseg/logger"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
)

var (
	UploadSessions []string
	uploadDir      = filepath.Join(config.BasePath, "uploads")
	tempDir        = filepath.Join(config.BasePath, "temp")
)

func init() {
	os.MkdirAll(uploadDir, 0755)
	os.MkdirAll(tempDir, 0755)
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	validSession := false
	session := vars["uploadSession"]
	for _, valid := range UploadSessions {
		if session == valid {
			validSession = true
			break
		}
	}
	if validSession {
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Unable to read uploaded file", http.StatusBadRequest)
			return
		}
		defer file.Close()
		filename := r.FormValue("dzfilename")
		chunkIndex := r.FormValue("dzchunkindex")
		totalChunks := r.FormValue("dztotalchunkcount")
		index, err := strconv.Atoi(chunkIndex)
		if err != nil {
			http.Error(w, "Invalid chunk index", http.StatusBadRequest)
			return
		}
		total, err := strconv.Atoi(totalChunks)
		if err != nil {
			http.Error(w, "Invalid total chunk count", http.StatusBadRequest)
			return
		}
		tempFilePath := filepath.Join(tempDir, fmt.Sprintf("%s-part-%d", filename, index))
		tempFile, err := os.Create(tempFilePath)
		if err != nil {
			http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
			return
		}
		defer tempFile.Close()
		io.Copy(tempFile, file)
		if allChunksReceived(filename, total) {
			err := combineChunks(filename, total)
			if err != nil {
				http.Error(w, "Failed to combine chunks", http.StatusInternalServerError)
				return
			}
		}
	} else {
		logger.Logger.Error(fmt.Sprintf("Invalid upload session request %v", session))
		http.Error(w, "Invalid upload session", http.StatusUnauthorized)
		return
	}
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
