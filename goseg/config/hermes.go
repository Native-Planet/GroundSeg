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
	return writeJSONDurably(path, &defaultConfig, 0o644)
}

func UpdateHermesConfig(input structs.HermesConfig) error {
	applyHermesDefaults(&input)
	hermesMutex.Lock()
	defer hermesMutex.Unlock()
	path := filepath.Join(BasePath, "settings", "hermes.json")
	if err := writeJSONDurably(path, &input, 0o644); err != nil {
		return fmt.Errorf("error persisting Hermes config: %v", err)
	}
	hermesConfig = input
	return nil
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
