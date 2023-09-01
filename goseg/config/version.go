package config

import (
	"encoding/json"
	"fmt"
	"goseg/defaults"
	"goseg/structs"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var (
	VersionServerReady = false
	VersionInfo        structs.Channel
)

// check the version server and return struct
func CheckVersion() (structs.Channel, bool) {
	versMutex.Lock()
	defer versMutex.Unlock()
	conf := Conf()
	releaseChannel := conf.UpdateBranch
	const retries = 10
	const delay = time.Second
	url := globalConfig.UpdateUrl
	var fetchedVersion structs.Version
	for i := 0; i < retries; i++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
		}
		userAgent := "NativePlanet.GroundSeg-" + conf.GsVersion
		req.Header.Set("User-Agent", userAgent)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errmsg := fmt.Sprintf("Unable to connect to update server: %v", err)
			Logger.Warn(errmsg)
			if i < retries-1 {
				time.Sleep(delay)
				continue
			} else {
				VersionServerReady = false
				return VersionInfo, false
			}
		}
		// read the body bytes
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			errmsg := fmt.Sprintf("Error reading version info: %v", err)
			Logger.Warn(errmsg)
			if i < retries-1 {
				time.Sleep(delay)
				continue
			} else {
				VersionServerReady = false
				return VersionInfo, false
			}
		}
		// unmarshal values into Version struct
		err = json.Unmarshal(body, &fetchedVersion)
		if err != nil {
			errmsg := fmt.Sprintf("Error unmarshalling JSON: %v", err)
			Logger.Warn(errmsg)
			if i < retries-1 {
				time.Sleep(delay)
				continue
			} else {
				VersionServerReady = false
				return VersionInfo, false
			}
		}
		VersionInfo = fetchedVersion.Groundseg[releaseChannel]
		// debug: re-marshal and write the entire fetched version to disk
		confPath := filepath.Join(BasePath, "settings", "version_info.json")
		file, err := os.Create(confPath)
		if err != nil {
			errmsg := fmt.Sprintf("Failed to create file: %v", err)
			Logger.Error(errmsg)
			VersionServerReady = false
			return VersionInfo, false
		}
		defer file.Close()
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "    ")
		if err := encoder.Encode(&fetchedVersion); err != nil {
			errmsg := fmt.Sprintf("Failed to write JSON: %v", err)
			Logger.Error(errmsg)
		}
		VersionServerReady = true
		return VersionInfo, true
	}
	VersionServerReady = false
	return VersionInfo, false
}

func CheckVersionLoop() {
	ticker := time.NewTicker(checkInterval)
	conf := Conf()
	releaseChannel := conf.UpdateBranch
	for {
		select {
		case <-ticker.C:
			// Get latest information
			latestVersion, _ := CheckVersion()

			// Check for gs binary updates based on hash
			currentHash := conf.BinHash
			latestHash := latestVersion.Groundseg.Amd64Sha256
			if Architecture != "amd64" {
				latestHash = latestVersion.Groundseg.Arm64Sha256
			}
			if currentHash != latestHash {
				Logger.Info("GroundSeg Binary update!")
				// updateBinary will likely restart the program, so
				// we don't have to care about the docker updates.
				updateBinary(releaseChannel, latestVersion)
			} else {
				// check docker updates
				currentChannelVersion := VersionInfo
				latestChannelVersion := latestVersion
				if latestChannelVersion != currentChannelVersion {
					VersionInfo = latestVersion
					updateDocker(releaseChannel, currentChannelVersion, latestChannelVersion)
				}
			}
		}
	}
}

