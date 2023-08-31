package config

import (
	"encoding/json"
	"fmt"
	"goseg/defaults"
	"goseg/structs"
	"io/ioutil"
	"net/http"
	"os"
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
				updateBinary(latestVersion)
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

func updateBinary(versionInfo structs.Channel) {
	Logger.Info(fmt.Sprintf("update binary called: %v", versionInfo))
	// download new binary, name it groundseg_new
	// delete groundseg binary
	// rename groundseg_new to groundseg
	// systemctl restart groundseg
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
