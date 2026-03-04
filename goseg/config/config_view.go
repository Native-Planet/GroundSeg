package config

import (
	"groundseg/structs"
)

func StartramSettingsSnapshot() StartramSettings {
	conf := Conf()
	return StartramSettings{
		EndpointURL:          conf.Connectivity.EndpointUrl,
		Pubkey:               conf.Startram.Pubkey,
		RemoteBackupPassword: conf.Connectivity.RemoteBackupPassword,
		WgRegistered:         conf.Connectivity.WgRegistered,
		WgOn:                 conf.Connectivity.WgOn,
		Piers:                copyStringSlice(conf.Connectivity.Piers),
	}
}

func AuthSettingsSnapshot() AuthSettings {
	conf := Conf()
	authorizedSessions := make(map[string]structs.SessionInfo, len(conf.AuthSession.Sessions.Authorized))
	for tokenID, session := range conf.AuthSession.Sessions.Authorized {
		authorizedSessions[tokenID] = session
	}
	return AuthSettings{
			KeyFile:            conf.AuthSession.KeyFile,
			Salt:               conf.AuthSession.Salt,
			PasswordHash:       conf.AuthSession.PwHash,
			AuthorizedSessions: authorizedSessions,
		}
}

func PenpaiSettingsSnapshot() PenpaiSettings {
	conf := Conf()
	return PenpaiSettings{
		Models:      append([]structs.Penpai(nil), conf.Penpai.PenpaiModels...),
		Allowed:     conf.Penpai.PenpaiAllow,
		ActiveModel: conf.Penpai.PenpaiActive,
		Running:     conf.Penpai.PenpaiRunning,
		ActiveCores: conf.Penpai.PenpaiCores,
	}
}

func Check502SettingsSnapshot() Check502Settings {
	conf := Conf()
	return Check502Settings{
		Piers:      copyStringSlice(conf.Connectivity.Piers),
		WgOn:       conf.Connectivity.WgOn,
		Disable502: conf.Runtime.Disable502,
	}
}

func HealthCheckSettingsSnapshot() HealthCheckSettings {
	conf := Conf()
	return HealthCheckSettings{
		Piers:        copyStringSlice(conf.Connectivity.Piers),
		DiskWarnings: copyDiskWarnings(conf.Connectivity.DiskWarning),
	}
}

func ShipSettingsSnapshot() ShipSettings {
	conf := Conf()
	return ShipSettings{Piers: copyStringSlice(conf.Connectivity.Piers)}
}

func ConnectivitySettingsSnapshot() ConnectivitySettings {
	conf := Conf()
	return ConnectivitySettings{C2cInterval: conf.Connectivity.C2cInterval}
}

func UpdateSettingsSnapshot() UpdateSettings {
	conf := Conf()
	return UpdateSettings{
		UpdateMode:   conf.Connectivity.UpdateMode,
		UpdateBranch: conf.Connectivity.UpdateBranch,
	}
}

func SwapSettingsSnapshot() SwapSettings {
	conf := Conf()
	return SwapSettings{
		SwapFile: conf.Runtime.SwapFile,
		SwapVal:  conf.Runtime.SwapVal,
	}
}

func ShipRuntimeSettingsSnapshot() ShipRuntimeSettings {
	conf := Conf()
	return ShipRuntimeSettings{SnapTime: conf.Runtime.SnapTime}
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
