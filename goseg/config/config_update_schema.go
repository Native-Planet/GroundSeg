package config

import (
	"fmt"

	"groundseg/structs"
)

type ConfUpdateOption func(*ConfPatch)

type ConfPatch struct {
	Piers                 *[]string
	WgOn                  *bool
	StartramReminderOne   *bool
	StartramReminderThree *bool
	StartramReminderSeven *bool
	PenpaiAllow           *bool
	GracefulExit          *bool
	SwapVal               *int
	PenpaiRunning         *bool
	PenpaiActive          *string
	PenpaiCores           *int
	EndpointURL           *string
	WgRegistered          *bool
	RemoteBackupPassword  *string
	Pubkey                *string
	Privkey               *string
	Salt                  *string
	KeyFile               *string
	C2cInterval           *int
	Setup                 *string
	PwHash                *string
	AuthorizedSessions    map[string]structs.SessionInfo
	UnauthorizedSessions  map[string]structs.SessionInfo
	DiskWarning           *map[string]structs.DiskWarning
	LastKnownMDNS         *string
	DisableSlsa           *bool
	GSVersion             *string
	BinHash               *string
}

func (patch *ConfPatch) hasUpdates() bool {
	for _, field := range confPatchRegistry {
		if field.hasUpdates(patch) {
			return true
		}
	}
	return false
}

type confPatchField struct {
	key        string
	hasUpdates func(*ConfPatch) bool
	parse      func(*ConfPatch, interface{}) error
	apply      func(*structs.SysConfig, *ConfPatch)
}

type confPatchParser[T any] func(string, interface{}) (T, error)

func scalarConfPatchField[T any](
	key string,
	parse confPatchParser[T],
	getPatch func(*ConfPatch) *T,
	setPatch func(*ConfPatch, T),
	applyConfig func(*structs.SysConfig, T),
) confPatchField {
	return confPatchField{
		key: key,
		hasUpdates: func(p *ConfPatch) bool {
			return getPatch(p) != nil
		},
		parse: func(p *ConfPatch, value interface{}) error {
			parsed, err := parse(key, value)
			if err != nil {
				return err
			}
			setPatch(p, parsed)
			return nil
		},
		apply: func(configStruct *structs.SysConfig, p *ConfPatch) {
			value := getPatch(p)
			if value == nil {
				return
			}
			applyConfig(configStruct, *value)
		},
	}
}

func boolConfPatchField(
	key string,
	getPatch func(*ConfPatch) *bool,
	setPatch func(*ConfPatch, bool),
	applyConfig func(*structs.SysConfig, bool),
) confPatchField {
	return scalarConfPatchField(key, parseBoolValue, getPatch, setPatch, applyConfig)
}

func intConfPatchField(
	key string,
	getPatch func(*ConfPatch) *int,
	setPatch func(*ConfPatch, int),
	applyConfig func(*structs.SysConfig, int),
) confPatchField {
	return scalarConfPatchField(key, parseIntValue, getPatch, setPatch, applyConfig)
}

func stringConfPatchField(
	key string,
	getPatch func(*ConfPatch) *string,
	setPatch func(*ConfPatch, string),
	applyConfig func(*structs.SysConfig, string),
) confPatchField {
	return scalarConfPatchField(key, parseStringValue, getPatch, setPatch, applyConfig)
}

func parseStringSliceField(_ string, value interface{}) ([]string, error) {
	return parseStringSlice(value)
}

func stringSliceConfPatchField(
	key string,
	getPatch func(*ConfPatch) *[]string,
	setPatch func(*ConfPatch, []string),
	applyConfig func(*structs.SysConfig, []string),
) confPatchField {
	return scalarConfPatchField(key, parseStringSliceField, getPatch, setPatch, applyConfig)
}

func mapConfPatchField[V any](
	key string,
	parse confPatchParser[map[string]V],
	getPatch func(*ConfPatch) map[string]V,
	setPatch func(*ConfPatch, map[string]V),
	applyConfig func(*structs.SysConfig, map[string]V),
) confPatchField {
	return confPatchField{
		key: key,
		hasUpdates: func(p *ConfPatch) bool {
			return len(getPatch(p)) > 0
		},
		parse: func(p *ConfPatch, value interface{}) error {
			parsed, err := parse(key, value)
			if err != nil {
				return err
			}
			setPatch(p, parsed)
			return nil
		},
		apply: func(configStruct *structs.SysConfig, p *ConfPatch) {
			updates := getPatch(p)
			if len(updates) == 0 {
				return
			}
			applyConfig(configStruct, updates)
		},
	}
}

