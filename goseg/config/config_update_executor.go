package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"groundseg/structs"
)

type confPatchApplyContext struct {
	config *structs.SysConfig
}

func (ctx confPatchApplyContext) UpdateConnectivityConfig(update func(*structs.ConnectivityConfig)) {
	if ctx.config != nil && update != nil {
		ctx.config.UpdateConnectivityConfig(update)
	}
}

func (ctx confPatchApplyContext) UpdateRuntimeConfig(update func(*structs.RuntimeConfig)) {
	if ctx.config != nil && update != nil {
		ctx.config.UpdateRuntimeConfig(update)
	}
}

func (ctx confPatchApplyContext) UpdateStartramConfig(update func(*structs.StartramConfig)) {
	if ctx.config != nil && update != nil {
		ctx.config.UpdateStartramConfig(update)
	}
}

func (ctx confPatchApplyContext) UpdatePenpaiConfig(update func(*structs.PenpaiConfig)) {
	if ctx.config != nil && update != nil {
		ctx.config.UpdatePenpaiConfig(update)
	}
}

func (ctx confPatchApplyContext) UpdateAuthSessionConfig(update func(*structs.AuthSessionConfig)) {
	if ctx.config != nil && update != nil {
		ctx.config.UpdateAuthSessionConfig(update)
	}
}

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

	applyTarget := confPatchApplyContext{config: &configStruct}
	applyConfPatch(applyTarget, patch)

	if err := persistConfig(configStruct); err != nil {
		return fmt.Errorf("unable to persist config update: %w", err)
	}
	return nil
}

func applyConfPatch(target confPatchApplyTarget, patch *ConfPatch) {
	for _, field := range confPatchRegistry {
		field.apply(target, patch)
	}
}
