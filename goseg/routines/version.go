package routines

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/handler"
	"groundseg/structs"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/slsa-framework/slsa-verifier/cli/slsa-verifier/verify"
	"go.uber.org/zap"
)

func CheckVersionLoop() {
	conf := config.Conf()
	var updateInterval int
	if conf.UpdateInterval < 60 {
		updateInterval = 60
	} else {
		updateInterval = conf.UpdateInterval
	}
	checkInterval := time.Duration(updateInterval) * time.Second
	ticker := time.NewTicker(checkInterval)
	releaseChannel := conf.UpdateBranch
	if conf.UpdateMode == "auto" {
		callUpdater(releaseChannel)
		for {
			select {
			case <-ticker.C:
				callUpdater(releaseChannel)
			}
		}
	}
}

func callUpdater(releaseChannel string) {
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
		zap.L().Error(fmt.Sprintf("Couldn't hash binary: %v", err))
		return
	}
	latestHash := latestVersion.Groundseg.Amd64Sha256
	if config.Architecture != "amd64" {
		latestHash = latestVersion.Groundseg.Arm64Sha256
	}
	if currentHash != latestHash {
		zap.L().Info("GroundSeg Binary update!")
		// updateBinary will likely restart the program, so
		// we don't have to care about the docker updates.
		updateBinary(releaseChannel, latestVersion)
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
	zap.L().Info(msg)
	// delete old instance of groundseg_new if it exists
	if _, err := os.Stat(filepath.Join(config.BasePath, "groundseg_new")); err == nil {
		// Remove the file
		zap.L().Info("Deleting old groundseg_new download")
		if err := os.Remove(filepath.Join(config.BasePath, "groundseg_new")); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to remove old instance of groundseg_new: %v", err))
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
	zap.L().Info(fmt.Sprintf("Downloading new GroundSeg binary from %v", url))
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to download new GroundSeg binary: %v", err))
		return
	}
	if resp.StatusCode != 200 {
		zap.L().Error(fmt.Sprintf("Couldn't download binary: %v", resp.StatusCode))
		return
	}
	defer resp.Body.Close()

	// Create a new file to save the downloaded content
	zap.L().Info("Creating groundseg_new")
	file, err := os.Create(filepath.Join(config.BasePath, "groundseg_new"))
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to save GroundSeg binary: %v", err))
		return
	}
	defer file.Close()
	zap.L().Info("Writing groundseg_new contents")
	// Write the contents from the HTTP response to the new file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to write contents: %v", err))
		return
	}
	// Chmod groundseg_new
	zap.L().Info("Modifying groundseg_new permissions")
	binaryPath := filepath.Join(config.BasePath, "groundseg_new")
	if err := os.Chmod(binaryPath, 0755); err != nil {
		zap.L().Error(fmt.Sprintf("Failed to write contents: %v", err))
		return
	}
	newVersionHash := versionInfo.Groundseg.Arm64Sha256
	if config.Architecture == "amd64" {
		newVersionHash = versionInfo.Groundseg.Amd64Sha256
	}
	newBinHash, err := config.GetSHA256(filepath.Join(config.BasePath, "groundseg_new"))
	if err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't get SHA for new binary: %v", err))
		return
	}
	if newVersionHash != newBinHash {
		zap.L().Error(fmt.Sprintf("New binary hash does not match downloaded file: remote %v / downloaded %v", newVersionHash, newBinHash))
		return
	}
	if !conf.DisableSlsa {
		zap.L().Info("Verifying SLSA provenance")
		if err := verifySlsaProvenance(
			versionInfo.Groundseg.SlsaURL,
			binaryPath,
			"git+https://github.com/Native-Planet/GroundSeg",
		); err != nil {
			zap.L().Error(fmt.Sprintf("SLSA verification failed: %v", err))
			return
		}
		zap.L().Info("SLSA verification successful")
	} else {
		zap.L().Warn("SLSA verification disabled by configuration")
	}
	// delete groundseg binary if exists
	zap.L().Info("Deleting old groundseg")
	if _, err := os.Stat(filepath.Join(config.BasePath, "groundseg")); err == nil {
		// Remove the file
		if err := os.Remove(filepath.Join(config.BasePath, "groundseg")); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to remove old instance of groundseg: %v", err))
			return
		}
	}
	// rename groundseg_new to groundseg
	zap.L().Info("Renaming groundseg_new to groundseg")
	oldPath := filepath.Join(config.BasePath, "groundseg_new")
	newPath := filepath.Join(config.BasePath, "groundseg")
	if err := os.Rename(oldPath, newPath); err != nil {
		zap.L().Error(fmt.Sprintf("Failed to rename groundseg_new to groundseg: %v", err))
		return
	}
	// re-disable bypass after one update
	if conf.DisableSlsa {
		if err := config.UpdateConf(map[string]interface{}{
			"disableSlsa": false,
		}); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't reset SLSA bypass config: %v", err))
		}
	}
	versionStr := "v" + strconv.Itoa(versionInfo.Groundseg.Major) + "." +
		strconv.Itoa(versionInfo.Groundseg.Minor) + "." +
		strconv.Itoa(versionInfo.Groundseg.Patch)
	binHash, err := getSha256(newPath)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't hash new binary: %v", err))
	}
	if err := config.UpdateConf(map[string]interface{}{
		"gsVersion": versionStr,
		"binHash":   binHash,
	}); err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't update config: %v", err))
	}
	// systemctl restart groundseg
	if config.DebugMode {
		zap.L().Debug("DebugMode detected. Skipping systemd command. Exiting istead..")
		os.Exit(0)
	} else {
		zap.L().Info("Restarting GroundSeg systemd service")
		cmd := exec.Command("systemctl", "restart", "groundseg")
		err := cmd.Run()
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to restart systemd service: %v", err))
			return
		}
	}
}

