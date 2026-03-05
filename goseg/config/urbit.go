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

type UrbitConfigSection = structs.UrbitConfigSection

const (
	UrbitConfigSectionRuntime  = structs.UrbitConfigSectionRuntime
	UrbitConfigSectionNetwork  = structs.UrbitConfigSectionNetwork
	UrbitConfigSectionSchedule = structs.UrbitConfigSectionSchedule
	UrbitConfigSectionFeature  = structs.UrbitConfigSectionFeature
	UrbitConfigSectionWeb      = structs.UrbitConfigSectionWeb
	UrbitConfigSectionBackup   = structs.UrbitConfigSectionBackup
)

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

func UpdateUrbitSection(pier string, section UrbitConfigSection, mutateFn func(any) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	return UpdateUrbit(pier, func(conf *structs.UrbitDocker) error {
		return mutateUrbitSection(conf, section, mutateFn)
	})
}

func mutateUrbitSection(conf *structs.UrbitDocker, section UrbitConfigSection, mutateFn func(any) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	switch section {
	case UrbitConfigSectionRuntime:
		return mutateFn(&conf.UrbitRuntimeConfig)
	case UrbitConfigSectionNetwork:
		return mutateFn(&conf.UrbitNetworkConfig)
	case UrbitConfigSectionSchedule:
		return mutateFn(&conf.UrbitScheduleConfig)
	case UrbitConfigSectionFeature:
		return mutateFn(&conf.UrbitFeatureConfig)
	case UrbitConfigSectionWeb:
		return mutateFn(&conf.UrbitWebConfig)
	case UrbitConfigSectionBackup:
		return mutateFn(&conf.UrbitBackupConfig)
	default:
		return fmt.Errorf("unsupported urbit section: %s", section)
	}
}

// AdaptUrbitSectionMutation adapts a typed urbit section mutation function to the generic section update contract.
func AdaptUrbitSectionMutation[T any](mutateFn func(*T) error) func(any) error {
	return func(section any) error {
		typed, ok := section.(*T)
		if !ok {
			return fmt.Errorf("unsupported section payload: expected %T, got %T", *new(T), section)
		}
		return mutateFn(typed)
	}
}

func UpdateUrbitRuntimeConfig(pier string, mutateFn func(*structs.UrbitRuntimeConfig) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	return UpdateUrbitSection(pier, UrbitConfigSectionRuntime, AdaptUrbitSectionMutation(mutateFn))
}

func UpdateUrbitNetworkConfig(pier string, mutateFn func(*structs.UrbitNetworkConfig) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	return UpdateUrbitSection(pier, UrbitConfigSectionNetwork, AdaptUrbitSectionMutation(mutateFn))
}

func UpdateUrbitScheduleConfig(pier string, mutateFn func(*structs.UrbitScheduleConfig) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	return UpdateUrbitSection(pier, UrbitConfigSectionSchedule, AdaptUrbitSectionMutation(mutateFn))
}

func UpdateUrbitFeatureConfig(pier string, mutateFn func(*structs.UrbitFeatureConfig) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	return UpdateUrbitSection(pier, UrbitConfigSectionFeature, AdaptUrbitSectionMutation(mutateFn))
}

func UpdateUrbitWebConfig(pier string, mutateFn func(*structs.UrbitWebConfig) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	return UpdateUrbitSection(pier, UrbitConfigSectionWeb, AdaptUrbitSectionMutation(mutateFn))
}

func UpdateUrbitBackupConfig(pier string, mutateFn func(*structs.UrbitBackupConfig) error) error {
	if mutateFn == nil {
		return fmt.Errorf("mutate function is required")
	}
	return UpdateUrbitSection(pier, UrbitConfigSectionBackup, AdaptUrbitSectionMutation(mutateFn))
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
