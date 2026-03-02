package config

import (
	"groundseg/structs"
)

func StartramSettingsSnapshot() StartramSettings {
	conf := Conf()
	return StartramSettings{
		EndpointURL:          conf.EndpointUrl,
		Pubkey:               conf.Pubkey,
		RemoteBackupPassword: conf.RemoteBackupPassword,
		WgRegistered:         conf.WgRegistered,
		WgOn:                 conf.WgOn,
		Piers:                copyStringSlice(conf.Piers),
	}
}

func AuthSettingsSnapshot() AuthSettings {
	conf := Conf()
	authorizedSessions := make(map[string]structs.SessionInfo, len(conf.Sessions.Authorized))
	for tokenID, session := range conf.Sessions.Authorized {
		authorizedSessions[tokenID] = session
	}
	return AuthSettings{
		KeyFile:            conf.KeyFile,
		Salt:               conf.Salt,
		PasswordHash:       conf.PwHash,
		AuthorizedSessions: authorizedSessions,
	}
}

func PenpaiSettingsSnapshot() PenpaiSettings {
	conf := Conf()
	return PenpaiSettings{
		Models:      append([]structs.Penpai(nil), conf.PenpaiModels...),
		Allowed:     conf.PenpaiAllow,
		ActiveModel: conf.PenpaiActive,
		Running:     conf.PenpaiRunning,
		ActiveCores: conf.PenpaiCores,
	}
}

func Check502SettingsSnapshot() Check502Settings {
	conf := Conf()
	return Check502Settings{
		Piers:      copyStringSlice(conf.Piers),
		WgOn:       conf.WgOn,
		Disable502: conf.Disable502,
	}
}

func HealthCheckSettingsSnapshot() HealthCheckSettings {
	conf := Conf()
	return HealthCheckSettings{
		Piers:        copyStringSlice(conf.Piers),
		DiskWarnings: copyDiskWarnings(conf.DiskWarning),
	}
}

func ShipSettingsSnapshot() ShipSettings {
	conf := Conf()
	return ShipSettings{Piers: copyStringSlice(conf.Piers)}
}

func ConnectivitySettingsSnapshot() ConnectivitySettings {
	conf := Conf()
	return ConnectivitySettings{C2cInterval: conf.C2cInterval}
}

func UpdateSettingsSnapshot() UpdateSettings {
	conf := Conf()
	return UpdateSettings{
		UpdateMode:   conf.UpdateMode,
		UpdateBranch: conf.UpdateBranch,
	}
}

func SwapSettingsSnapshot() SwapSettings {
	conf := Conf()
	return SwapSettings{
		SwapFile: conf.SwapFile,
		SwapVal:  conf.SwapVal,
	}
}

func ShipRuntimeSettingsSnapshot() ShipRuntimeSettings {
	conf := Conf()
	return ShipRuntimeSettings{SnapTime: conf.SnapTime}
}

func RuntimeSettingsSnapshot() RuntimeSettings {
	context := RuntimeContextSnapshot()
	return RuntimeSettings{
		BasePath:     context.BasePath,
		Architecture: context.Architecture,
		DebugMode:    context.DebugMode,
	}
}

func copyStringSlice(values []string) []string {
	return append([]string(nil), values...)
}
