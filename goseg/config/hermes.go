package config

import (
	"encoding/json"
	"fmt"
	"groundseg/defaults"
	"groundseg/structs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	hermesConfig structs.HermesConfig
	hermesMutex  sync.RWMutex
)

func HermesConf() structs.HermesConfig {
	hermesMutex.RLock()
	defer hermesMutex.RUnlock()
	return hermesConfig
}

func LoadHermesConfig() error {
	path := filepath.Join(BasePath, "settings", "hermes.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := CreateDefaultHermesConf(); err != nil {
			return err
		}
	}
	file, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to load Hermes config: %w", err)
	}
	var target structs.HermesConfig
	if err := json.Unmarshal(file, &target); err != nil {
		return fmt.Errorf("error decoding Hermes config: %w", err)
	}
	applyHermesDefaults(&target)
	hermesMutex.Lock()
	hermesConfig = target
	hermesMutex.Unlock()
	return nil
}

func CreateDefaultHermesConf() error {
	defaultConfig := defaults.HermesConfig
	path := filepath.Join(BasePath, "settings", "hermes.json")
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	return encoder.Encode(&defaultConfig)
}

func UpdateHermesConfig(input structs.HermesConfig) error {
	applyHermesDefaults(&input)
	hermesMutex.Lock()
	defer hermesMutex.Unlock()
	hermesConfig = input
	path := filepath.Join(BasePath, "settings", "hermes.json")
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(path), "hermes.json.*")
	if err != nil {
		return fmt.Errorf("error creating temp Hermes config: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(&input); err != nil {
		tmpFile.Close()
		return fmt.Errorf("error encoding Hermes config: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("error closing Hermes config: %v", err)
	}
	if fi, err := os.Stat(tmpPath); err != nil {
		return fmt.Errorf("error checking Hermes config: %v", err)
	} else if fi.Size() == 0 {
		return fmt.Errorf("refusing to persist empty Hermes config")
	}
	return os.Rename(tmpPath, path)
}

func applyHermesDefaults(target *structs.HermesConfig) {
	if target.Port == 0 {
		target.Port = defaults.HermesConfig.Port
	}
	if target.Image == "" {
		target.Image = defaults.HermesConfig.Image
	}
	if target.HermesVersion == "" {
		target.HermesVersion = defaults.HermesConfig.HermesVersion
	}
	if target.HermesAgentRef == "" {
		target.HermesAgentRef = defaults.HermesConfig.HermesAgentRef
	}
	if target.TlonAdapterVersion == "" {
		target.TlonAdapterVersion = defaults.HermesConfig.TlonAdapterVersion
	}
	if target.TlonAdapterRef == "" {
		target.TlonAdapterRef = defaults.HermesConfig.TlonAdapterRef
	}
	if target.ModelProvider == "" {
		target.ModelProvider = defaults.HermesConfig.ModelProvider
	}
	if target.Model == "" {
		target.Model = defaults.HermesConfig.Model
	}
	if target.WebProvider != "" {
		target.WebProvider = strings.TrimSpace(target.WebProvider)
	}
	target.WebURL = strings.TrimSpace(target.WebURL)
	target.APIKey = strings.TrimSpace(target.APIKey)
}