func verifySlsaProvenance(provenanceURL string, binaryPath string, sourceURI string) error {
	if _, err := rekorKey(); err != nil {
		return fmt.Errorf("failed to ensure Rekor key is available: %w", err)
	}
	provenanceFile, err := os.CreateTemp("", "provenance-*.intoto.jsonl")
	if err != nil {
		return fmt.Errorf("failed to create temp file for provenance: %v", err)
	}
	defer os.Remove(provenanceFile.Name())
	defer provenanceFile.Close()
	resp, err := http.Get(provenanceURL)
	if err != nil {
		return fmt.Errorf("failed to download provenance: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to download provenance, status: %v", resp.StatusCode)
	}
	_, err = io.Copy(provenanceFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write provenance file: %v", err)
	}
	verifyCmd := &verify.VerifyArtifactCommand{
		ProvenancePath:  provenanceFile.Name(),
		SourceURI:       sourceURI,
		PrintProvenance: false,
	}
	ctx := context.Background()
	trustedBuilder, err := verifyCmd.Exec(ctx, []string{binaryPath})
	if err != nil {
		return fmt.Errorf("SLSA verification failed: %v", err)
	}
	zap.L().Info(fmt.Sprintf("Verified by trusted builder: %v", trustedBuilder))
	return nil
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
	zap.L().Info(fmt.Sprintf("update docker called: Current: %v , Latest %v", currentVersion, latestVersion))
	zap.L().Info(fmt.Sprintf(
		"New version available in %s channel! Current: %+v, Latest: %+v\n",
		release, currentVersion, latestVersion,
	))
	conf := config.Conf()
	statuses, err := docker.GetShipStatus(conf.Piers)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't get ship statuses: %v", err))
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
								_, err := docker.StartContainer(pier, "vere")
								if err != nil {
									zap.L().Error(fmt.Sprintf("Failed to start %s after vere update: %v", err))
								}
								continue
							}
							// after starting (or not starting) the container,
							// check if it wants a chop
							urbConf := config.UrbitConf(pier)
							if urbConf.ChopOnUpgrade == true {
								go handler.ChopPier(pier, urbConf)
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
								_, err := docker.StartContainer(pier, "vere")
								if err != nil {
									zap.L().Error(fmt.Sprintf("Failed to start %s after vere update: %v", err))
								}
								continue
							}
							// after starting (or not starting) the container,
							// check if it wants a chop
							urbConf := config.UrbitConf(pier)
							if urbConf.ChopOnUpgrade == true {
								go handler.ChopPier(pier, urbConf)
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
