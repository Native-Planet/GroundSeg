package config

import (
	"groundseg/structs"
	"reflect"
)

type confPatchFieldSpec[T any] struct {
	key        string
	patchField string
	parse      confPatchValueParser[T]
	apply      confPatchApplyValueFn[T]
	hasUpdates func(T) bool
}

func patchFieldRefByName[T any](field string) confPatchValueRef[T] {
	return confPatchValueRef[T]{
		getValue: func(patch *ConfPatch) (T, bool) {
			return getConfigPatchValueByName[T](patch, field)
		},
		setValue: func(patch *ConfPatch, value T) {
			setConfigPatchValueByName(patch, field, value)
		},
	}
}

func configPatchFieldFromSpec[T any](spec confPatchFieldSpec[T]) confPatchField {
	apply := spec.apply
	if apply == nil {
		panic("patch field spec requires explicit apply handler")
	}
	return newConfPatchField(
		spec.key,
		spec.patchField,
		spec.parse,
		patchFieldRefByName[T](spec.patchField),
		apply,
		spec.hasUpdates,
	)
}

type confPatchDomain string

const (
	confPatchDomainConnectivity confPatchDomain = "connectivity"
	confPatchDomainStartram     confPatchDomain = "startram"
	confPatchDomainRuntime      confPatchDomain = "runtime"
	confPatchDomainPenpai       confPatchDomain = "penpai"
	confPatchDomainAuthSession  confPatchDomain = "authSession"
	confPatchDomainUnsupported  confPatchDomain = "unsupported"
)

type confPatchDomainService struct {
	name   confPatchDomain
	fields []confPatchField
}

func (service confPatchDomainService) apply(target confPatchApplyTarget, patch *ConfPatch) {
	for _, field := range service.fields {
		field.apply(target, patch)
	}
}

func (service confPatchDomainService) hasUpdates(patch *ConfPatch) bool {
	for _, field := range service.fields {
		if field.hasUpdates(patch) {
			return true
		}
	}
	return false
}

func confPatchServices() []confPatchDomainService {
	return []confPatchDomainService{
		{name: confPatchDomainConnectivity, fields: connectivityPatchFields()},
		{name: confPatchDomainStartram, fields: startramPatchFields()},
		{name: confPatchDomainRuntime, fields: runtimePatchFields()},
		{name: confPatchDomainPenpai, fields: penpaiPatchFields()},
		{name: confPatchDomainAuthSession, fields: authSessionPatchFields()},
		{name: confPatchDomainUnsupported, fields: unsupportedPatchFields()},
	}
}

func parseBoolField(name string) confPatchValueParser[bool] {
	return func(value interface{}) (bool, error) {
		return parseBoolValue(name, value)
	}
}

func parseIntField(name string) confPatchValueParser[int] {
	return func(value interface{}) (int, error) {
		return parseIntValue(name, value)
	}
}

func parseStringField(name string) confPatchValueParser[string] {
	return func(value interface{}) (string, error) {
		return parseStringValue(name, value)
	}
}

func parseStringSliceField(name string) confPatchValueParser[[]string] {
	return func(value interface{}) ([]string, error) {
		return parseStringSliceValue(name, value)
	}
}

func applyConnectivityPatchValue[T any](setter func(*structs.ConnectivityConfig, T)) confPatchApplyValueFn[T] {
	return func(target confPatchApplyTarget, value T) {
		if target == nil {
			return
		}
		target.UpdateConnectivityConfig(func(config *structs.ConnectivityConfig) {
			setter(config, value)
		})
	}
}

func applyStartramPatchValue[T any](setter func(*structs.StartramConfig, T)) confPatchApplyValueFn[T] {
	return func(target confPatchApplyTarget, value T) {
		if target == nil {
			return
		}
		target.UpdateStartramConfig(func(config *structs.StartramConfig) {
			setter(config, value)
		})
	}
}

func applyRuntimePatchValue[T any](setter func(*structs.RuntimeConfig, T)) confPatchApplyValueFn[T] {
	return func(target confPatchApplyTarget, value T) {
		if target == nil {
			return
		}
		target.UpdateRuntimeConfig(func(config *structs.RuntimeConfig) {
			setter(config, value)
		})
	}
}