func unsupportedConfPatchField(key string) confPatchField {
	return confPatchField{
		key: key,
		hasUpdates: func(*ConfPatch) bool {
			return false
		},
		parse: func(_ *ConfPatch, _ interface{}) error {
			return fmt.Errorf("unsupported config key: %s", key)
		},
		apply: func(*structs.SysConfig, *ConfPatch) {},
	}
}

func mergeSessions(target map[string]structs.SessionInfo, sessions map[string]structs.SessionInfo) map[string]structs.SessionInfo {
	if target == nil {
		target = make(map[string]structs.SessionInfo)
	}
	for tokenID, session := range sessions {
		target[tokenID] = session
	}
	return target
}

func copyDiskWarnings(warnings map[string]structs.DiskWarning) map[string]structs.DiskWarning {
	copied := make(map[string]structs.DiskWarning, len(warnings))
	for key, warning := range warnings {
		copied[key] = warning
	}
	return copied
}

var confPatchRegistry = []confPatchField{
	stringSliceConfPatchField(
		"piers",
		func(p *ConfPatch) *[]string { return p.Piers },
		func(p *ConfPatch, piers []string) {
			p.Piers = &piers
		},
		func(configStruct *structs.SysConfig, piers []string) {
			configStruct.Piers = append([]string(nil), piers...)
		},
	),
	boolConfPatchField(
		"wgOn",
		func(p *ConfPatch) *bool { return p.WgOn },
		func(p *ConfPatch, value bool) {
			p.WgOn = &value
		},
		func(configStruct *structs.SysConfig, value bool) {
			configStruct.WgOn = value
		},
	),
	boolConfPatchField(
		"startramSetReminderOne",
		func(p *ConfPatch) *bool { return p.StartramReminderOne },
		func(p *ConfPatch, value bool) {
			p.StartramReminderOne = &value
		},
		func(configStruct *structs.SysConfig, value bool) {
			configStruct.StartramSetReminder.One = value
		},
	),
	boolConfPatchField(
		"startramSetReminderThree",
		func(p *ConfPatch) *bool { return p.StartramReminderThree },
		func(p *ConfPatch, value bool) {
			p.StartramReminderThree = &value
		},
		func(configStruct *structs.SysConfig, value bool) {
			configStruct.StartramSetReminder.Three = value
		},
	),
	boolConfPatchField(
		"startramSetReminderSeven",
		func(p *ConfPatch) *bool { return p.StartramReminderSeven },
		func(p *ConfPatch, value bool) {
			p.StartramReminderSeven = &value
		},
		func(configStruct *structs.SysConfig, value bool) {
			configStruct.StartramSetReminder.Seven = value
		},
	),
	boolConfPatchField(
		"penpaiAllow",
		func(p *ConfPatch) *bool { return p.PenpaiAllow },
		func(p *ConfPatch, value bool) {
			p.PenpaiAllow = &value
		},
		func(configStruct *structs.SysConfig, value bool) {
			configStruct.PenpaiAllow = value
		},
	),
	boolConfPatchField(
		"gracefulExit",
		func(p *ConfPatch) *bool { return p.GracefulExit },
		func(p *ConfPatch, value bool) {
			p.GracefulExit = &value
		},
		func(configStruct *structs.SysConfig, value bool) {
			configStruct.GracefulExit = value
		},
	),
	intConfPatchField(
		"swapVal",
		func(p *ConfPatch) *int { return p.SwapVal },
		func(p *ConfPatch, value int) {
			p.SwapVal = &value
		},
		func(configStruct *structs.SysConfig, value int) {
			configStruct.SwapVal = value
		},
	),
	boolConfPatchField(
		"penpaiRunning",
		func(p *ConfPatch) *bool { return p.PenpaiRunning },
		func(p *ConfPatch, value bool) {
			p.PenpaiRunning = &value
		},
		func(configStruct *structs.SysConfig, value bool) {
			configStruct.PenpaiRunning = value
		},
	),
	stringConfPatchField(
		"penpaiActive",
		func(p *ConfPatch) *string { return p.PenpaiActive },
		func(p *ConfPatch, value string) {
			p.PenpaiActive = &value
		},
		func(configStruct *structs.SysConfig, value string) {
			configStruct.PenpaiActive = value
		},
	),
	intConfPatchField(
		"penpaiCores",
		func(p *ConfPatch) *int { return p.PenpaiCores },
		func(p *ConfPatch, value int) {
			p.PenpaiCores = &value
		},
		func(configStruct *structs.SysConfig, value int) {
			configStruct.PenpaiCores = value
		},
	),
	stringConfPatchField(
		"endpointUrl",
		func(p *ConfPatch) *string { return p.EndpointURL },
		func(p *ConfPatch, value string) {
			p.EndpointURL = &value
		},
		func(configStruct *structs.SysConfig, value string) {
			configStruct.EndpointUrl = value
		},
	),
	boolConfPatchField(
		"wgRegistered",
		func(p *ConfPatch) *bool { return p.WgRegistered },
		func(p *ConfPatch, value bool) {
			p.WgRegistered = &value
		},
		func(configStruct *structs.SysConfig, value bool) {
			configStruct.WgRegistered = value
		},
	),
	stringConfPatchField(
		"remoteBackupPassword",
		func(p *ConfPatch) *string { return p.RemoteBackupPassword },
		func(p *ConfPatch, value string) {
			p.RemoteBackupPassword = &value
		},
		func(configStruct *structs.SysConfig, value string) {
			configStruct.RemoteBackupPassword = value
		},
	),
	stringConfPatchField(
		"pubkey",
		func(p *ConfPatch) *string { return p.Pubkey },
		func(p *ConfPatch, value string) {
			p.Pubkey = &value
		},
		func(configStruct *structs.SysConfig, value string) {
			configStruct.Pubkey = value
		},
	),
	stringConfPatchField(
		"privkey",
		func(p *ConfPatch) *string { return p.Privkey },
		func(p *ConfPatch, value string) {
			p.Privkey = &value
		},
		func(configStruct *structs.SysConfig, value string) {
			configStruct.Privkey = value
		},
	),
	stringConfPatchField(
		"salt",
		func(p *ConfPatch) *string { return p.Salt },
		func(p *ConfPatch, value string) {
			p.Salt = &value
		},
		func(configStruct *structs.SysConfig, value string) {
			configStruct.Salt = value
		},
	),
	stringConfPatchField(
		"keyFile",
		func(p *ConfPatch) *string { return p.KeyFile },
		func(p *ConfPatch, value string) {
			p.KeyFile = &value
		},
		func(configStruct *structs.SysConfig, value string) {
			configStruct.KeyFile = value
		},
	),
	intConfPatchField(
		"c2cInterval",
		func(p *ConfPatch) *int { return p.C2cInterval },
		func(p *ConfPatch, value int) {
			p.C2cInterval = &value
		},
		func(configStruct *structs.SysConfig, value int) {
			configStruct.C2cInterval = value
		},
	),
	stringConfPatchField(
		"setup",
		func(p *ConfPatch) *string { return p.Setup },
		func(p *ConfPatch, value string) {
			p.Setup = &value
		},
		func(configStruct *structs.SysConfig, value string) {
			configStruct.Setup = value
		},
	),
	stringConfPatchField(
		"pwHash",
		func(p *ConfPatch) *string { return p.PwHash },
		func(p *ConfPatch, value string) {
			p.PwHash = &value
		},
		func(configStruct *structs.SysConfig, value string) {
			configStruct.PwHash = value
		},
	),
	mapConfPatchField(
		"authorizedSessions",
		parseSessionMap,
		func(p *ConfPatch) map[string]structs.SessionInfo {
			return p.AuthorizedSessions
		},
		func(p *ConfPatch, sessions map[string]structs.SessionInfo) {
			p.AuthorizedSessions = sessions
		},
		func(configStruct *structs.SysConfig, sessions map[string]structs.SessionInfo) {
			configStruct.Sessions.Authorized = mergeSessions(configStruct.Sessions.Authorized, sessions)
		},
	),
	mapConfPatchField(
		"unauthorizedSessions",
		parseSessionMap,
		func(p *ConfPatch) map[string]structs.SessionInfo {
			return p.UnauthorizedSessions
		},
		func(p *ConfPatch, sessions map[string]structs.SessionInfo) {
			p.UnauthorizedSessions = sessions
		},
		func(configStruct *structs.SysConfig, sessions map[string]structs.SessionInfo) {
			configStruct.Sessions.Unauthorized = mergeSessions(configStruct.Sessions.Unauthorized, sessions)
		},
	),
	mapConfPatchField(
		"diskWarning",
		func(name string, value interface{}) (map[string]structs.DiskWarning, error) {
			return parseDiskWarningMap(value)
		},
		func(p *ConfPatch) map[string]structs.DiskWarning {
			if p.DiskWarning == nil {
				return nil
			}
			return *p.DiskWarning
		},
		func(p *ConfPatch, warnings map[string]structs.DiskWarning) {
			copied := copyDiskWarnings(warnings)
			p.DiskWarning = &copied
		},
		func(configStruct *structs.SysConfig, warnings map[string]structs.DiskWarning) {
			configStruct.DiskWarning = copyDiskWarnings(warnings)
		},
	),
	stringConfPatchField(
		"lastKnownMDNS",
		func(p *ConfPatch) *string { return p.LastKnownMDNS },
		func(p *ConfPatch, value string) {
			p.LastKnownMDNS = &value
		},
		func(configStruct *structs.SysConfig, value string) {
			configStruct.LastKnownMDNS = value
		},
	),
	boolConfPatchField(
		"disableSlsa",
		func(p *ConfPatch) *bool { return p.DisableSlsa },
		func(p *ConfPatch, value bool) {
			p.DisableSlsa = &value
		},
		func(configStruct *structs.SysConfig, value bool) {
			configStruct.DisableSlsa = value
		},
	),
	stringConfPatchField(
		"gsVersion",
		func(p *ConfPatch) *string { return p.GSVersion },
		func(p *ConfPatch, value string) {
			p.GSVersion = &value
		},
		func(configStruct *structs.SysConfig, value string) {
			configStruct.GsVersion = value
		},
	),
	stringConfPatchField(
		"binHash",
		func(p *ConfPatch) *string { return p.BinHash },
		func(p *ConfPatch, value string) {
			p.BinHash = &value
		},
		func(configStruct *structs.SysConfig, value string) {
			configStruct.BinHash = value
		},
	),
	unsupportedConfPatchField("isEMMCMachine"),
}

