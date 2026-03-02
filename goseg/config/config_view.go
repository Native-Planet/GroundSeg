package config

import (
	"groundseg/structs"
)

type ConfigView interface {
	StartramSettings() StartramSettings
	AuthSettings() AuthSettings
	PenpaiSettings() PenpaiSettings
	Check502Settings() Check502Settings
	HealthCheckSettings() HealthCheckSettings
	ShipSettings() ShipSettings
	ConnectivitySettings() ConnectivitySettings
	UpdateSettings() UpdateSettings
	SwapSettings() SwapSettings
	ShipRuntimeSettings() ShipRuntimeSettings
	RuntimeSettings() RuntimeSettings
	CurrentRuntimeContext() RuntimeContext
}

// ConfigViewSnapshot captures a frozen copy of runtime config state.
type ConfigViewSnapshot struct {
	conf           structs.SysConfig
	runtimeContext RuntimeContext
}

// NewConfigViewSnapshot returns a full config view without callers needing to
// read mutable global state directly.
func NewConfigViewSnapshot() ConfigView {
	return ConfigViewSnapshot{
		conf:           Conf(),
		runtimeContext: RuntimeContextSnapshot(),
	}
}

// NewConfigViewSnapshotFromConfig builds a view from a provided config value.
func NewConfigViewSnapshotFromConfig(conf structs.SysConfig) ConfigView {
	return ConfigViewSnapshot{
		conf:           conf,
		runtimeContext: RuntimeContextSnapshot(),
	}
}

func (snapshot ConfigViewSnapshot) StartramSettings() StartramSettings {
	return StartramSettings{
		EndpointURL:          snapshot.conf.EndpointUrl,
		Pubkey:               snapshot.conf.Pubkey,
		RemoteBackupPassword: snapshot.conf.RemoteBackupPassword,
		WgRegistered:         snapshot.conf.WgRegistered,
		WgOn:                 snapshot.conf.WgOn,
		Piers:                append([]string(nil), snapshot.conf.Piers...),
	}
}

func (snapshot ConfigViewSnapshot) AuthSettings() AuthSettings {
	authorizedSessions := make(map[string]structs.SessionInfo, len(snapshot.conf.Sessions.Authorized))
	for tokenID, session := range snapshot.conf.Sessions.Authorized {
		authorizedSessions[tokenID] = session
	}
	return AuthSettings{
		KeyFile:            snapshot.conf.KeyFile,
		Salt:               snapshot.conf.Salt,
		PasswordHash:       snapshot.conf.PwHash,
		AuthorizedSessions: authorizedSessions,
	}
}

func (snapshot ConfigViewSnapshot) PenpaiSettings() PenpaiSettings {
	return PenpaiSettings{
		Models:      append([]structs.Penpai(nil), snapshot.conf.PenpaiModels...),
		Allowed:     snapshot.conf.PenpaiAllow,
		ActiveModel: snapshot.conf.PenpaiActive,
		Running:     snapshot.conf.PenpaiRunning,
		ActiveCores: snapshot.conf.PenpaiCores,
	}
}

func (snapshot ConfigViewSnapshot) Check502Settings() Check502Settings {
	return Check502Settings{
		Piers:      append([]string(nil), snapshot.conf.Piers...),
		WgOn:       snapshot.conf.WgOn,
		Disable502: snapshot.conf.Disable502,
	}
}

func (snapshot ConfigViewSnapshot) HealthCheckSettings() HealthCheckSettings {
	return HealthCheckSettings{
		Piers:        append([]string(nil), snapshot.conf.Piers...),
		DiskWarnings: copyDiskWarnings(snapshot.conf.DiskWarning),
	}
}

func (snapshot ConfigViewSnapshot) ShipSettings() ShipSettings {
	return ShipSettings{
		Piers: append([]string(nil), snapshot.conf.Piers...),
	}
}

func (snapshot ConfigViewSnapshot) ConnectivitySettings() ConnectivitySettings {
	return ConnectivitySettings{
		C2cInterval: snapshot.conf.C2cInterval,
	}
}

func (snapshot ConfigViewSnapshot) UpdateSettings() UpdateSettings {
	return UpdateSettings{
		UpdateMode:   snapshot.conf.UpdateMode,
		UpdateBranch: snapshot.conf.UpdateBranch,
	}
}

func (snapshot ConfigViewSnapshot) SwapSettings() SwapSettings {
	return SwapSettings{
		SwapFile: snapshot.conf.SwapFile,
		SwapVal:  snapshot.conf.SwapVal,
	}
}

func (snapshot ConfigViewSnapshot) ShipRuntimeSettings() ShipRuntimeSettings {
	return ShipRuntimeSettings{
		SnapTime: snapshot.conf.SnapTime,
	}
}

func (snapshot ConfigViewSnapshot) RuntimeSettings() RuntimeSettings {
	return RuntimeSettings{
		BasePath:     snapshot.runtimeContext.BasePath,
		Architecture: snapshot.runtimeContext.Architecture,
		DebugMode:    snapshot.runtimeContext.DebugMode,
	}
}

func (snapshot ConfigViewSnapshot) CurrentRuntimeContext() RuntimeContext {
	return RuntimeContext{
		BasePath:     snapshot.runtimeContext.BasePath,
		Architecture: snapshot.runtimeContext.Architecture,
		DebugMode:    snapshot.runtimeContext.DebugMode,
		DockerDir:    snapshot.runtimeContext.DockerDir,
	}
}

func StartramSettingsSnapshot() StartramSettings {
	return NewConfigViewSnapshot().StartramSettings()
}

func AuthSettingsSnapshot() AuthSettings {
	return NewConfigViewSnapshot().AuthSettings()
}

func PenpaiSettingsSnapshot() PenpaiSettings {
	return NewConfigViewSnapshot().PenpaiSettings()
}

func Check502SettingsSnapshot() Check502Settings {
	return NewConfigViewSnapshot().Check502Settings()
}

func HealthCheckSettingsSnapshot() HealthCheckSettings {
	return NewConfigViewSnapshot().HealthCheckSettings()
}

func ShipSettingsSnapshot() ShipSettings {
	return NewConfigViewSnapshot().ShipSettings()
}

func ConnectivitySettingsSnapshot() ConnectivitySettings {
	return NewConfigViewSnapshot().ConnectivitySettings()
}

func UpdateSettingsSnapshot() UpdateSettings {
	return NewConfigViewSnapshot().UpdateSettings()
}

func SwapSettingsSnapshot() SwapSettings {
	return NewConfigViewSnapshot().SwapSettings()
}

func ShipRuntimeSettingsSnapshot() ShipRuntimeSettings {
	return NewConfigViewSnapshot().ShipRuntimeSettings()
}

func RuntimeSettingsSnapshot() RuntimeSettings {
	return NewConfigViewSnapshot().RuntimeSettings()
}
