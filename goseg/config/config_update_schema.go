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

type confPatchParser func(*ConfPatch, interface{}) error
type confPatchHasUpdateFn func(*ConfPatch) bool
type confPatchApplyFn func(*structs.SysConfig, *ConfPatch)

type confPatchField struct {
	key          string
	patchField   string
	parse        confPatchParser
	hasUpdates   confPatchHasUpdateFn
	applyUpdates confPatchApplyFn
}

func (patch *ConfPatch) hasUpdates() bool {
	for _, field := range confPatchRegistry {
		if field.hasUpdates(patch) {
			return true
		}
	}
	return false
}

func (field confPatchField) has(patch *ConfPatch) bool {
	if field.hasUpdates == nil {
		return false
	}
	return field.hasUpdates(patch)
}

func (field confPatchField) parsePatch(patch *ConfPatch, value interface{}) error {
	if field.parse == nil {
		return nil
	}
	return field.parse(patch, value)
}

func (field confPatchField) apply(config *structs.SysConfig, patch *ConfPatch) {
	if field.applyUpdates == nil || !field.has(patch) {
		return
	}
	field.applyUpdates(config, patch)
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
	{
		key:        "piers",
		patchField: "Piers",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseStringSliceValue("piers", value)
			if err != nil {
				return err
			}
			copied := append([]string(nil), parsed...)
			patch.Piers = &copied
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.Piers != nil && len(*patch.Piers) > 0
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.Piers = *patch.Piers
		},
	},
	{
		key:        "wgOn",
		patchField: "WgOn",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseBoolValue("wgOn", value)
			if err != nil {
				return err
			}
			parsedBool := parsed.(bool)
			patch.WgOn = &parsedBool
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.WgOn != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.WgOn = *patch.WgOn
		},
	},
	{
		key:        "startramSetReminderOne",
		patchField: "StartramReminderOne",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseBoolValue("startramSetReminderOne", value)
			if err != nil {
				return err
			}
			parsedBool := parsed.(bool)
			patch.StartramReminderOne = &parsedBool
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.StartramReminderOne != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.StartramSetReminder.One = *patch.StartramReminderOne
		},
	},
	{
		key:        "startramSetReminderThree",
		patchField: "StartramReminderThree",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseBoolValue("startramSetReminderThree", value)
			if err != nil {
				return err
			}
			parsedBool := parsed.(bool)
			patch.StartramReminderThree = &parsedBool
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.StartramReminderThree != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.StartramSetReminder.Three = *patch.StartramReminderThree
		},
	},
	{
		key:        "startramSetReminderSeven",
		patchField: "StartramReminderSeven",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseBoolValue("startramSetReminderSeven", value)
			if err != nil {
				return err
			}
			parsedBool := parsed.(bool)
			patch.StartramReminderSeven = &parsedBool
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.StartramReminderSeven != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.StartramSetReminder.Seven = *patch.StartramReminderSeven
		},
	},
	{
		key:        "penpaiAllow",
		patchField: "PenpaiAllow",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseBoolValue("penpaiAllow", value)
			if err != nil {
				return err
			}
			parsedBool := parsed.(bool)
			patch.PenpaiAllow = &parsedBool
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.PenpaiAllow != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.PenpaiAllow = *patch.PenpaiAllow
		},
	},
	{
		key:        "gracefulExit",
		patchField: "GracefulExit",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseBoolValue("gracefulExit", value)
			if err != nil {
				return err
			}
			parsedBool := parsed.(bool)
			patch.GracefulExit = &parsedBool
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.GracefulExit != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.GracefulExit = *patch.GracefulExit
		},
	},
	{
		key:        "swapVal",
		patchField: "SwapVal",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseIntValue("swapVal", value)
			if err != nil {
				return err
			}
			parsedInt := parsed.(int)
			patch.SwapVal = &parsedInt
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.SwapVal != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.SwapVal = *patch.SwapVal
		},
	},
	{
		key:        "penpaiRunning",
		patchField: "PenpaiRunning",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseBoolValue("penpaiRunning", value)
			if err != nil {
				return err
			}
			parsedBool := parsed.(bool)
			patch.PenpaiRunning = &parsedBool
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.PenpaiRunning != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.PenpaiRunning = *patch.PenpaiRunning
		},
	},
	{
		key:        "penpaiActive",
		patchField: "PenpaiActive",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseStringValue("penpaiActive", value)
			if err != nil {
				return err
			}
			parsedString := parsed.(string)
			patch.PenpaiActive = &parsedString
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.PenpaiActive != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.PenpaiActive = *patch.PenpaiActive
		},
	},
	{
		key:        "penpaiCores",
		patchField: "PenpaiCores",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseIntValue("penpaiCores", value)
			if err != nil {
				return err
			}
			parsedInt := parsed.(int)
			patch.PenpaiCores = &parsedInt
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.PenpaiCores != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.PenpaiCores = *patch.PenpaiCores
		},
	},
	{
		key:        "endpointUrl",
		patchField: "EndpointURL",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseStringValue("endpointUrl", value)
			if err != nil {
				return err
			}
			parsedString := parsed.(string)
			patch.EndpointURL = &parsedString
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.EndpointURL != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.EndpointUrl = *patch.EndpointURL
		},
	},
	{
		key:        "wgRegistered",
		patchField: "WgRegistered",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseBoolValue("wgRegistered", value)
			if err != nil {
				return err
			}
			parsedBool := parsed.(bool)
			patch.WgRegistered = &parsedBool
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.WgRegistered != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.WgRegistered = *patch.WgRegistered
		},
	},
	{
		key:        "remoteBackupPassword",
		patchField: "RemoteBackupPassword",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseStringValue("remoteBackupPassword", value)
			if err != nil {
				return err
			}
			parsedString := parsed.(string)
			patch.RemoteBackupPassword = &parsedString
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.RemoteBackupPassword != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.RemoteBackupPassword = *patch.RemoteBackupPassword
		},
	},
	{
		key:        "pubkey",
		patchField: "Pubkey",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseStringValue("pubkey", value)
			if err != nil {
				return err
			}
			parsedString := parsed.(string)
			patch.Pubkey = &parsedString
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.Pubkey != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.Pubkey = *patch.Pubkey
		},
	},
	{
		key:        "privkey",
		patchField: "Privkey",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseStringValue("privkey", value)
			if err != nil {
				return err
			}
			parsedString := parsed.(string)
			patch.Privkey = &parsedString
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.Privkey != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.Privkey = *patch.Privkey
		},
	},
	{
		key:        "salt",
		patchField: "Salt",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseStringValue("salt", value)
			if err != nil {
				return err
			}
			parsedString := parsed.(string)
			patch.Salt = &parsedString
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.Salt != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.Salt = *patch.Salt
		},
	},
	{
		key:        "keyFile",
		patchField: "KeyFile",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseStringValue("keyFile", value)
			if err != nil {
				return err
			}
			parsedString := parsed.(string)
			patch.KeyFile = &parsedString
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.KeyFile != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.KeyFile = *patch.KeyFile
		},
	},
	{
		key:        "c2cInterval",
		patchField: "C2cInterval",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseIntValue("c2cInterval", value)
			if err != nil {
				return err
			}
			parsedInt := parsed.(int)
			patch.C2cInterval = &parsedInt
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.C2cInterval != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.C2cInterval = *patch.C2cInterval
		},
	},
	{
		key:        "setup",
		patchField: "Setup",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseStringValue("setup", value)
			if err != nil {
				return err
			}
			parsedString := parsed.(string)
			patch.Setup = &parsedString
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.Setup != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.Setup = *patch.Setup
		},
	},
	{
		key:        "pwHash",
		patchField: "PwHash",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseStringValue("pwHash", value)
			if err != nil {
				return err
			}
			parsedString := parsed.(string)
			patch.PwHash = &parsedString
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.PwHash != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.PwHash = *patch.PwHash
		},
	},
	{
		key:        "authorizedSessions",
		patchField: "AuthorizedSessions",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseSessionMap("authorizedSessions", value)
			if err != nil {
				return err
			}
			patch.AuthorizedSessions = parsed.(map[string]structs.SessionInfo)
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return len(patch.AuthorizedSessions) > 0
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.Sessions.Authorized = mergeSessions(config.Sessions.Authorized, patch.AuthorizedSessions)
		},
	},
	{
		key:        "unauthorizedSessions",
		patchField: "UnauthorizedSessions",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseSessionMap("unauthorizedSessions", value)
			if err != nil {
				return err
			}
			patch.UnauthorizedSessions = parsed.(map[string]structs.SessionInfo)
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return len(patch.UnauthorizedSessions) > 0
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.Sessions.Unauthorized = mergeSessions(config.Sessions.Unauthorized, patch.UnauthorizedSessions)
		},
	},
	{
		key:        "diskWarning",
		patchField: "DiskWarning",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseDiskWarningMap("diskWarning", value)
			if err != nil {
				return err
			}
			parsedWarnings := parsed.(map[string]structs.DiskWarning)
			patch.DiskWarning = &parsedWarnings
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.DiskWarning != nil && len(*patch.DiskWarning) > 0
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.DiskWarning = copyDiskWarnings(*patch.DiskWarning)
		},
	},
	{
		key:        "lastKnownMDNS",
		patchField: "LastKnownMDNS",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseStringValue("lastKnownMDNS", value)
			if err != nil {
				return err
			}
			parsedString := parsed.(string)
			patch.LastKnownMDNS = &parsedString
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.LastKnownMDNS != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.LastKnownMDNS = *patch.LastKnownMDNS
		},
	},
	{
		key:        "disableSlsa",
		patchField: "DisableSlsa",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseBoolValue("disableSlsa", value)
			if err != nil {
				return err
			}
			parsedBool := parsed.(bool)
			patch.DisableSlsa = &parsedBool
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.DisableSlsa != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.DisableSlsa = *patch.DisableSlsa
		},
	},
	{
		key:        "gsVersion",
		patchField: "GSVersion",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseStringValue("gsVersion", value)
			if err != nil {
				return err
			}
			parsedString := parsed.(string)
			patch.GSVersion = &parsedString
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.GSVersion != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.GsVersion = *patch.GSVersion
		},
	},
	{
		key:        "binHash",
		patchField: "BinHash",
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseStringValue("binHash", value)
			if err != nil {
				return err
			}
			parsedString := parsed.(string)
			patch.BinHash = &parsedString
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			return patch.BinHash != nil
		},
		applyUpdates: func(config *structs.SysConfig, patch *ConfPatch) {
			config.BinHash = *patch.BinHash
		},
	},
	{
		key: "isEMMCMachine",
		parse: func(_ *ConfPatch, _ interface{}) error {
			return fmt.Errorf("unsupported config key: isEMMCMachine")
		},
		hasUpdates: func(_ *ConfPatch) bool {
			return false
		},
		applyUpdates: nil,
	},
}

