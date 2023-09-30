package routines

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"goseg/config"
	"goseg/docker"
	"goseg/logger"
	"goseg/structs"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func CheckVersionLoop() {
	conf := config.Conf()
	checkInterval := time.Duration(conf.UpdateInterval) * time.Second
	ticker := time.NewTicker(checkInterval)
	releaseChannel := conf.UpdateBranch
	if conf.UpdateMode == "auto" {
		for {
			select {
			case <-ticker.C:
				// Get latest information
				latestVersion, _ := config.CheckVersion()
				currentChannelVersion := config.VersionInfo
				latestChannelVersion := latestVersion
				// check docker updates
				if latestChannelVersion != currentChannelVersion {
					config.VersionInfo = latestVersion
					updateDocker(releaseChannel, currentChannelVersion, latestChannelVersion)
				}
				// Check for gs binary updates based on hash
				binPath := filepath.Join(config.BasePath, "groundseg")
				currentHash, err := getSha256(binPath)
				if err != nil {
					logger.Logger.Error(fmt.Sprintf("Couldn't hash binary: %v", err))
					continue
				}
				latestHash := latestVersion.Groundseg.Amd64Sha256
				if config.Architecture != "amd64" {
					latestHash = latestVersion.Groundseg.Arm64Sha256
				}
				if currentHash != latestHash {
					logger.Logger.Info("GroundSeg Binary update!")
					// updateBinary will likely restart the program, so
					// we don't have to care about the docker updates.
					updateBinary(releaseChannel, latestVersion)
				}
			}
		}
	}
}

func updateBinary(branch string, versionInfo structs.Channel) {
	// get config
	conf := config.Conf()
	var displayedBranch string
	if branch != "latest" {
		displayedBranch = fmt.Sprintf("-%v", branch)
	}
	msg := fmt.Sprintf(
		"A GroundSeg binary update detected! Current Version: %v%v , Latest Version v%v.%v.%v%v",
		conf.GsVersion, displayedBranch,
		versionInfo.Groundseg.Major, versionInfo.Groundseg.Minor,
		versionInfo.Groundseg.Patch, displayedBranch,
	)
	logger.Logger.Info(msg)
	// delete old instance of groundseg_new if it exists
	if _, err := os.Stat(filepath.Join(config.BasePath, "groundseg_new")); err == nil {
		// Remove the file
		logger.Logger.Info("Deleting old groundseg_new download")
		if err := os.Remove(filepath.Join(config.BasePath, "groundseg_new")); err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to remove old instance of groundseg_new: %v", err))
			return
		}
	}
	// download new binary, name it groundseg_new
	url := versionInfo.Groundseg.Arm64URL
	if config.Architecture == "amd64" {
		url = versionInfo.Groundseg.Amd64URL
	}
	// Create a new HTTP GET request
	resp, err := http.Get(url)
	logger.Logger.Info(fmt.Sprintf("Downloading new GroundSeg binary from %v", url))
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to download new GroundSeg binary: %v", err))
		return
	}
	defer resp.Body.Close()

	// Create a new file to save the downloaded content
	logger.Logger.Info("Creating groundseg_new")
	file, err := os.Create(filepath.Join(config.BasePath, "groundseg_new"))
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to save GroundSeg binary: %v", err))
		return
	}
	defer file.Close()
	logger.Logger.Info("Writing groundseg_new contents")
	// Write the contents from the HTTP response to the new file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to write contents: %v", err))
		return
	}
	// Chmod groundseg_new
	logger.Logger.Info("Modifying groundseg_new permissions")
	if err := os.Chmod(filepath.Join(config.BasePath, "groundseg_new"), 0755); err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to write contents: %v", err))
		return
	}
	// delete groundseg binary if exists
	logger.Logger.Info("Deleting old groundseg")
	if _, err := os.Stat(filepath.Join(config.BasePath, "groundseg")); err == nil {
		// Remove the file
		if err := os.Remove(filepath.Join(config.BasePath, "groundseg")); err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to remove old instance of groundseg: %v", err))
			return
		}
	}
	// rename groundseg_new to groundseg
	logger.Logger.Info("Renaming groundseg_new to groundseg")
	oldPath := filepath.Join(config.BasePath, "groundseg_new")
	newPath := filepath.Join(config.BasePath, "groundseg")
	if err := os.Rename(oldPath, newPath); err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to rename groundseg_new to groundseg: %v", err))
		return
	}
	versionStr := "v" + strconv.Itoa(versionInfo.Groundseg.Major) + "." +
		strconv.Itoa(versionInfo.Groundseg.Minor) + "." +
		strconv.Itoa(versionInfo.Groundseg.Patch)
	binHash, err := getSha256(newPath)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Couldn't hash new binary: %v", err))
	}
	if err := config.UpdateConf(map[string]interface{}{
		"gsVersion": versionStr,
		"binHash":   binHash,
	}); err != nil {
		logger.Logger.Error(fmt.Sprintf("Couldn't update config: %v", err))
	}
	// systemctl restart groundseg
	if config.DebugMode {
		logger.Logger.Debug("DebugMode detected. Skipping systemd command. Exiting istead..")
		os.Exit(0)
	} else {
		logger.Logger.Info("Restarting GroundSeg systemd service")
		cmd := exec.Command("systemctl", "restart", "groundseg")
		err := cmd.Run()
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to restart systemd service: %v", err))
			return
		}
	}
}

