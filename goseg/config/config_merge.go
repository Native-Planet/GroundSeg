package config

import "groundseg/structs"

type ConfigMerger interface {
	Merge(defaultConfig structs.SysConfig, customConfig structs.SysConfig) structs.SysConfig
}

type DefaultConfigMerger struct{}

var configMerger ConfigMerger = DefaultConfigMerger{}

func SetConfigMerger(merger ConfigMerger) {
	if merger == nil {
		return
	}
	configMerger = merger
}

// MergeConfigs applies the configured schema-evolution policy.
func MergeConfigs(defaultConfig structs.SysConfig, customConfig structs.SysConfig) structs.SysConfig {
	if configMerger == nil {
		return defaultConfigMerger{}.Merge(defaultConfig, customConfig)
	}
	return configMerger.Merge(defaultConfig, customConfig)
}

// mergeConfigs keeps the historical function name used by bootstrap and tests.
func mergeConfigs(defaultConfig structs.SysConfig, customConfig structs.SysConfig) structs.SysConfig {
	return MergeConfigs(defaultConfig, customConfig)
}

func (DefaultConfigMerger) Merge(defaultConfig structs.SysConfig, customConfig structs.SysConfig) structs.SysConfig {
	mergedConfig := defaultConfig
	mergedConfig.AuthSession.Salt = ""
	for _, field := range allConfPatchFields() {
		field.merge(defaultConfig, customConfig, &mergedConfig)
	}
	return mergedConfig
}

type defaultConfigMerger struct{}

func (defaultConfigMerger) Merge(defaultConfig structs.SysConfig, customConfig structs.SysConfig) structs.SysConfig {
	return DefaultConfigMerger{}.Merge(defaultConfig, customConfig)
}
