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

type urbitConfigSubConfig string

const (
	urbitRuntimeConfigSubConfig  urbitConfigSubConfig = "runtime"
	urbitNetworkConfigSubConfig  urbitConfigSubConfig = "network"
	urbitScheduleConfigSubConfig urbitConfigSubConfig = "schedule"
	urbitFeatureConfigSubConfig  urbitConfigSubConfig = "feature"
	urbitWebConfigSubConfig      urbitConfigSubConfig = "web"
	urbitBackupConfigSubConfig   urbitConfigSubConfig = "backup"
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
	err := os.Remove(filepath.Join(BasePath(), "settings", "pier", pier+".json"))
	return err
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
	return UpdateUrbitSectionConfig(pier, UrbitConfigSectionRuntime, mutateFn)
}

func UpdateUrbitNetworkConfig(pier string, mutateFn func(*structs.UrbitNetworkConfig) error) error {
	return UpdateUrbitSectionConfig(pier, UrbitConfigSectionNetwork, mutateFn)
}

func UpdateUrbitScheduleConfig(pier string, mutateFn func(*structs.UrbitScheduleConfig) error) error {
	return UpdateUrbitSectionConfig(pier, UrbitConfigSectionSchedule, mutateFn)
}

func UpdateUrbitFeatureConfig(pier string, mutateFn func(*structs.UrbitFeatureConfig) error) error {
	return UpdateUrbitSectionConfig(pier, UrbitConfigSectionFeature, mutateFn)
}

func UpdateUrbitWebConfig(pier string, mutateFn func(*structs.UrbitWebConfig) error) error {
	return UpdateUrbitSectionConfig(pier, UrbitConfigSectionWeb, mutateFn)
}

func UpdateUrbitBackupConfig(pier string, mutateFn func(*structs.UrbitBackupConfig) error) error {
	return UpdateUrbitSectionConfig(pier, UrbitConfigSectionBackup, mutateFn)
}

func UpdateUrbitSectionConfig(pier string, section UrbitConfigSection, mutateFn any) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	if !isAllowedUrbitConfigSubConfig(section) {
		return fmt.Errorf("unsupported urbit config scope: %s", section)
	}
	switch section {
	case UrbitConfigSectionRuntime:
		runtimeMutate, ok := mutateFn.(func(*structs.UrbitRuntimeConfig) error)
		if !ok {
			return fmt.Errorf("section %s expects mutate func(*structs.UrbitRuntimeConfig)", section)
		}
		return updateUrbitSubConfig(pier, urbitRuntimeConfigSubConfig, func(conf *structs.UrbitDocker) error {
			return runtimeMutate(&conf.UrbitRuntimeConfig)
		})
	case UrbitConfigSectionNetwork:
		networkMutate, ok := mutateFn.(func(*structs.UrbitNetworkConfig) error)
		if !ok {
			return fmt.Errorf("section %s expects mutate func(*structs.UrbitNetworkConfig)", section)
		}
		return updateUrbitSubConfig(pier, urbitNetworkConfigSubConfig, func(conf *structs.UrbitDocker) error {
			return networkMutate(&conf.UrbitNetworkConfig)
		})
	case UrbitConfigSectionSchedule:
		scheduleMutate, ok := mutateFn.(func(*structs.UrbitScheduleConfig) error)
		if !ok {
			return fmt.Errorf("section %s expects mutate func(*structs.UrbitScheduleConfig)", section)
		}
		return updateUrbitSubConfig(pier, urbitScheduleConfigSubConfig, func(conf *structs.UrbitDocker) error {
			return scheduleMutate(&conf.UrbitScheduleConfig)
		})
	case UrbitConfigSectionFeature:
		featureMutate, ok := mutateFn.(func(*structs.UrbitFeatureConfig) error)
		if !ok {
			return fmt.Errorf("section %s expects mutate func(*structs.UrbitFeatureConfig)", section)
		}
		return updateUrbitSubConfig(pier, urbitFeatureConfigSubConfig, func(conf *structs.UrbitDocker) error {
			return featureMutate(&conf.UrbitFeatureConfig)
		})
	case UrbitConfigSectionWeb:
		webMutate, ok := mutateFn.(func(*structs.UrbitWebConfig) error)
		if !ok {
			return fmt.Errorf("section %s expects mutate func(*structs.UrbitWebConfig)", section)
		}
		return updateUrbitSubConfig(pier, urbitWebConfigSubConfig, func(conf *structs.UrbitDocker) error {
			return webMutate(&conf.UrbitWebConfig)
		})
	case UrbitConfigSectionBackup:
		backupMutate, ok := mutateFn.(func(*structs.UrbitBackupConfig) error)
		if !ok {
			return fmt.Errorf("section %s expects mutate func(*structs.UrbitBackupConfig)", section)
		}
		return updateUrbitSubConfig(pier, urbitBackupConfigSubConfig, func(conf *structs.UrbitDocker) error {
			return backupMutate(&conf.UrbitBackupConfig)
		})
	default:
		return fmt.Errorf("unsupported urbit config scope: %s", section)
	}
	return nil
}

func isAllowedUrbitConfigSubConfig(section UrbitConfigSection) bool {
	switch section {
	case UrbitConfigSectionRuntime,
		UrbitConfigSectionNetwork,
		UrbitConfigSectionSchedule,
		UrbitConfigSectionFeature,
		UrbitConfigSectionWeb,
		UrbitConfigSectionBackup:
		return true
	default:
		return false
	}
}

func updateUrbitSubConfig(pier string, scope urbitConfigSubConfig, mutateFn func(*structs.UrbitDocker) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	if !isLegacyUrbitConfigSubConfig(scope) {
		return fmt.Errorf("unsupported urbit config scope: %s", scope)
	}
	return UpdateUrbit(pier, mutateFn)
}

func isLegacyUrbitConfigSubConfig(scope urbitConfigSubConfig) bool {
	switch scope {
	case urbitRuntimeConfigSubConfig,
		urbitNetworkConfigSubConfig,
		urbitScheduleConfigSubConfig,
		urbitFeatureConfigSubConfig,
		urbitWebConfigSubConfig,
		urbitBackupConfigSubConfig:
		return true
	default:
		return false
	}
}

func loadUrbitConfigFromDisk(pier string) (structs.UrbitDocker, error) {
	confPath := filepath.Join(BasePath(), "settings", "pier", pier+".json")
	file, err := os.ReadFile(confPath)
	if err != nil {
		return structs.UrbitDocker{}, fmt.Errorf("unable to load %s config: %v", pier, err)
	}
	var targetStruct structs.UrbitDocker
	if err := json.Unmarshal(file, &targetStruct); err != nil {
		return structs.UrbitDocker{}, fmt.Errorf("error decoding %s JSON: %v", pier, err)
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
		return fmt.Errorf("error creating urbit config dir for %s: %v", pier, err)
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(path), pier+".json.*")
	if err != nil {
		return fmt.Errorf("error creating temp file for %s: %v", pier, err)
	}
	// write and validate temp file before overwriting
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(&conf); err != nil {
		tmpFile.Close()
		return fmt.Errorf("error encoding %s config: %v", pier, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("error closing temp file for %s: %v", pier, err)
	}
	fi, err := os.Stat(tmpPath)
	if err != nil {
		return fmt.Errorf("error checking temp file for %s: %v", pier, err)
	}
	if fi.Size() == 0 {
		return fmt.Errorf("refusing to persist empty configuration for pier %s", pier)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("error moving temp file for %s: %v", pier, err)
	}
	UrbitsConfig[pier] = conf
	return nil
}