func updateBinary(branch string, versionInfo structs.Channel) {
	// get config
	conf := Conf()
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
	Logger.Info(msg)
	// delete old instance of groundseg_new if it exists
	if _, err := os.Stat(filepath.Join(BasePath, "groundseg_new")); err == nil {
		// Remove the file
		Logger.Info("Deleting old groundseg_new download")
		if err := os.Remove(filepath.Join(BasePath, "groundseg_new")); err != nil {
			Logger.Error(fmt.Sprintf("Failed to remove old instance of groundseg_new: %v", err))
			return
		}
	}
	// download new binary, name it groundseg_new
	url := versionInfo.Groundseg.Arm64URL
	if Architecture == "amd64" {
		url = versionInfo.Groundseg.Amd64URL
	}
	// Create a new HTTP GET request
	resp, err := http.Get(url)
	Logger.Info(fmt.Sprintf("Downloading new GroundSeg binary from %v", url))
	if err != nil {
		Logger.Error(fmt.Sprintf("Failed to download new GroundSeg binary: %v", err))
		return
	}
	defer resp.Body.Close()

	// Create a new file to save the downloaded content
	Logger.Info("Creating groundseg_new")
	file, err := os.Create(filepath.Join(BasePath, "groundseg_new"))
	if err != nil {
		Logger.Error(fmt.Sprintf("Failed to save GroundSeg binary: %v", err))
		return
	}
	defer file.Close()
	Logger.Info("Writing groundseg_new contents")
	// Write the contents from the HTTP response to the new file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		Logger.Error(fmt.Sprintf("Failed to write contents: %v", err))
		return
	}
	// Chmod groundseg_new
	Logger.Info("Modifying groundseg_new permissions")
	if err := os.Chmod(filepath.Join(BasePath, "groundseg_new"), 0755); err != nil {
		Logger.Error(fmt.Sprintf("Failed to write contents: %v", err))
		return
	}
	// delete groundseg binary if exists
	Logger.Info("Deleting old groundseg")
	if _, err := os.Stat(filepath.Join(BasePath, "groundseg")); err == nil {
		// Remove the file
		if err := os.Remove(filepath.Join(BasePath, "groundseg")); err != nil {
			Logger.Error(fmt.Sprintf("Failed to remove old instance of groundseg: %v", err))
			return
		}
	}
	// rename groundseg_new to groundseg
	Logger.Info("Renaming groundseg_new to groundseg")
	oldPath := filepath.Join(BasePath, "groundseg_new")
	newPath := filepath.Join(BasePath, "groundseg")
	if err := os.Rename(oldPath, newPath); err != nil {
		Logger.Error(fmt.Sprintf("Failed to rename groundseg_new to groundseg: %v", err))
		return
	}
	// systemctl restart groundseg
	if DebugMode {
		Logger.Info("DebugMode detected. Skipping systemd command. Exiting istead..")
		os.Exit(0)
	} else {
		Logger.Info("Restarting GroundSeg systemd service")
		cmd := exec.Command("systemctl", "restart", "groundseg")
		err := cmd.Run()
		if err != nil {
			Logger.Error(fmt.Sprintf("Failed to restart systemd service: %v", err))
			return
		}
	}
}

func updateDocker(release string, currentVersion structs.Channel, latestVersion structs.Channel) {
	Logger.Info(fmt.Sprintf("update docker called: Current: %v , Latest %v", currentVersion, latestVersion))
	Logger.Info(fmt.Sprintf(
		"New version available in %s channel! Current: %+v, Latest: %+v\n",
		release, currentVersion, latestVersion,
	))
	// check individual images
	// update persistent
	// restart affected containers
}

// write the defaults.VersionInfo value to disk
func CreateDefaultVersion() error {
	var versionInfo structs.Version
	err := json.Unmarshal([]byte(defaults.DefaultVersionText), &versionInfo)
	if err != nil {
		return err
	}
	prettyJSON, err := json.MarshalIndent(versionInfo, "", "    ")
	if err != nil {
		return err
	}
	filePath := filepath.Join(BasePath, "settings", "version_info.json")
	err = ioutil.WriteFile(filePath, prettyJSON, 0644)
	if err != nil {
		return err
	}
	return nil
}

// return the existing local version info or create default
func LocalVersion() structs.Version {
	confPath := filepath.Join(BasePath, "settings", "version_info.json")
	_, err := os.Open(confPath)
	if err != nil {
		// create a default if it doesn't exist
		err = CreateDefaultVersion()
		if err != nil {
			// panic if we can't create it
			errmsg := fmt.Sprintf("Unable to write version info! %v", err)
			Logger.Error(errmsg)
			panic(errmsg)
		}
	}
	file, err := ioutil.ReadFile(confPath)
	if err != nil {
		errmsg := fmt.Sprintf("Unable to load version info: %v", err)
		panic(errmsg)
	}
	var versionStruct structs.Version
	if err := json.Unmarshal(file, &versionStruct); err != nil {
		errmsg := fmt.Sprintf("Error decoding version JSON: %v", err)
		panic(errmsg)
	}
	return versionStruct
}
