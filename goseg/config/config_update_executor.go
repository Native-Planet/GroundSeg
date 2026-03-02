package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"groundseg/structs"
)

func updateConfFromPatch(patch *ConfPatch) error {
	confMutex.Lock()
	defer confMutex.Unlock()

	file, err := ioutil.ReadFile(ConfigFilePath())
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	var configStruct structs.SysConfig
	if err := json.Unmarshal(file, &configStruct); err != nil {
		return fmt.Errorf("error decoding JSON: %w", err)
	}

	applyConfPatch(&configStruct, patch)

	if err := persistConfig(configStruct); err != nil {
		return fmt.Errorf("unable to persist config update: %w", err)
	}
	return nil
}

func applyConfPatch(configStruct *structs.SysConfig, patch *ConfPatch) {
	for _, field := range confPatchRegistry {
		field.apply(configStruct, patch)
	}
}