var confPatchByKey = buildConfPatchByKey(confPatchRegistry)

func buildConfPatchByKey(fields []confPatchField) map[string]confPatchField {
	registry := make(map[string]confPatchField, len(fields))
	for _, field := range fields {
		registry[field.key] = field
	}
	return registry
}

func parseBoolValue(name string, value interface{}) (bool, error) {
	parsed, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("invalid %s value: %T", name, value)
	}
	return parsed, nil
}

func parseIntValue(name string, value interface{}) (int, error) {
	parsed, ok := value.(int)
	if !ok {
		return 0, fmt.Errorf("invalid %s value: %T", name, value)
	}
	return parsed, nil
}

func parseStringValue(name string, value interface{}) (string, error) {
	parsed, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("invalid %s value: %T", name, value)
	}
	return parsed, nil
}

func parseSessionMap(name string, value interface{}) (map[string]structs.SessionInfo, error) {
	sessions, ok := value.(map[string]structs.SessionInfo)
	if !ok {
		return nil, fmt.Errorf("invalid %s value: %T", name, value)
	}
	copied := make(map[string]structs.SessionInfo, len(sessions))
	for tokenID, session := range sessions {
		copied[tokenID] = session
	}
	return copied, nil
}

func parseStringSlice(value interface{}) ([]string, error) {
	if typed, ok := value.([]string); ok {
		return append([]string(nil), typed...), nil
	}
	typedInterface, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid piers value: %T", value)
	}
	piers := make([]string, 0, len(typedInterface))
	for _, item := range typedInterface {
		piers = append(piers, fmt.Sprint(item))
	}
	return piers, nil
}

func parseDiskWarningMap(value interface{}) (map[string]structs.DiskWarning, error) {
	rawWarnings, ok := value.(map[string]structs.DiskWarning)
	if !ok {
		return nil, fmt.Errorf("invalid diskWarning value: %T", value)
	}
	copied := make(map[string]structs.DiskWarning, len(rawWarnings))
	for key, warning := range rawWarnings {
		copied[key] = warning
	}
	return copied, nil
}

func buildConfigPatch(values map[string]interface{}) (*ConfPatch, error) {
	patch := &ConfPatch{}
	for key, value := range values {
		field, exists := confPatchByKey[key]
		if !exists {
			return nil, fmt.Errorf("unsupported config key: %s", key)
		}
		if err := field.parse(patch, value); err != nil {
			return nil, err
		}
	}
	return patch, nil
}