var (
	confPatchByKey    = map[string]confPatchField{}
	confPatchByKeyErr error
)

func init() {
	var err error
	confPatchByKey, err = buildConfPatchByKey(confPatchRegistry)
	confPatchByKeyErr = err
}

func buildConfPatchByKey(fields []confPatchField) (map[string]confPatchField, error) {
	registry := make(map[string]confPatchField, len(fields))
	for _, field := range fields {
		if _, exists := registry[field.key]; exists {
			return nil, fmt.Errorf("duplicate config patch key: %s", field.key)
		}
		registry[field.key] = field
	}
	return registry, nil
}

func parseBoolValue(name string, value interface{}) (any, error) {
	parsed, ok := value.(bool)
	if !ok {
		return nil, fmt.Errorf("invalid %s value: %T", name, value)
	}
	return parsed, nil
}

func parseIntValue(name string, value interface{}) (any, error) {
	switch parsed := value.(type) {
	case int:
		return parsed, nil
	case float64:
		return int(parsed), nil
	default:
		return nil, fmt.Errorf("invalid %s value: %T", name, value)
	}
}

func parseStringValue(name string, value interface{}) (any, error) {
	parsed, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("invalid %s value: %T", name, value)
	}
	return parsed, nil
}

func parseStringSliceValue(_ string, value interface{}) ([]string, error) {
	return parseStringSlice(value)
}

func parseSessionMap(name string, value interface{}) (any, error) {
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

func parseDiskWarningMap(name string, value interface{}) (any, error) {
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
	if confPatchByKeyErr != nil {
		return nil, fmt.Errorf("invalid config patch registry: %w", confPatchByKeyErr)
	}
	patch := &ConfPatch{}
	for key, value := range values {
		field, exists := confPatchByKey[key]
		if !exists {
			return nil, fmt.Errorf("unsupported config key: %s", key)
		}
		if err := field.parsePatch(patch, value); err != nil {
			return nil, err
		}
	}
	return patch, nil
}