func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

func updateDocker(release string, currentVersion structs.Channel, latestVersion structs.Channel) {
	logger.Logger.Info(fmt.Sprintf("update docker called: Current: %v , Latest %v", currentVersion, latestVersion))
	logger.Logger.Info(fmt.Sprintf(
		"New version available in %s channel! Current: %+v, Latest: %+v\n",
		release, currentVersion, latestVersion,
	))
	conf := config.Conf()
	statuses, err := docker.GetShipStatus(conf.Piers)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Couldn't get ship statuses: %v", err))
		return
	}
	valCurrent := reflect.ValueOf(currentVersion)
	valLatest := reflect.ValueOf(latestVersion)

	typeOfVersion := valCurrent.Type()

	for i := 0; i < valCurrent.NumField(); i++ {
		sw := typeOfVersion.Field(i).Name
		if sw != "groundseg" {
			currentDetail := valCurrent.Field(i).Interface().(structs.VersionDetails)
			latestDetail := valLatest.Field(i).Interface().(structs.VersionDetails)
			if config.Architecture == "amd64" {
				if latestDetail.Amd64Sha256 != currentDetail.Amd64Sha256 {
					if contains([]string{"netdata", "wireguard", "miniomc"}, sw) {
						docker.StartContainer(sw, sw)
					} else if sw == "vere" {
						for pier, status := range statuses {
							isRunning := (status == "Up" || strings.HasPrefix(status, "Up "))
							if isRunning {
								docker.StartContainer(pier, "vere")
							}
						}
					} else if sw == "minio" {
						for pier, status := range statuses {
							isRunning := (status == "Up" || strings.HasPrefix(status, "Up "))
							if isRunning {
								docker.StartContainer("minio_"+pier, "minio")
							}
						}
					}
				}
			} else {
				if latestDetail.Arm64Sha256 != currentDetail.Arm64Sha256 {
					if contains([]string{"netdata", "wireguard", "miniomc"}, sw) {
						docker.StartContainer(sw, sw)
					} else if sw == "vere" {
						for pier, status := range statuses {
							isRunning := (status == "Up" || strings.HasPrefix(status, "Up "))
							if isRunning {
								docker.StartContainer(pier, "vere")
							}
						}
					} else if sw == "minio" {
						for pier, status := range statuses {
							isRunning := (status == "Up" || strings.HasPrefix(status, "Up "))
							if isRunning {
								docker.StartContainer("minio_"+pier, "minio")
							}
						}
					}
				}
			}
		}
	}
}

func getSha256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	hashValue := hex.EncodeToString(hasher.Sum(nil))
	return hashValue, nil
}
