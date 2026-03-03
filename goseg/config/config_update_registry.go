package config

import "groundseg/structs"

func applyConnectivityMutation[T any](apply func(*structs.ConnectivityConfig, T)) confPatchApplyValueFn[T] {
	return func(target confPatchApplyTarget, value T) {
		target.UpdateConnectivityConfig(func(domain *structs.ConnectivityConfig) {
			apply(domain, value)
		})
	}
}

func applyRuntimeMutation[T any](apply func(*structs.RuntimeConfig, T)) confPatchApplyValueFn[T] {
	return func(target confPatchApplyTarget, value T) {
		target.UpdateRuntimeConfig(func(domain *structs.RuntimeConfig) {
			apply(domain, value)
		})
	}
}

func applyStartramMutation[T any](apply func(*structs.StartramConfig, T)) confPatchApplyValueFn[T] {
	return func(target confPatchApplyTarget, value T) {
		target.UpdateStartramConfig(func(domain *structs.StartramConfig) {
			apply(domain, value)
		})
	}
}

func applyPenpaiMutation[T any](apply func(*structs.PenpaiConfig, T)) confPatchApplyValueFn[T] {
	return func(target confPatchApplyTarget, value T) {
		target.UpdatePenpaiConfig(func(domain *structs.PenpaiConfig) {
			apply(domain, value)
		})
	}
}

func applyAuthSessionMutation[T any](apply func(*structs.AuthSessionConfig, T)) confPatchApplyValueFn[T] {
	return func(target confPatchApplyTarget, value T) {
		target.UpdateAuthSessionConfig(func(domain *structs.AuthSessionConfig) {
			apply(domain, value)
		})
	}
}

