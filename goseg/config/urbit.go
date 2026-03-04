package config

// functions related to managing urbit config jsons & corresponding structs

import (
	"encoding/json"
	"fmt"
	"groundseg/structs"
	"os"
	"path/filepath"
	"sync"
)

type UrbitConfigSection int

const (
	UrbitConfigSectionRuntime UrbitConfigSection = iota
	UrbitConfigSectionNetwork
	UrbitConfigSectionSchedule
	UrbitConfigSectionFeature
	UrbitConfigSectionWeb
	UrbitConfigSectionBackup
)

func (section UrbitConfigSection) String() string {
	switch section {
	case UrbitConfigSectionRuntime:
		return "runtime"
	case UrbitConfigSectionNetwork:
		return "network"
	case UrbitConfigSectionSchedule:
		return "schedule"
	case UrbitConfigSectionFeature:
		return "feature"
	case UrbitConfigSectionWeb:
		return "web"
	case UrbitConfigSectionBackup:
		return "backup"
	default:
		return "unknown"
	}
}

var (
	UrbitsConfig = make(map[string]structs.UrbitDocker)
	urbitMutex   sync.RWMutex
)

// retrieve struct corresponding with urbit json file
func UrbitConf(pier string) structs.UrbitDocker {
	urbitMutex.RLock()
	defer urbitMutex.RUnlock()
	return UrbitsConfig[pier]
}

// this should eventually be a click/conn.c command
func GetMinIOLinkedStatus(patp string) bool {
	urbConf := UrbitConf(patp)
	return urbConf.MinIOLinked
}

// retrieve map of urbit config structs
func UrbitConfAll() map[string]structs.UrbitDocker {
	urbitMutex.RLock()
	defer urbitMutex.RUnlock()
	configCopy := make(map[string]structs.UrbitDocker, len(UrbitsConfig))
	for pier, conf := range UrbitsConfig {
		configCopy[pier] = conf
	}
	return configCopy
}

// load urbit conf json into memory
func LoadUrbitConfig(pier string) error {
	urbitMutex.Lock()
	defer urbitMutex.Unlock()
	targetStruct, err := loadUrbitConfigFromDisk(pier)
	if err != nil {
		return err
	}
	UrbitsConfig[pier] = targetStruct
	return nil
}

// Delete urbit config entry
func RemoveUrbitConfig(pier string) error {
	// remove from memory
	urbitMutex.Lock()
	defer urbitMutex.Unlock()
	delete(UrbitsConfig, pier)
	// remove from disk
	path := filepath.Join(BasePath(), "settings", "pier", pier+".json")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove urbit config for %s at %s: %w", pier, path, err)
	}
	return nil
}

// update the in-memory struct and save it to json
func UpdateUrbitConfig(inputConfig map[string]structs.UrbitDocker) error {
	urbitMutex.Lock()
	defer urbitMutex.Unlock()
	for pier, config := range inputConfig {
		if err := persistUrbitConfigLocked(pier, config); err != nil {
			return err
		}
	}
	return nil
}

// UpdateUrbit centralizes single-ship config mutation so callsites don't need ad-hoc map scaffolding.
func UpdateUrbit(pier string, mutateFn func(*structs.UrbitDocker) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	urbitMutex.Lock()
	defer urbitMutex.Unlock()

	current, ok := UrbitsConfig[pier]
	if !ok {
		loaded, err := loadUrbitConfigFromDisk(pier)
		if err != nil {
			return err
		}
		current = loaded
	}
	if err := mutateFn(&current); err != nil {
		return err
	}
	return persistUrbitConfigLocked(pier, current)
}

func UpdateUrbitRuntimeConfig(pier string, mutateFn func(*structs.UrbitRuntimeConfig) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	return UpdateUrbit(pier, func(conf *structs.UrbitDocker) error {
		return mutateFn(&conf.UrbitRuntimeConfig)
	})
}


func UpdateUrbitNetworkConfig(pier string, mutateFn func(*structs.UrbitNetworkConfig) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	return UpdateUrbit(pier, func(conf *structs.UrbitDocker) error {
		return mutateFn(&conf.UrbitNetworkConfig)
	})
}


func UpdateUrbitScheduleConfig(pier string, mutateFn func(*structs.UrbitScheduleConfig) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	return UpdateUrbit(pier, func(conf *structs.UrbitDocker) error {
		return mutateFn(&conf.UrbitScheduleConfig)
	})
}


func UpdateUrbitFeatureConfig(pier string, mutateFn func(*structs.UrbitFeatureConfig) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	return UpdateUrbit(pier, func(conf *structs.UrbitDocker) error {
		return mutateFn(&conf.UrbitFeatureConfig)
	})
}

func UpdateUrbitWebConfig(pier string, mutateFn func(*structs.UrbitWebConfig) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	return UpdateUrbit(pier, func(conf *structs.UrbitDocker) error {
		return mutateFn(&conf.UrbitWebConfig)
	})
}

func UpdateUrbitBackupConfig(pier string, mutateFn func(*structs.UrbitBackupConfig) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	return UpdateUrbit(pier, func(conf *structs.UrbitDocker) error {
		return mutateFn(&conf.UrbitBackupConfig)
	})
}

func loadUrbitConfigFromDisk(pier string) (structs.UrbitDocker, error) {
	confPath := filepath.Join(BasePath(), "settings", "pier", pier+".json")
	file, err := os.ReadFile(confPath)
	if err != nil {
		return structs.UrbitDocker{}, fmt.Errorf("unable to load %s config: %w", pier, err)
	}
	var targetStruct structs.UrbitDocker
	if err := json.Unmarshal(file, &targetStruct); err != nil {
		return structs.UrbitDocker{}, fmt.Errorf("error decoding %s JSON: %w", pier, err)
	}
	normalizeUrbitConfig(&targetStruct)
	return targetStruct, nil
}

func normalizeUrbitConfig(conf *structs.UrbitDocker) {
	if conf.SnapTime == 0 {
		conf.SnapTime = 60
	}
}

func persistUrbitConfigLocked(pier string, conf structs.UrbitDocker) error {
	normalizeUrbitConfig(&conf)
	path := filepath.Join(BasePath(), "settings", "pier", pier+".json")
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return fmt.Errorf("error creating urbit config dir for %s: %w", pier, err)
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(path), pier+".json.*")
	if err != nil {
		return fmt.Errorf("error creating temp file for %s: %w", pier, err)
	}
	// write and validate temp file before overwriting
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(&conf); err != nil {
		tmpFile.Close()
		return fmt.Errorf("error encoding %s config: %w", pier, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("error closing temp file for %s: %w", pier, err)
	}
	fi, err := os.Stat(tmpPath)
	if err != nil {
		return fmt.Errorf("error checking temp file for %s: %w", pier, err)
	}
	if fi.Size() == 0 {
		return fmt.Errorf("refusing to persist empty configuration for pier %s", pier)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("error moving temp file for %s: %w", pier, err)
	}
	UrbitsConfig[pier] = conf
	return nil
}