func applyPenpaiPatchValue[T any](setter func(*structs.PenpaiConfig, T)) confPatchApplyValueFn[T] {
	return func(target confPatchApplyTarget, value T) {
		if target == nil {
			return
		}
		target.UpdatePenpaiConfig(func(config *structs.PenpaiConfig) {
			setter(config, value)
		})
	}
}

func applyAuthSessionPatchValue[T any](setter func(*structs.AuthSessionConfig, T)) confPatchApplyValueFn[T] {
	return func(target confPatchApplyTarget, value T) {
		if target == nil {
			return
		}
		target.UpdateAuthSessionConfig(func(config *structs.AuthSessionConfig) {
			setter(config, value)
		})
	}
}

func getConfigPatchValueByName[T any](patch *ConfPatch, field string) (T, bool) {
	var zero T
	if patch == nil {
		return zero, false
	}
	patchValue := reflect.ValueOf(patch)
	if !patchValue.IsValid() || patchValue.Kind() != reflect.Ptr || patchValue.IsNil() {
		return zero, false
	}

	fieldValue := patchValue.Elem().FieldByName(field)
	if !fieldValue.IsValid() {
		return zero, false
	}
	if fieldValue.Kind() == reflect.Ptr {
		if fieldValue.IsNil() {
			return zero, false
		}
		fieldValue = fieldValue.Elem()
	}
	typed, ok := fieldValue.Interface().(T)
	return typed, ok
}

func setConfigPatchValueByName[T any](patch *ConfPatch, field string, value T) {
	if patch == nil {
		return
	}
	patchValue := reflect.ValueOf(patch)
	if !patchValue.IsValid() || patchValue.Kind() != reflect.Ptr || patchValue.IsNil() {
		return
	}

	fieldValue := patchValue.Elem().FieldByName(field)
	if !fieldValue.IsValid() || !fieldValue.CanSet() {
		return
	}

	raw := reflect.ValueOf(value)
	if !raw.IsValid() {
		return
	}

	if fieldValue.Kind() == reflect.Ptr {
		pointer := reflect.New(fieldValue.Type().Elem())
		if !assignValueToField(pointer.Elem(), raw) {
			if raw.Type().ConvertibleTo(pointer.Elem().Type()) {
				assignValueToField(pointer.Elem(), raw.Convert(pointer.Elem().Type()))
			} else {
				return
			}
		}
		fieldValue.Set(pointer)
		return
	}

	if !assignValueToField(fieldValue, raw) {
		if raw.Type().ConvertibleTo(fieldValue.Type()) {
			fieldValue.Set(raw.Convert(fieldValue.Type()))
		}
	}
}

func assignValueToField(target reflect.Value, value reflect.Value) bool {
	if !target.IsValid() || !value.IsValid() {
		return false
	}
	if !target.CanSet() {
		return false
	}

	if value.Type().AssignableTo(target.Type()) {
		target.Set(value)
		return true
	}
	if value.Type().ConvertibleTo(target.Type()) {
		target.Set(value.Convert(target.Type()))
		return true
	}
	return false
}