var confPatchRegistry = []confPatchField{
	// Connectivity
	newConfPatchField(
		"piers",
		"Piers",
		func(value interface{}) ([]string, error) {
			return parseStringSliceValue("piers", value)
		},
		confPatchValueRef[[]string]{getTarget: func(patch *ConfPatch) **[]string { return &patch.Piers }},
		applyConnectivityMutation(func(config *structs.ConnectivityConfig, value []string) {
			config.Piers = value
		}),
		func(value []string) bool {
			return len(value) > 0
		},
	),
	newConfPatchField(
		"wgOn",
		"WgOn",
		func(value interface{}) (bool, error) {
			return parseBoolValue("wgOn", value)
		},
		confPatchValueRef[bool]{getTarget: func(patch *ConfPatch) **bool { return &patch.WgOn }},
		applyConnectivityMutation(func(config *structs.ConnectivityConfig, value bool) {
			config.WgOn = value
		}),
		nil,
	),
	newConfPatchField(
		"startramSetReminderOne",
		"StartramReminderOne",
		func(value interface{}) (bool, error) {
			return parseBoolValue("startramSetReminderOne", value)
		},
		confPatchValueRef[bool]{getTarget: func(patch *ConfPatch) **bool { return &patch.StartramReminderOne }},
		applyStartramMutation(func(config *structs.StartramConfig, value bool) {
			config.StartramSetReminder.One = value
		}),
		nil,
	),
	newConfPatchField(
		"startramSetReminderThree",
		"StartramReminderThree",
		func(value interface{}) (bool, error) {
			return parseBoolValue("startramSetReminderThree", value)
		},
		confPatchValueRef[bool]{getTarget: func(patch *ConfPatch) **bool { return &patch.StartramReminderThree }},
		applyStartramMutation(func(config *structs.StartramConfig, value bool) {
			config.StartramSetReminder.Three = value
		}),
		nil,
	),
	newConfPatchField(
		"startramSetReminderSeven",
		"StartramReminderSeven",
		func(value interface{}) (bool, error) {
			return parseBoolValue("startramSetReminderSeven", value)
		},
		confPatchValueRef[bool]{getTarget: func(patch *ConfPatch) **bool { return &patch.StartramReminderSeven }},
		applyStartramMutation(func(config *structs.StartramConfig, value bool) {
			config.StartramSetReminder.Seven = value
		}),
		nil,
	),
	newConfPatchField(
		"penpaiAllow",
		"PenpaiAllow",
		func(value interface{}) (bool, error) {
			return parseBoolValue("penpaiAllow", value)
		},
		confPatchValueRef[bool]{getTarget: func(patch *ConfPatch) **bool { return &patch.PenpaiAllow }},
		applyPenpaiMutation(func(config *structs.PenpaiConfig, value bool) {
			config.PenpaiAllow = value
		}),
		nil,
	),
	newConfPatchField(
		"gracefulExit",
		"GracefulExit",
		func(value interface{}) (bool, error) {
			return parseBoolValue("gracefulExit", value)
		},
		confPatchValueRef[bool]{getTarget: func(patch *ConfPatch) **bool { return &patch.GracefulExit }},
		applyRuntimeMutation(func(config *structs.RuntimeConfig, value bool) {
			config.GracefulExit = value
		}),
		nil,
	),
	newConfPatchField(
		"swapVal",
		"SwapVal",
		func(value interface{}) (int, error) {
			return parseIntValue("swapVal", value)
		},
		confPatchValueRef[int]{getTarget: func(patch *ConfPatch) **int { return &patch.SwapVal }},
		applyRuntimeMutation(func(config *structs.RuntimeConfig, value int) {
			config.SwapVal = value
		}),
		nil,
	),
	newConfPatchField(
		"penpaiRunning",
		"PenpaiRunning",
		func(value interface{}) (bool, error) {
			return parseBoolValue("penpaiRunning", value)
		},
		confPatchValueRef[bool]{getTarget: func(patch *ConfPatch) **bool { return &patch.PenpaiRunning }},
		applyPenpaiMutation(func(config *structs.PenpaiConfig, value bool) {
			config.PenpaiRunning = value
		}),
		nil,
	),
	newConfPatchField(
		"penpaiActive",
		"PenpaiActive",
		func(value interface{}) (string, error) {
			return parseStringValue("penpaiActive", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.PenpaiActive }},
		applyPenpaiMutation(func(config *structs.PenpaiConfig, value string) {
			config.PenpaiActive = value
		}),
		nil,
	),
	newConfPatchField(
		"penpaiCores",
		"PenpaiCores",
		func(value interface{}) (int, error) {
			return parseIntValue("penpaiCores", value)
		},
		confPatchValueRef[int]{getTarget: func(patch *ConfPatch) **int { return &patch.PenpaiCores }},
		applyPenpaiMutation(func(config *structs.PenpaiConfig, value int) {
			config.PenpaiCores = value
		}),
		nil,
	),
	newConfPatchField(
		"penpaiModels",
		"PenpaiModels",
		func(value interface{}) ([]structs.Penpai, error) {
			return parsePenpaiModels("penpaiModels", value)
		},
		confPatchValueAccessor(func(patch *ConfPatch) *[]structs.Penpai { return &patch.PenpaiModels }),
		applyPenpaiMutation(func(config *structs.PenpaiConfig, value []structs.Penpai) {
			config.PenpaiModels = append([]structs.Penpai(nil), value...)
		}),
		func(value []structs.Penpai) bool {
			return len(value) > 0
		},
	),
	newConfPatchField(
		"endpointUrl",
		"EndpointURL",
		func(value interface{}) (string, error) {
			return parseStringValue("endpointUrl", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.EndpointURL }},
		applyConnectivityMutation(func(config *structs.ConnectivityConfig, value string) {
			config.EndpointUrl = value
		}),
		nil,
	),
	newConfPatchField(
		"wgRegistered",
		"WgRegistered",
		func(value interface{}) (bool, error) {
			return parseBoolValue("wgRegistered", value)
		},
		confPatchValueRef[bool]{getTarget: func(patch *ConfPatch) **bool { return &patch.WgRegistered }},
		applyConnectivityMutation(func(config *structs.ConnectivityConfig, value bool) {
			config.WgRegistered = value
		}),
		nil,
	),
	newConfPatchField(
		"remoteBackupPassword",
		"RemoteBackupPassword",
		func(value interface{}) (string, error) {
			return parseStringValue("remoteBackupPassword", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.RemoteBackupPassword }},
		applyConnectivityMutation(func(config *structs.ConnectivityConfig, value string) {
			config.RemoteBackupPassword = value
		}),
		nil,
	),
	newConfPatchField(
		"netCheck",
		"NetCheck",
		func(value interface{}) (string, error) {
			return parseStringValue("netCheck", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.NetCheck }},
		applyConnectivityMutation(func(config *structs.ConnectivityConfig, value string) {
			config.NetCheck = value
		}),
		nil,
	),
	newConfPatchField(
		"updateMode",
		"UpdateMode",
		func(value interface{}) (string, error) {
			return parseStringValue("updateMode", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.UpdateMode }},
		applyConnectivityMutation(func(config *structs.ConnectivityConfig, value string) {
			config.UpdateMode = value
		}),
		nil,
	),
	newConfPatchField(
		"updateUrl",
		"UpdateUrl",
		func(value interface{}) (string, error) {
			return parseStringValue("updateUrl", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.UpdateUrl }},
		applyConnectivityMutation(func(config *structs.ConnectivityConfig, value string) {
			config.UpdateUrl = value
		}),
		nil,
	),
	newConfPatchField(
		"updateBranch",
		"UpdateBranch",
		func(value interface{}) (string, error) {
			return parseStringValue("updateBranch", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.UpdateBranch }},
		applyConnectivityMutation(func(config *structs.ConnectivityConfig, value string) {
			config.UpdateBranch = value
		}),
		nil,
	),
	newConfPatchField(
		"apiVersion",
		"ApiVersion",
		func(value interface{}) (string, error) {
			return parseStringValue("apiVersion", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.ApiVersion }},
		applyConnectivityMutation(func(config *structs.ConnectivityConfig, value string) {
			config.ApiVersion = value
		}),
		nil,
	),
	newConfPatchField(
		"pubkey",
		"Pubkey",
		func(value interface{}) (string, error) {
			return parseStringValue("pubkey", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.Pubkey }},
		applyStartramMutation(func(config *structs.StartramConfig, value string) {
			config.Pubkey = value
		}),
		nil,
	),
	newConfPatchField(
		"privkey",
		"Privkey",
		func(value interface{}) (string, error) {
			return parseStringValue("privkey", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.Privkey }},
		applyStartramMutation(func(config *structs.StartramConfig, value string) {
			config.Privkey = value
		}),
		nil,
	),
	newConfPatchField(
		"salt",
		"Salt",
		func(value interface{}) (string, error) {
			return parseStringValue("salt", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.Salt }},
		applyAuthSessionMutation(func(config *structs.AuthSessionConfig, value string) {
			config.Salt = value
		}),
		nil,
	),
	newConfPatchField(
		"keyFile",
		"KeyFile",
		func(value interface{}) (string, error) {
			return parseStringValue("keyFile", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.AuthSessionPatch.KeyFile }},
		applyAuthSessionMutation(func(config *structs.AuthSessionConfig, value string) {
			config.KeyFile = value
		}),
		nil,
	),
	newConfPatchField(
		"c2cInterval",
		"C2cInterval",
		func(value interface{}) (int, error) {
			return parseIntValue("c2cInterval", value)
		},
		confPatchValueRef[int]{getTarget: func(patch *ConfPatch) **int { return &patch.C2cInterval }},
		applyConnectivityMutation(func(config *structs.ConnectivityConfig, value int) {
			config.C2cInterval = value
		}),
		nil,
	),
	newConfPatchField(
		"swapFile",
		"SwapFile",
		func(value interface{}) (string, error) {
			return parseStringValue("swapFile", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.SwapFile }},
		applyRuntimeMutation(func(config *structs.RuntimeConfig, value string) {
			config.SwapFile = value
		}),
		nil,
	),
	newConfPatchField(
		"linuxUpdates",
		"LinuxUpdates",
		func(value interface{}) (struct {
			Value    int    `json:"value"`
			Interval string `json:"interval"`
		}, error) {
			return parseLinuxUpdatesValue("linuxUpdates", value)
		},
		confPatchValueRef[struct {
			Value    int    `json:"value"`
			Interval string `json:"interval"`
		}]{getTarget: func(patch *ConfPatch) **struct {
			Value    int    `json:"value"`
			Interval string `json:"interval"`
		} {
			return &patch.LinuxUpdates
		}},
		applyRuntimeMutation(func(config *structs.RuntimeConfig, value struct {
			Value    int    `json:"value"`
			Interval string `json:"interval"`
		}) {
			config.LinuxUpdates = value
		}),
		nil,
	),
	newConfPatchField(
		"dockerData",
		"DockerData",
		func(value interface{}) (string, error) {
			return parseStringValue("dockerData", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.DockerData }},
		applyRuntimeMutation(func(config *structs.RuntimeConfig, value string) {
			config.DockerData = value
		}),
		nil,
	),
	newConfPatchField(
		"setup",
		"Setup",
		func(value interface{}) (string, error) {
			return parseStringValue("setup", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.Setup }},
		applyRuntimeMutation(func(config *structs.RuntimeConfig, value string) {
			config.Setup = value
		}),
		nil,
	),
	newConfPatchField(
		"cfgDir",
		"CfgDir",
		func(value interface{}) (string, error) {
			return parseStringValue("cfgDir", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.CfgDir }},
		applyRuntimeMutation(func(config *structs.RuntimeConfig, value string) {
			config.CfgDir = value
		}),
		nil,
	),
	newConfPatchField(
		"updateInterval",
		"UpdateInterval",
		func(value interface{}) (int, error) {
			return parseIntValue("updateInterval", value)
		},
		confPatchValueRef[int]{getTarget: func(patch *ConfPatch) **int { return &patch.UpdateInterval }},
		applyRuntimeMutation(func(config *structs.RuntimeConfig, value int) {
			config.UpdateInterval = value
		}),
		nil,
	),
	newConfPatchField(
		"disable502",
		"Disable502",
		func(value interface{}) (bool, error) {
			return parseBoolValue("disable502", value)
		},
		confPatchValueRef[bool]{getTarget: func(patch *ConfPatch) **bool { return &patch.Disable502 }},
		applyRuntimeMutation(func(config *structs.RuntimeConfig, value bool) {
			config.Disable502 = value
		}),
		nil,
	),
	newConfPatchField(
		"snapTime",
		"SnapTime",
		func(value interface{}) (int, error) {
			return parseIntValue("snapTime", value)
		},
		confPatchValueRef[int]{getTarget: func(patch *ConfPatch) **int { return &patch.SnapTime }},
		applyRuntimeMutation(func(config *structs.RuntimeConfig, value int) {
			config.SnapTime = value
		}),
		nil,
	),
	newConfPatchField(
		"pwHash",
		"PwHash",
		func(value interface{}) (string, error) {
			return parseStringValue("pwHash", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.PwHash }},
		applyAuthSessionMutation(func(config *structs.AuthSessionConfig, value string) {
			config.PwHash = value
		}),
		nil,
	),
	newConfPatchField(
		"authorizedSessions",
		"AuthorizedSessions",
		func(value interface{}) (map[string]structs.SessionInfo, error) {
			return parseSessionMap("authorizedSessions", value)
		},
		confPatchValueAccessor(func(patch *ConfPatch) *map[string]structs.SessionInfo {
			return &patch.AuthorizedSessions
		}),
		applyAuthSessionMutation(func(config *structs.AuthSessionConfig, sessions map[string]structs.SessionInfo) {
			config.Sessions.Authorized = mergeSessions(config.Sessions.Authorized, sessions)
		}),
		func(value map[string]structs.SessionInfo) bool {
			return len(value) > 0
		},
	),
	newConfPatchField(
		"unauthorizedSessions",
		"UnauthorizedSessions",
		func(value interface{}) (map[string]structs.SessionInfo, error) {
			return parseSessionMap("unauthorizedSessions", value)
		},
		confPatchValueAccessor(func(patch *ConfPatch) *map[string]structs.SessionInfo {
			return &patch.UnauthorizedSessions
		}),
		applyAuthSessionMutation(func(config *structs.AuthSessionConfig, sessions map[string]structs.SessionInfo) {
			config.Sessions.Unauthorized = mergeSessions(config.Sessions.Unauthorized, sessions)
		}),
		func(value map[string]structs.SessionInfo) bool {
			return len(value) > 0
		},
	),
	newConfPatchField(
		"diskWarning",
		"DiskWarning",
		func(value interface{}) (map[string]structs.DiskWarning, error) {
			return parseDiskWarningMap("diskWarning", value)
		},
		confPatchValueRef[map[string]structs.DiskWarning]{getTarget: func(patch *ConfPatch) **map[string]structs.DiskWarning {
			return &patch.DiskWarning
		}},
		applyConnectivityMutation(func(config *structs.ConnectivityConfig, warnings map[string]structs.DiskWarning) {
			config.DiskWarning = copyDiskWarnings(warnings)
		}),
		func(warnings map[string]structs.DiskWarning) bool {
			return len(warnings) > 0
		},
	),
	newConfPatchField(
		"lastKnownMDNS",
		"LastKnownMDNS",
		func(value interface{}) (string, error) {
			return parseStringValue("lastKnownMDNS", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.LastKnownMDNS }},
		applyRuntimeMutation(func(config *structs.RuntimeConfig, value string) {
			config.LastKnownMDNS = value
		}),
		nil,
	),
	newConfPatchField(
		"disableSlsa",
		"DisableSlsa",
		func(value interface{}) (bool, error) {
			return parseBoolValue("disableSlsa", value)
		},
		confPatchValueRef[bool]{getTarget: func(patch *ConfPatch) **bool { return &patch.DisableSlsa }},
		applyStartramMutation(func(config *structs.StartramConfig, value bool) {
			config.DisableSlsa = value
		}),
		nil,
	),
	newConfPatchField(
		"gsVersion",
		"GSVersion",
		func(value interface{}) (string, error) {
			return parseStringValue("gsVersion", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.GSVersion }},
		applyRuntimeMutation(func(config *structs.RuntimeConfig, value string) {
			config.GsVersion = value
		}),
		nil,
	),
	newConfPatchField(
		"binHash",
		"BinHash",
		func(value interface{}) (string, error) {
			return parseStringValue("binHash", value)
		},
		confPatchValueRef[string]{getTarget: func(patch *ConfPatch) **string { return &patch.BinHash }},
		applyRuntimeMutation(func(config *structs.RuntimeConfig, value string) {
			config.BinHash = value
		}),
		nil,
	),

	unsupportedConfPatchField("isEMMCMachine"),
}
