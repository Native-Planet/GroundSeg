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

	if err := applyConfPatch(&configStruct, patch); err != nil {
		return err
	}

	if err := persistConfig(configStruct); err != nil {
		return fmt.Errorf("unable to persist config update: %w", err)
	}
	return nil
}

func applyConfPatch(target confPatchApplyTarget, patch *ConfPatch) error {
	for _, field := range allConfPatchFields() {
		service := field
		if err := service.apply(target, patch); err != nil {
			return err
		}
	}
	return nil
}
