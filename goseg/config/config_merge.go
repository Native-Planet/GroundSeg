package config

import (
	"strings"

	"groundseg/structs"
)

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
	mergedConfig := structs.SysConfig{}

	// GracefulExit
	mergedConfig.GracefulExit = customConfig.GracefulExit || defaultConfig.GracefulExit

	//LastKnownMDNS
	if customConfig.LastKnownMDNS != "" {
		mergedConfig.LastKnownMDNS = customConfig.LastKnownMDNS
	} else {
		mergedConfig.LastKnownMDNS = defaultConfig.LastKnownMDNS
	}

	// Setup
	// if pwhash is empty:
	//    "setup" is "start" (new install)
	// if pwhash not empty:
	//    if setup is empty:
	//        "setup" is "complete" (migration case)
	//    if setup not empty:
	//        "setup" remains (standard)
	if customConfig.PwHash == "" {
		mergedConfig.Setup = "start"       // new install
		mergedConfig.Salt = RandString(32) // reset salt
	} else {
		if customConfig.Setup == "" {
			mergedConfig.Setup = "complete" // migration case
		} else {
			mergedConfig.Setup = customConfig.Setup // standard
		}
	}

	// EndpointUrl
	if customConfig.EndpointUrl != "" {
		mergedConfig.EndpointUrl = customConfig.EndpointUrl
	} else {
		mergedConfig.EndpointUrl = defaultConfig.EndpointUrl
	}

	// ApiVersion
	if customConfig.ApiVersion != "" {
		mergedConfig.ApiVersion = customConfig.ApiVersion
	} else {
		mergedConfig.ApiVersion = defaultConfig.ApiVersion
	}

	// Piers
	if len(customConfig.Piers) > 0 {
		mergedConfig.Piers = customConfig.Piers
	} else {
		mergedConfig.Piers = defaultConfig.Piers
	}

	// NetCheck
	if customConfig.NetCheck != "" {
		mergedConfig.NetCheck = customConfig.NetCheck
	} else {
		mergedConfig.NetCheck = defaultConfig.NetCheck
	}

	// UpdateMode
	if customConfig.UpdateMode != "" {
		mergedConfig.UpdateMode = customConfig.UpdateMode
	} else {
		mergedConfig.UpdateMode = defaultConfig.UpdateMode
	}

	// UpdateUrl
	if customConfig.UpdateUrl != "" {
		mergedConfig.UpdateUrl = customConfig.UpdateUrl
	} else {
		mergedConfig.UpdateUrl = defaultConfig.UpdateUrl
	}

	// UpdateBranch
	if customConfig.UpdateBranch != "" {
		mergedConfig.UpdateBranch = customConfig.UpdateBranch
	} else {
		mergedConfig.UpdateBranch = defaultConfig.UpdateBranch
	}

	// SwapVal
	if customConfig.SwapVal != 0 {
		mergedConfig.SwapVal = customConfig.SwapVal
	} else {
		mergedConfig.SwapVal = defaultConfig.SwapVal
	}

	// SwapFile
	if customConfig.SwapFile != "" {
		mergedConfig.SwapFile = customConfig.SwapFile
	} else {
		mergedConfig.SwapFile = defaultConfig.SwapFile
	}

	// KeyFile
	if customConfig.KeyFile != "" {
		mergedConfig.KeyFile = customConfig.KeyFile
	} else {
		mergedConfig.KeyFile = defaultConfig.KeyFile
	}

	// Sessions
	if customConfig.Sessions.Authorized != nil {
		mergedConfig.Sessions.Authorized = customConfig.Sessions.Authorized
	} else {
		mergedConfig.Sessions.Authorized = defaultConfig.Sessions.Authorized
	}

	if customConfig.Sessions.Unauthorized != nil {
		mergedConfig.Sessions.Unauthorized = customConfig.Sessions.Unauthorized
	} else {
		mergedConfig.Sessions.Unauthorized = defaultConfig.Sessions.Unauthorized
	}

	// LinuxUpdates
	if customConfig.LinuxUpdates.Value != 0 {
		mergedConfig.LinuxUpdates.Value = customConfig.LinuxUpdates.Value
	} else {
		mergedConfig.LinuxUpdates.Value = defaultConfig.LinuxUpdates.Value
	}

	if customConfig.LinuxUpdates.Interval != "" {
		mergedConfig.LinuxUpdates.Interval = customConfig.LinuxUpdates.Interval
	} else {
		mergedConfig.LinuxUpdates.Interval = defaultConfig.LinuxUpdates.Interval
	}

	// DockerData
	if customConfig.DockerData != "" {
		mergedConfig.DockerData = customConfig.DockerData
	} else {
		mergedConfig.DockerData = defaultConfig.DockerData
	}

	// WgOn
	mergedConfig.WgOn = customConfig.WgOn || defaultConfig.WgOn

	// WgRegistered
	mergedConfig.WgRegistered = customConfig.WgRegistered || defaultConfig.WgRegistered

	// StartramSetReminder
	mergedConfig.StartramSetReminder.One = customConfig.StartramSetReminder.One || defaultConfig.StartramSetReminder.One
	mergedConfig.StartramSetReminder.Three = customConfig.StartramSetReminder.Three || defaultConfig.StartramSetReminder.Three
	mergedConfig.StartramSetReminder.Seven = customConfig.StartramSetReminder.Seven || defaultConfig.StartramSetReminder.Seven

	// DiskWarning
	mergedConfig.DiskWarning = customConfig.DiskWarning

	// PwHash
	if customConfig.PwHash != "" {
		mergedConfig.PwHash = customConfig.PwHash
	} else {
		mergedConfig.PwHash = defaultConfig.PwHash
	}

	// C2cInterval
	if customConfig.C2cInterval != 0 {
		mergedConfig.C2cInterval = customConfig.C2cInterval
	} else {
		mergedConfig.C2cInterval = defaultConfig.C2cInterval
	}

	// GsVersion
	if customConfig.GsVersion != "" {
		mergedConfig.GsVersion = customConfig.GsVersion
	} else {
		mergedConfig.GsVersion = defaultConfig.GsVersion
	}

	// CfgDir
	if customConfig.CfgDir != "" {
		mergedConfig.CfgDir = customConfig.CfgDir
	} else {
		mergedConfig.CfgDir = defaultConfig.CfgDir
	}

	// UpdateInterval
	if customConfig.UpdateInterval != 0 {
		mergedConfig.UpdateInterval = customConfig.UpdateInterval
	} else {
		mergedConfig.UpdateInterval = defaultConfig.UpdateInterval
	}

	// BinHash
	if customConfig.BinHash != "" {
		mergedConfig.BinHash = customConfig.BinHash
	} else {
		mergedConfig.BinHash = defaultConfig.BinHash
	}

	// Pubkey
	if customConfig.Pubkey != "" {
		mergedConfig.Pubkey = customConfig.Pubkey
	} else {
		mergedConfig.Pubkey = defaultConfig.Pubkey
	}

	// Privkey
	if customConfig.Privkey != "" {
		mergedConfig.Privkey = customConfig.Privkey
	} else {
		mergedConfig.Privkey = defaultConfig.Privkey
	}

	// Salt
	if mergedConfig.Salt == "" {
		mergedConfig.Salt = customConfig.Salt
	}

	// PenpaiAllow
	mergedConfig.PenpaiAllow = customConfig.PenpaiAllow || defaultConfig.PenpaiAllow

	// PenpaiCores
	if customConfig.PenpaiCores != 0 {
		mergedConfig.PenpaiCores = customConfig.PenpaiCores
	} else {
		mergedConfig.PenpaiCores = defaultConfig.PenpaiCores
	}

	// PenpaiModels
	// always use defaults as newest
	mergedConfig.PenpaiModels = defaultConfig.PenpaiModels

	// PenpaiRunning
	mergedConfig.PenpaiRunning = customConfig.PenpaiRunning

	// PenpaiActive
	validModel := false
	for _, model := range defaultConfig.PenpaiModels {
		if strings.EqualFold(model.ModelName, customConfig.PenpaiActive) {
			validModel = true
		}
	}
	if customConfig.PenpaiActive != "" && validModel {
		mergedConfig.PenpaiActive = customConfig.PenpaiActive
	} else {
		mergedConfig.PenpaiActive = defaultConfig.PenpaiActive
	}

	// 502 checker
	if customConfig.Disable502 {
		mergedConfig.Disable502 = customConfig.Disable502
	} else {
		mergedConfig.Disable502 = defaultConfig.Disable502
	}
	mergedConfig.RemoteBackupPassword = customConfig.RemoteBackupPassword
	if customConfig.SnapTime == 0 {
		mergedConfig.SnapTime = defaultConfig.SnapTime
	} else {
		mergedConfig.SnapTime = customConfig.SnapTime
	}

	return mergedConfig
}

type defaultConfigMerger struct{}

func (defaultConfigMerger) Merge(defaultConfig structs.SysConfig, customConfig structs.SysConfig) structs.SysConfig {
	return DefaultConfigMerger{}.Merge(defaultConfig, customConfig)
}
