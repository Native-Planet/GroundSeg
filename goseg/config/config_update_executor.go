package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"groundseg/structs"
)

func updateConfigFromPatch(patch *ConfigPatch) error {
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

	if err := applyConfigPatch(&configStruct, patch); err != nil {
		return err
	}

	if err := persistConfig(configStruct); err != nil {
		return fmt.Errorf("unable to persist config update: %w", err)
	}
	return nil
}

func applyConfigPatch(target configPatchApplyTarget, patch *ConfigPatch) error {
	for _, field := range allConfigPatchFields() {
		service := field
		if err := service.apply(target, patch); err != nil {
			return err
		}
	}
	return nil
}

func updateConfFromPatch(patch *ConfPatch) error {
	return updateConfigFromPatch(patch)
}

func applyConfPatch(target configPatchApplyTarget, patch *ConfPatch) error {
	return applyConfigPatch(target, patch)
}