func connectivityPatchFields() []confPatchField {
	return []confPatchField{
		configPatchFieldFromSpec(confPatchFieldSpec[[]string]{
			key:        "piers",
			patchField: "Piers",
			parse:      parseStringSliceField("piers"),
			apply: applyConnectivityPatchValue(func(config *structs.ConnectivityConfig, value []string) {
				config.Piers = append([]string(nil), value...)
			}),
			hasUpdates: func(value []string) bool {
				return len(value) > 0
			},
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[bool]{
			key:        "wgOn",
			patchField: "WgOn",
			parse:      parseBoolField("wgOn"),
			apply:      applyConnectivityPatchValue(func(config *structs.ConnectivityConfig, value bool) { config.WgOn = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "endpointUrl",
			patchField: "EndpointURL",
			parse:      parseStringField("endpointUrl"),
			apply:      applyConnectivityPatchValue(func(config *structs.ConnectivityConfig, value string) { config.EndpointUrl = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[bool]{
			key:        "wgRegistered",
			patchField: "WgRegistered",
			parse:      parseBoolField("wgRegistered"),
			apply:      applyConnectivityPatchValue(func(config *structs.ConnectivityConfig, value bool) { config.WgRegistered = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "remoteBackupPassword",
			patchField: "RemoteBackupPassword",
			parse:      parseStringField("remoteBackupPassword"),
			apply: applyConnectivityPatchValue(func(config *structs.ConnectivityConfig, value string) {
				config.RemoteBackupPassword = value
			}),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "netCheck",
			patchField: "NetCheck",
			parse:      parseStringField("netCheck"),
			apply:      applyConnectivityPatchValue(func(config *structs.ConnectivityConfig, value string) { config.NetCheck = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "updateMode",
			patchField: "UpdateMode",
			parse:      parseStringField("updateMode"),
			apply:      applyConnectivityPatchValue(func(config *structs.ConnectivityConfig, value string) { config.UpdateMode = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "updateUrl",
			patchField: "UpdateUrl",
			parse:      parseStringField("updateUrl"),
			apply:      applyConnectivityPatchValue(func(config *structs.ConnectivityConfig, value string) { config.UpdateUrl = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "updateBranch",
			patchField: "UpdateBranch",
			parse:      parseStringField("updateBranch"),
			apply:      applyConnectivityPatchValue(func(config *structs.ConnectivityConfig, value string) { config.UpdateBranch = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "apiVersion",
			patchField: "ApiVersion",
			parse:      parseStringField("apiVersion"),
			apply:      applyConnectivityPatchValue(func(config *structs.ConnectivityConfig, value string) { config.ApiVersion = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[int]{
			key:        "c2cInterval",
			patchField: "C2cInterval",
			parse:      parseIntField("c2cInterval"),
			apply:      applyConnectivityPatchValue(func(config *structs.ConnectivityConfig, value int) { config.C2cInterval = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[map[string]structs.DiskWarning]{
			key:        "diskWarning",
			patchField: "DiskWarning",
			parse:      parseDiskWarningMapField("diskWarning"),
			hasUpdates: func(warnings map[string]structs.DiskWarning) bool {
				return len(warnings) > 0
			},
			apply: func(target confPatchApplyTarget, warnings map[string]structs.DiskWarning) {
				target.UpdateConnectivityConfig(func(config *structs.ConnectivityConfig) {
					config.DiskWarning = copyDiskWarnings(warnings)
				})
			},
		}),
	}
}

func startramPatchFields() []confPatchField {
	return []confPatchField{
		configPatchFieldFromSpec(confPatchFieldSpec[bool]{
			key:        "startramSetReminderOne",
			patchField: "StartramReminderOne",
			parse:      parseBoolField("startramSetReminderOne"),
			apply: applyStartramPatchValue(func(config *structs.StartramConfig, value bool) {
				config.StartramSetReminder.One = value
			}),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[bool]{
			key:        "startramSetReminderThree",
			patchField: "StartramReminderThree",
			parse:      parseBoolField("startramSetReminderThree"),
			apply: applyStartramPatchValue(func(config *structs.StartramConfig, value bool) {
				config.StartramSetReminder.Three = value
			}),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[bool]{
			key:        "startramSetReminderSeven",
			patchField: "StartramReminderSeven",
			parse:      parseBoolField("startramSetReminderSeven"),
			apply: applyStartramPatchValue(func(config *structs.StartramConfig, value bool) {
				config.StartramSetReminder.Seven = value
			}),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "pubkey",
			patchField: "Pubkey",
			parse:      parseStringField("pubkey"),
			apply:      applyStartramPatchValue(func(config *structs.StartramConfig, value string) { config.Pubkey = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "privkey",
			patchField: "Privkey",
			parse:      parseStringField("privkey"),
			apply:      applyStartramPatchValue(func(config *structs.StartramConfig, value string) { config.Privkey = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[bool]{
			key:        "disableSlsa",
			patchField: "DisableSlsa",
			parse:      parseBoolField("disableSlsa"),
			apply:      applyStartramPatchValue(func(config *structs.StartramConfig, value bool) { config.DisableSlsa = value }),
		}),
	}
}

func runtimePatchFields() []confPatchField {
	return []confPatchField{
		configPatchFieldFromSpec(confPatchFieldSpec[bool]{
			key:        "gracefulExit",
			patchField: "GracefulExit",
			parse:      parseBoolField("gracefulExit"),
			apply:      applyRuntimePatchValue(func(config *structs.RuntimeConfig, value bool) { config.GracefulExit = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[int]{
			key:        "swapVal",
			patchField: "SwapVal",
			parse:      parseIntField("swapVal"),
			apply:      applyRuntimePatchValue(func(config *structs.RuntimeConfig, value int) { config.SwapVal = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "swapFile",
			patchField: "SwapFile",
			parse:      parseStringField("swapFile"),
			apply:      applyRuntimePatchValue(func(config *structs.RuntimeConfig, value string) { config.SwapFile = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[linuxUpdatesPatch]{
			key:        "linuxUpdates",
			patchField: "LinuxUpdates",
			parse: func(value interface{}) (linuxUpdatesPatch, error) {
				return parseLinuxUpdatesValue("linuxUpdates", value)
			},
			apply: applyRuntimePatchValue(func(config *structs.RuntimeConfig, value linuxUpdatesPatch) {
				config.LinuxUpdates = struct {
					Value    int    `json:"value"`
					Interval string `json:"interval"`
				}{
					Value:    value.Value,
					Interval: value.Interval,
				}
			}),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "dockerData",
			patchField: "DockerData",
			parse:      parseStringField("dockerData"),
			apply:      applyRuntimePatchValue(func(config *structs.RuntimeConfig, value string) { config.DockerData = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "setup",
			patchField: "Setup",
			parse:      parseStringField("setup"),
			apply:      applyRuntimePatchValue(func(config *structs.RuntimeConfig, value string) { config.Setup = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "cfgDir",
			patchField: "CfgDir",
			parse:      parseStringField("cfgDir"),
			apply:      applyRuntimePatchValue(func(config *structs.RuntimeConfig, value string) { config.CfgDir = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[int]{
			key:        "updateInterval",
			patchField: "UpdateInterval",
			parse:      parseIntField("updateInterval"),
			apply:      applyRuntimePatchValue(func(config *structs.RuntimeConfig, value int) { config.UpdateInterval = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[bool]{
			key:        "disable502",
			patchField: "Disable502",
			parse:      parseBoolField("disable502"),
			apply:      applyRuntimePatchValue(func(config *structs.RuntimeConfig, value bool) { config.Disable502 = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[int]{
			key:        "snapTime",
			patchField: "SnapTime",
			parse:      parseIntField("snapTime"),
			apply:      applyRuntimePatchValue(func(config *structs.RuntimeConfig, value int) { config.SnapTime = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "lastKnownMDNS",
			patchField: "LastKnownMDNS",
			parse:      parseStringField("lastKnownMDNS"),
			apply:      applyRuntimePatchValue(func(config *structs.RuntimeConfig, value string) { config.LastKnownMDNS = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "gsVersion",
			patchField: "GSVersion",
			parse:      parseStringField("gsVersion"),
			apply:      applyRuntimePatchValue(func(config *structs.RuntimeConfig, value string) { config.GsVersion = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "binHash",
			patchField: "BinHash",
			parse:      parseStringField("binHash"),
			apply:      applyRuntimePatchValue(func(config *structs.RuntimeConfig, value string) { config.BinHash = value }),
		}),
	}
}

func penpaiPatchFields() []confPatchField {
	return []confPatchField{
		configPatchFieldFromSpec(confPatchFieldSpec[bool]{
			key:        "penpaiAllow",
			patchField: "PenpaiAllow",
			parse:      parseBoolField("penpaiAllow"),
			apply:      applyPenpaiPatchValue(func(config *structs.PenpaiConfig, value bool) { config.PenpaiAllow = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[bool]{
			key:        "penpaiRunning",
			patchField: "PenpaiRunning",
			parse:      parseBoolField("penpaiRunning"),
			apply:      applyPenpaiPatchValue(func(config *structs.PenpaiConfig, value bool) { config.PenpaiRunning = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[int]{
			key:        "penpaiCores",
			patchField: "PenpaiCores",
			parse:      parseIntField("penpaiCores"),
			apply:      applyPenpaiPatchValue(func(config *structs.PenpaiConfig, value int) { config.PenpaiCores = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "penpaiActive",
			patchField: "PenpaiActive",
			parse:      parseStringField("penpaiActive"),
			apply:      applyPenpaiPatchValue(func(config *structs.PenpaiConfig, value string) { config.PenpaiActive = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[[]structs.Penpai]{
			key:        "penpaiModels",
			patchField: "PenpaiModels",
			parse: func(value interface{}) ([]structs.Penpai, error) {
				return parsePenpaiModels("penpaiModels", value)
			},
			apply: func(target confPatchApplyTarget, value []structs.Penpai) {
				target.UpdatePenpaiConfig(func(config *structs.PenpaiConfig) {
					config.PenpaiModels = append([]structs.Penpai(nil), value...)
				})
			},
			hasUpdates: func(value []structs.Penpai) bool {
				return len(value) > 0
			},
		}),
	}
}

func authSessionPatchFields() []confPatchField {
	return []confPatchField{
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "salt",
			patchField: "Salt",
			parse:      parseStringField("salt"),
			apply:      applyAuthSessionPatchValue(func(config *structs.AuthSessionConfig, value string) { config.Salt = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "keyFile",
			patchField: "KeyFile",
			parse:      parseStringField("keyFile"),
			apply:      applyAuthSessionPatchValue(func(config *structs.AuthSessionConfig, value string) { config.KeyFile = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[string]{
			key:        "pwHash",
			patchField: "PwHash",
			parse:      parseStringField("pwHash"),
			apply:      applyAuthSessionPatchValue(func(config *structs.AuthSessionConfig, value string) { config.PwHash = value }),
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[map[string]structs.SessionInfo]{
			key:        "authorizedSessions",
			patchField: "AuthorizedSessions",
			parse:      parseSessionMapField("authorizedSessions"),
			apply: func(target confPatchApplyTarget, sessions map[string]structs.SessionInfo) {
				target.UpdateAuthSessionConfig(func(config *structs.AuthSessionConfig) {
					config.Sessions.Authorized = mergeSessions(config.Sessions.Authorized, sessions)
				})
			},
			hasUpdates: func(value map[string]structs.SessionInfo) bool {
				return len(value) > 0
			},
		}),
		configPatchFieldFromSpec(confPatchFieldSpec[map[string]structs.SessionInfo]{
			key:        "unauthorizedSessions",
			patchField: "UnauthorizedSessions",
			parse:      parseSessionMapField("unauthorizedSessions"),
			apply: func(target confPatchApplyTarget, sessions map[string]structs.SessionInfo) {
				target.UpdateAuthSessionConfig(func(config *structs.AuthSessionConfig) {
					config.Sessions.Unauthorized = mergeSessions(config.Sessions.Unauthorized, sessions)
				})
			},
			hasUpdates: func(value map[string]structs.SessionInfo) bool {
				return len(value) > 0
			},
		}),
	}
}

func unsupportedPatchFields() []confPatchField {
	return []confPatchField{unsupportedConfPatchField("isEMMCMachine")}
}

func parseDiskWarningMapField(name string) confPatchValueParser[map[string]structs.DiskWarning] {
	return func(value interface{}) (map[string]structs.DiskWarning, error) {
		return parseDiskWarningMap(name, value)
	}
}

func parseSessionMapField(name string) confPatchValueParser[map[string]structs.SessionInfo] {
	return func(value interface{}) (map[string]structs.SessionInfo, error) {
		return parseSessionMap(name, value)
	}
}

func mergeConfPatchDomainFields(services ...confPatchDomainService) []confPatchField {
	total := 0
	for _, service := range services {
		total += len(service.fields)
	}
	fields := make([]confPatchField, 0, total+1)
	for _, service := range services {
		fields = append(fields, service.fields...)
	}
	return fields
}

func allConfPatchFields() []confPatchField {
	return mergeConfPatchDomainFields(confPatchServices()...)
}
