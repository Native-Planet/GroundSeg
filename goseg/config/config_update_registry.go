package config

import (
	"fmt"
	"groundseg/structs"
	"strings"
)

type configPatchConfigFieldRef[T any] func(*structs.SysConfig) *T

type configPatchFieldBinding[T any] struct {
	key         string
	patchField  string
	parse       configPatchValueParser[T]
	target      configPatchValueRef[T]
	configField configPatchConfigFieldRef[T]
	apply       configPatchApplyValueFn[T]
	hasUpdates  func(T) bool
	merge       configPatchMergeFn
}

func sectionField[Section any, T any](section func(*structs.SysConfig) *Section, field func(*Section) *T) configPatchConfigFieldRef[T] {
	return func(cfg *structs.SysConfig) *T {
		return field(section(cfg))
	}
}

func applyPatchField[T any](field configPatchConfigFieldRef[T]) configPatchApplyValueFn[T] {
	return func(target configPatchApplyTarget, value T) error {
		if target == nil || field == nil {
			return nil
		}
		destination := field(target)
		if destination == nil {
			return nil
		}
		*destination = value
		return nil
	}
}

func parseStringField(key string) configPatchValueParser[string] {
	return func(value interface{}) (string, error) {
		return parseStringValue(key, value)
	}
}

func parseBoolField(key string) configPatchValueParser[bool] {
	return func(value interface{}) (bool, error) {
		return parseBoolValue(key, value)
	}
}

func parseIntField(key string) configPatchValueParser[int] {
	return func(value interface{}) (int, error) {
		return parseIntValue(key, value)
	}
}

func parseStringSliceField(key string) configPatchValueParser[[]string] {
	return func(value interface{}) ([]string, error) {
		return parseStringSliceValue(key, value)
	}
}

func configPatchFieldFromBinding[T any](spec configPatchFieldBinding[T]) configPatchField {
	apply := spec.apply
	if apply == nil {
		apply = applyPatchField(spec.configField)
	}
	if apply == nil {
		return configPatchField{
			key:        spec.key,
			patchField: spec.patchField,
			initErr:    fmt.Errorf("patch field binding requires explicit apply handler for %q", spec.key),
			hasUpdates: func(_ *ConfPatch) bool {
				return false
			},
		}
	}
	if spec.target.setValue == nil && spec.target.getValue == nil {
		return configPatchField{
			key:        spec.key,
			patchField: spec.patchField,
			initErr:    fmt.Errorf("patch field binding requires explicit target accessor for %q", spec.key),
			hasUpdates: func(_ *ConfPatch) bool {
				return false
			},
		}
	}
	return newConfigPatchField(
		spec.key,
		spec.patchField,
		spec.parse,
		spec.target,
		apply,
		spec.hasUpdates,
		spec.merge,
	)
}

type configPatchFieldOptions[T any] struct {
	apply      configPatchApplyValueFn[T]
	hasUpdates func(T) bool
	merge      configPatchMergeFn
}

type configPatchSectionBuilder[Section any] struct {
	configSection func(*structs.SysConfig) *Section
}

func newConfigPatchSection[Section any](section func(*structs.SysConfig) *Section) configPatchSectionBuilder[Section] {
	return configPatchSectionBuilder[Section]{
		configSection: section,
	}
}

func patchConfigFieldFromSection[Section any, T any](
	builder configPatchSectionBuilder[Section],
	key string,
	patchField string,
	target configPatchValueRef[T],
	parse configPatchValueParser[T],
	cfgField func(*Section) *T,
	options configPatchFieldOptions[T],
) configPatchField {
	configField := sectionField(builder.configSection, cfgField)
	return configPatchFieldFromBinding(configPatchFieldBinding[T]{
		key:         key,
		patchField:  patchField,
		target:      target,
		parse:       parse,
		configField: configField,
		apply:       options.apply,
		hasUpdates:  options.hasUpdates,
		merge:       options.merge,
	})
}

func (builder configPatchSectionBuilder[Section]) boolField(
	key string,
	patchField string,
	target configPatchValueRef[bool],
	cfgField func(*Section) *bool,
	options configPatchFieldOptions[bool],
) configPatchField {
	return patchConfigFieldFromSection(builder, key, patchField, target, parseBoolField(key), cfgField, options)
}

func (builder configPatchSectionBuilder[Section]) intField(
	key string,
	patchField string,
	target configPatchValueRef[int],
	cfgField func(*Section) *int,
	options configPatchFieldOptions[int],
) configPatchField {
	return patchConfigFieldFromSection(builder, key, patchField, target, parseIntField(key), cfgField, options)
}

func (builder configPatchSectionBuilder[Section]) stringSliceField(
	key string,
	patchField string,
	target configPatchValueRef[[]string],
	cfgField func(*Section) *[]string,
	options configPatchFieldOptions[[]string],
) configPatchField {
	return patchConfigFieldFromSection(builder, key, patchField, target, parseStringSliceField(key), cfgField, options)
}

func (builder configPatchSectionBuilder[Section]) stringField(
	key string,
	patchField string,
	target configPatchValueRef[string],
	cfgField func(*Section) *string,
	options configPatchFieldOptions[string],
) configPatchField {
	return patchConfigFieldFromSection(builder, key, patchField, target, parseStringField(key), cfgField, options)
}

func (builder configPatchSectionBuilder[Section]) boolSetField(cfgField func(*Section) *bool) configPatchMergeFn {
	return mergeBoolSetRef(sectionField(builder.configSection, cfgField))
}

func (builder configPatchSectionBuilder[Section]) boolSetDirectField(cfgField func(*Section) *bool) configPatchMergeFn {
	return mergeBoolSetDirectRef(sectionField(builder.configSection, cfgField))
}

func (builder configPatchSectionBuilder[Section]) stringSetIfNonEmptyField(cfgField func(*Section) *string) configPatchMergeFn {
	return mergeStringSetIfNonEmptyRef(sectionField(builder.configSection, cfgField))
}

func (builder configPatchSectionBuilder[Section]) stringSetDirectField(cfgField func(*Section) *string) configPatchMergeFn {
	return mergeStringSetDirectRef(sectionField(builder.configSection, cfgField))
}

func (builder configPatchSectionBuilder[Section]) intSetIfNonZeroField(cfgField func(*Section) *int) configPatchMergeFn {
	return mergeIntSetIfNonZeroRef(sectionField(builder.configSection, cfgField))
}

func (builder configPatchSectionBuilder[Section]) stringSliceIfNonEmptyField(cfgField func(*Section) *[]string) configPatchMergeFn {
	return mergeStringSliceSetIfNonEmptyRef(sectionField(builder.configSection, cfgField))
}

func mergeIfString(configField configPatchConfigFieldRef[string], shouldMerge func(string) bool) configPatchMergeFn {
	return func(_ structs.SysConfig, customConfig structs.SysConfig, mergedConfig *structs.SysConfig) {
		if shouldMerge == nil {
			shouldMerge = func(_ string) bool { return true }
		}
		value := configField(&customConfig)
		if value == nil || !shouldMerge(*value) {
			return
		}
		destination := configField(mergedConfig)
		if destination != nil {
			*destination = *value
		}
	}
}

func mergeIfInt(configField configPatchConfigFieldRef[int], shouldMerge func(int) bool) configPatchMergeFn {
	return func(_ structs.SysConfig, customConfig structs.SysConfig, mergedConfig *structs.SysConfig) {
		if shouldMerge == nil {
			shouldMerge = func(_ int) bool { return true }
		}
		value := configField(&customConfig)
		if value == nil || !shouldMerge(*value) {
			return
		}
		destination := configField(mergedConfig)
		if destination != nil {
			*destination = *value
		}
	}
}

func mergeIfBool(configField configPatchConfigFieldRef[bool], shouldMerge func(bool) bool) configPatchMergeFn {
	return func(_ structs.SysConfig, customConfig structs.SysConfig, mergedConfig *structs.SysConfig) {
		if shouldMerge == nil {
			shouldMerge = func(_ bool) bool { return true }
		}
		value := configField(&customConfig)
		if value == nil || !shouldMerge(*value) {
			return
		}
		destination := configField(mergedConfig)
		if destination != nil {
			*destination = *destination || *value
		}
	}
}

func mergeSliceWhen[T any](configField configPatchConfigFieldRef[[]T], shouldMerge func([]T) bool) configPatchMergeFn {
	return func(_ structs.SysConfig, customConfig structs.SysConfig, mergedConfig *structs.SysConfig) {
		if shouldMerge == nil {
			shouldMerge = func(_ []T) bool { return true }
		}
		value := configField(&customConfig)
		if value == nil || !shouldMerge(*value) {
			return
		}
		destination := configField(mergedConfig)
		if destination != nil {
			copied := append([]T(nil), (*value)...)
			*destination = copied
		}
	}
}

func mergeStringSetIfNonEmptyRef(configField configPatchConfigFieldRef[string]) configPatchMergeFn {
	return mergeIfString(configField, func(value string) bool { return value != "" })
}

func mergeStringSetDirectRef(configField configPatchConfigFieldRef[string]) configPatchMergeFn {
	return mergeIfString(configField, nil)
}

func mergeIntSetIfNonZeroRef(configField configPatchConfigFieldRef[int]) configPatchMergeFn {
	return mergeIfInt(configField, func(value int) bool { return value != 0 })
}

func mergeBoolSetRef(configField configPatchConfigFieldRef[bool]) configPatchMergeFn {
	return mergeIfBool(configField, nil)
}

func mergeBoolSetDirectRef(configField configPatchConfigFieldRef[bool]) configPatchMergeFn {
	return func(_ structs.SysConfig, customConfig structs.SysConfig, mergedConfig *structs.SysConfig) {
		value := configField(&customConfig)
		if value == nil {
			return
		}
		destination := configField(mergedConfig)
		if destination != nil {
			*destination = *value
		}
	}
}

func mergeStringSliceSetIfNonEmptyRef(configField configPatchConfigFieldRef[[]string]) configPatchMergeFn {
	return mergeSliceWhen(configField, func(value []string) bool { return len(value) > 0 })
}

func mergeDiskWarning(_ structs.SysConfig, customConfig structs.SysConfig, mergedConfig *structs.SysConfig) {
	mergedConfig.Connectivity.DiskWarning = customConfig.Connectivity.DiskWarning
}

func mergeLinuxUpdates(_ structs.SysConfig, customConfig structs.SysConfig, mergedConfig *structs.SysConfig) {
	if customConfig.Runtime.LinuxUpdates.Value == 0 {
		return
	}
	mergedConfig.Runtime.LinuxUpdates = customConfig.Runtime.LinuxUpdates
}

func mergeSetupFromConfig(_ structs.SysConfig, customConfig structs.SysConfig, mergedConfig *structs.SysConfig) {
	if customConfig.AuthSession.PwHash == "" {
		mergedConfig.Runtime.Setup = "start"
		salt, err := RandStringWithError(32)
		if err == nil {
			mergedConfig.AuthSession.Salt = salt
		}
		return
	}

	if customConfig.Runtime.Setup == "" {
		mergedConfig.Runtime.Setup = "complete"
	} else {
		mergedConfig.Runtime.Setup = customConfig.Runtime.Setup
	}
}

func mergeAuthSessionSalt(_ structs.SysConfig, customConfig structs.SysConfig, mergedConfig *structs.SysConfig) {
	if customConfig.AuthSession.Salt != "" {
		mergedConfig.AuthSession.Salt = customConfig.AuthSession.Salt
	}
}

func mergeAuthSessionAuthorizedSessions(_ structs.SysConfig, customConfig structs.SysConfig, mergedConfig *structs.SysConfig) {
	if customConfig.AuthSession.Sessions.Authorized != nil {
		mergedConfig.AuthSession.Sessions.Authorized = customConfig.AuthSession.Sessions.Authorized
	}
}

func mergeAuthSessionUnauthorizedSessions(_ structs.SysConfig, customConfig structs.SysConfig, mergedConfig *structs.SysConfig) {
	if customConfig.AuthSession.Sessions.Unauthorized != nil {
		mergedConfig.AuthSession.Sessions.Unauthorized = customConfig.AuthSession.Sessions.Unauthorized
	}
}

func mergeDefaultModels(defaultConfig structs.SysConfig, _ structs.SysConfig, mergedConfig *structs.SysConfig) {
	mergedConfig.Penpai.PenpaiModels = append([]structs.Penpai(nil), defaultConfig.Penpai.PenpaiModels...)
}

func mergePenpaiActive(_ structs.SysConfig, customConfig structs.SysConfig, mergedConfig *structs.SysConfig) {
	if customConfig.Penpai.PenpaiActive == "" {
		return
	}

	for _, model := range mergedConfig.Penpai.PenpaiModels {
		if strings.EqualFold(model.ModelName, customConfig.Penpai.PenpaiActive) {
			mergedConfig.Penpai.PenpaiActive = customConfig.Penpai.PenpaiActive
			return
		}
	}
}

func allConfigPatchFields() []configPatchField {
	connectivity := newConfigPatchSection(func(cfg *structs.SysConfig) *structs.ConnectivityConfig {
		return &cfg.Connectivity
	})
	startram := newConfigPatchSection(func(cfg *structs.SysConfig) *structs.StartramConfig {
		return &cfg.Startram
	})
	runtime := newConfigPatchSection(func(cfg *structs.SysConfig) *structs.RuntimeConfig {
		return &cfg.Runtime
	})
	penpai := newConfigPatchSection(func(cfg *structs.SysConfig) *structs.PenpaiConfig {
		return &cfg.Penpai
	})
	authSession := newConfigPatchSection(func(cfg *structs.SysConfig) *structs.AuthSessionConfig {
		return &cfg.AuthSession
	})

	fields := []configPatchField{
		connectivity.stringSliceField(
			"piers",
			"Piers",
			configPatchPointerRef(func(patch *ConfPatch) **[]string {
				return &patch.Piers
			}),
			func(cfg *structs.ConnectivityConfig) *[]string {
				return &cfg.Piers
			},
			configPatchFieldOptions[[]string]{
				hasUpdates: func(value []string) bool {
					return len(value) > 0
				},
				merge: connectivity.stringSliceIfNonEmptyField(func(cfg *structs.ConnectivityConfig) *[]string {
					return &cfg.Piers
				}),
			},
		),
		connectivity.boolField(
			"wgOn",
			"WgOn",
			configPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.WgOn
			}),
			func(cfg *structs.ConnectivityConfig) *bool {
				return &cfg.WgOn
			},
			configPatchFieldOptions[bool]{
				merge: connectivity.boolSetField(func(cfg *structs.ConnectivityConfig) *bool {
					return &cfg.WgOn
				}),
			},
		),
		connectivity.stringField(
			"endpointUrl",
			"EndpointURL",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.EndpointURL
			}),
			func(cfg *structs.ConnectivityConfig) *string {
				return &cfg.EndpointURL
			},
			configPatchFieldOptions[string]{
				merge: connectivity.stringSetIfNonEmptyField(func(cfg *structs.ConnectivityConfig) *string {
					return &cfg.EndpointURL
				}),
			},
		),
		connectivity.boolField(
			"wgRegistered",
			"WgRegistered",
			configPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.WgRegistered
			}),
			func(cfg *structs.ConnectivityConfig) *bool {
				return &cfg.WgRegistered
			},
			configPatchFieldOptions[bool]{
				merge: connectivity.boolSetField(func(cfg *structs.ConnectivityConfig) *bool {
					return &cfg.WgRegistered
				}),
			},
		),
		connectivity.stringField(
			"remoteBackupPassword",
			"RemoteBackupPassword",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.RemoteBackupPassword
			}),
			func(cfg *structs.ConnectivityConfig) *string {
				return &cfg.RemoteBackupPassword
			},
			configPatchFieldOptions[string]{
				merge: connectivity.stringSetDirectField(func(cfg *structs.ConnectivityConfig) *string {
					return &cfg.RemoteBackupPassword
				}),
			},
		),
		connectivity.stringField(
			"netCheck",
			"NetCheck",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.NetCheck
			}),
			func(cfg *structs.ConnectivityConfig) *string {
				return &cfg.NetCheck
			},
			configPatchFieldOptions[string]{
				merge: connectivity.stringSetIfNonEmptyField(func(cfg *structs.ConnectivityConfig) *string {
					return &cfg.NetCheck
				}),
			},
		),
		connectivity.stringField(
			"updateMode",
			"UpdateMode",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.UpdateMode
			}),
			func(cfg *structs.ConnectivityConfig) *string {
				return &cfg.UpdateMode
			},
			configPatchFieldOptions[string]{
				merge: connectivity.stringSetIfNonEmptyField(func(cfg *structs.ConnectivityConfig) *string {
					return &cfg.UpdateMode
				}),
			},
		),
		connectivity.stringField(
			"updateUrl",
			"UpdateURL",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.UpdateURL
			}),
			func(cfg *structs.ConnectivityConfig) *string {
				return &cfg.UpdateURL
			},
			configPatchFieldOptions[string]{
				merge: connectivity.stringSetIfNonEmptyField(func(cfg *structs.ConnectivityConfig) *string {
					return &cfg.UpdateURL
				}),
			},
		),
		connectivity.stringField(
			"updateBranch",
			"UpdateBranch",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.UpdateBranch
			}),
			func(cfg *structs.ConnectivityConfig) *string {
				return &cfg.UpdateBranch
			},
			configPatchFieldOptions[string]{
				merge: connectivity.stringSetIfNonEmptyField(func(cfg *structs.ConnectivityConfig) *string {
					return &cfg.UpdateBranch
				}),
			},
		),
		connectivity.stringField(
			"apiVersion",
			"ApiVersion",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.ApiVersion
			}),
			func(cfg *structs.ConnectivityConfig) *string {
				return &cfg.ApiVersion
			},
			configPatchFieldOptions[string]{
				merge: connectivity.stringSetIfNonEmptyField(func(cfg *structs.ConnectivityConfig) *string {
					return &cfg.ApiVersion
				}),
			},
		),
		connectivity.intField(
			"c2cInterval",
			"C2CInterval",
			configPatchPointerRef(func(patch *ConfPatch) **int {
				return &patch.C2CInterval
			}),
			func(cfg *structs.ConnectivityConfig) *int {
				return &cfg.C2CInterval
			},
			configPatchFieldOptions[int]{
				merge: connectivity.intSetIfNonZeroField(func(cfg *structs.ConnectivityConfig) *int {
					return &cfg.C2CInterval
				}),
			},
		),
		configPatchFieldFromBinding(configPatchFieldBinding[map[string]structs.DiskWarning]{
			key:        "diskWarning",
			patchField: "DiskWarning",
			target:     configPatchPointerRef(func(patch *ConfPatch) **map[string]structs.DiskWarning { return &patch.DiskWarning }),
			parse: func(value interface{}) (map[string]structs.DiskWarning, error) {
				return parseDiskWarningMap("diskWarning", value)
			},
			hasUpdates: func(warnings map[string]structs.DiskWarning) bool {
				return len(warnings) > 0
			},
			apply: func(target configPatchApplyTarget, value map[string]structs.DiskWarning) error {
				if target == nil {
					return nil
				}
				target.Connectivity.DiskWarning = copyDiskWarnings(value)
				return nil
			},
			merge: mergeDiskWarning,
		}),

		startram.boolField(
			"startramSetReminderOne",
			"StartramReminderOne",
			configPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.StartramReminderOne
			}),
			func(cfg *structs.StartramConfig) *bool {
				return &cfg.StartramSetReminder.One
			},
			configPatchFieldOptions[bool]{
				merge: startram.boolSetField(func(cfg *structs.StartramConfig) *bool {
					return &cfg.StartramSetReminder.One
				}),
			},
		),
		startram.boolField(
			"startramSetReminderThree",
			"StartramReminderThree",
			configPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.StartramReminderThree
			}),
			func(cfg *structs.StartramConfig) *bool {
				return &cfg.StartramSetReminder.Three
			},
			configPatchFieldOptions[bool]{
				merge: startram.boolSetField(func(cfg *structs.StartramConfig) *bool {
					return &cfg.StartramSetReminder.Three
				}),
			},
		),
		startram.boolField(
			"startramSetReminderSeven",
			"StartramReminderSeven",
			configPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.StartramReminderSeven
			}),
			func(cfg *structs.StartramConfig) *bool {
				return &cfg.StartramSetReminder.Seven
			},
			configPatchFieldOptions[bool]{
				merge: startram.boolSetField(func(cfg *structs.StartramConfig) *bool {
					return &cfg.StartramSetReminder.Seven
				}),
			},
		),
		startram.stringField(
			"pubkey",
			"Pubkey",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.Pubkey
			}),
			func(cfg *structs.StartramConfig) *string {
				return &cfg.Pubkey
			},
			configPatchFieldOptions[string]{
				merge: startram.stringSetIfNonEmptyField(func(cfg *structs.StartramConfig) *string {
					return &cfg.Pubkey
				}),
			},
		),
		startram.stringField(
			"privkey",
			"Privkey",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.Privkey
			}),
			func(cfg *structs.StartramConfig) *string {
				return &cfg.Privkey
			},
			configPatchFieldOptions[string]{
				merge: startram.stringSetIfNonEmptyField(func(cfg *structs.StartramConfig) *string {
					return &cfg.Privkey
				}),
			},
		),
		startram.boolField(
			"disableSlsa",
			"DisableSlsa",
			configPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.DisableSlsa
			}),
			func(cfg *structs.StartramConfig) *bool {
				return &cfg.DisableSlsa
			},
			configPatchFieldOptions[bool]{
				merge: startram.boolSetField(func(cfg *structs.StartramConfig) *bool {
					return &cfg.DisableSlsa
				}),
			},
		),

		runtime.boolField(
			"gracefulExit",
			"GracefulExit",
			configPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.GracefulExit
			}),
			func(cfg *structs.RuntimeConfig) *bool {
				return &cfg.GracefulExit
			},
			configPatchFieldOptions[bool]{
				merge: runtime.boolSetField(func(cfg *structs.RuntimeConfig) *bool {
					return &cfg.GracefulExit
				}),
			},
		),
		runtime.intField(
			"swapVal",
			"SwapVal",
			configPatchPointerRef(func(patch *ConfPatch) **int {
				return &patch.SwapVal
			}),
			func(cfg *structs.RuntimeConfig) *int {
				return &cfg.SwapVal
			},
			configPatchFieldOptions[int]{
				merge: runtime.intSetIfNonZeroField(func(cfg *structs.RuntimeConfig) *int {
					return &cfg.SwapVal
				}),
			},
		),
		runtime.stringField(
			"swapFile",
			"SwapFile",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.SwapFile
			}),
			func(cfg *structs.RuntimeConfig) *string {
				return &cfg.SwapFile
			},
			configPatchFieldOptions[string]{
				merge: runtime.stringSetIfNonEmptyField(func(cfg *structs.RuntimeConfig) *string {
					return &cfg.SwapFile
				}),
			},
		),
		configPatchFieldFromBinding(configPatchFieldBinding[linuxUpdatesPatch]{
			key:        "linuxUpdates",
			patchField: "LinuxUpdates",
			target: configPatchPointerRef(func(patch *ConfPatch) **linuxUpdatesPatch {
				return &patch.LinuxUpdates
			}),
			parse: func(value interface{}) (linuxUpdatesPatch, error) {
				return parseLinuxUpdatesValue("linuxUpdates", value)
			},
			apply: func(target configPatchApplyTarget, value linuxUpdatesPatch) error {
				if target == nil {
					return nil
				}
				target.Runtime.LinuxUpdates = struct {
					Value    int    `json:"value"`
					Interval string `json:"interval"`
				}{
					Value:    value.Value,
					Interval: value.Interval,
				}
				return nil
			},
			merge: mergeLinuxUpdates,
		}),
		runtime.stringField(
			"dockerData",
			"DockerData",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.DockerData
			}),
			func(cfg *structs.RuntimeConfig) *string {
				return &cfg.DockerData
			},
			configPatchFieldOptions[string]{
				merge: runtime.stringSetIfNonEmptyField(func(cfg *structs.RuntimeConfig) *string {
					return &cfg.DockerData
				}),
			},
		),
		runtime.stringField(
			"setup",
			"Setup",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.Setup
			}),
			func(cfg *structs.RuntimeConfig) *string {
				return &cfg.Setup
			},
			configPatchFieldOptions[string]{
				merge: mergeSetupFromConfig,
			},
		),
		runtime.stringField(
			"cfgDir",
			"CfgDir",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.CfgDir
			}),
			func(cfg *structs.RuntimeConfig) *string {
				return &cfg.CfgDir
			},
			configPatchFieldOptions[string]{
				merge: runtime.stringSetIfNonEmptyField(func(cfg *structs.RuntimeConfig) *string {
					return &cfg.CfgDir
				}),
			},
		),
		runtime.intField(
			"updateInterval",
			"UpdateInterval",
			configPatchPointerRef(func(patch *ConfPatch) **int {
				return &patch.UpdateInterval
			}),
			func(cfg *structs.RuntimeConfig) *int {
				return &cfg.UpdateInterval
			},
			configPatchFieldOptions[int]{
				merge: runtime.intSetIfNonZeroField(func(cfg *structs.RuntimeConfig) *int {
					return &cfg.UpdateInterval
				}),
			},
		),
		runtime.boolField(
			"disable502",
			"Disable502",
			configPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.Disable502
			}),
			func(cfg *structs.RuntimeConfig) *bool {
				return &cfg.Disable502
			},
			configPatchFieldOptions[bool]{
				merge: runtime.boolSetField(func(cfg *structs.RuntimeConfig) *bool {
					return &cfg.Disable502
				}),
			},
		),
		runtime.intField(
			"snapTime",
			"SnapTime",
			configPatchPointerRef(func(patch *ConfPatch) **int {
				return &patch.SnapTime
			}),
			func(cfg *structs.RuntimeConfig) *int {
				return &cfg.SnapTime
			},
			configPatchFieldOptions[int]{
				merge: runtime.intSetIfNonZeroField(func(cfg *structs.RuntimeConfig) *int {
					return &cfg.SnapTime
				}),
			},
		),
		runtime.stringField(
			"lastKnownMDNS",
			"LastKnownMDNS",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.LastKnownMDNS
			}),
			func(cfg *structs.RuntimeConfig) *string {
				return &cfg.LastKnownMDNS
			},
			configPatchFieldOptions[string]{
				merge: runtime.stringSetIfNonEmptyField(func(cfg *structs.RuntimeConfig) *string {
					return &cfg.LastKnownMDNS
				}),
			},
		),
		runtime.stringField(
			"gsVersion",
			"GSVersion",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.GSVersion
			}),
			func(cfg *structs.RuntimeConfig) *string {
				return &cfg.GsVersion
			},
			configPatchFieldOptions[string]{
				merge: runtime.stringSetIfNonEmptyField(func(cfg *structs.RuntimeConfig) *string {
					return &cfg.GsVersion
				}),
			},
		),
		runtime.stringField(
			"binHash",
			"BinHash",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.BinHash
			}),
			func(cfg *structs.RuntimeConfig) *string {
				return &cfg.BinHash
			},
			configPatchFieldOptions[string]{
				merge: runtime.stringSetIfNonEmptyField(func(cfg *structs.RuntimeConfig) *string {
					return &cfg.BinHash
				}),
			},
		),

		penpai.boolField(
			"penpaiAllow",
			"PenpaiAllow",
			configPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.PenpaiAllow
			}),
			func(cfg *structs.PenpaiConfig) *bool {
				return &cfg.PenpaiAllow
			},
			configPatchFieldOptions[bool]{
				merge: penpai.boolSetField(func(cfg *structs.PenpaiConfig) *bool {
					return &cfg.PenpaiAllow
				}),
			},
		),
		penpai.boolField(
			"penpaiRunning",
			"PenpaiRunning",
			configPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.PenpaiRunning
			}),
			func(cfg *structs.PenpaiConfig) *bool {
				return &cfg.PenpaiRunning
			},
			configPatchFieldOptions[bool]{
				merge: penpai.boolSetDirectField(func(cfg *structs.PenpaiConfig) *bool {
					return &cfg.PenpaiRunning
				}),
			},
		),
		penpai.intField(
			"penpaiCores",
			"PenpaiCores",
			configPatchPointerRef(func(patch *ConfPatch) **int {
				return &patch.PenpaiCores
			}),
			func(cfg *structs.PenpaiConfig) *int {
				return &cfg.PenpaiCores
			},
			configPatchFieldOptions[int]{
				merge: penpai.intSetIfNonZeroField(func(cfg *structs.PenpaiConfig) *int {
					return &cfg.PenpaiCores
				}),
			},
		),
		penpai.stringField(
			"penpaiActive",
			"PenpaiActive",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.PenpaiActive
			}),
			func(cfg *structs.PenpaiConfig) *string {
				return &cfg.PenpaiActive
			},
			configPatchFieldOptions[string]{
				merge: mergePenpaiActive,
			},
		),
		configPatchFieldFromBinding(configPatchFieldBinding[[]structs.Penpai]{
			key:        "penpaiModels",
			patchField: "PenpaiModels",
			target:     configPatchValueAccessor(func(patch *ConfPatch) *[]structs.Penpai { return &patch.PenpaiModels }),
			parse: func(value interface{}) ([]structs.Penpai, error) {
				return parsePenpaiModels("penpaiModels", value)
			},
			apply: func(target configPatchApplyTarget, value []structs.Penpai) error {
				if target == nil {
					return nil
				}
				target.Penpai.PenpaiModels = append([]structs.Penpai(nil), value...)
				return nil
			},
			hasUpdates: func(models []structs.Penpai) bool {
				return len(models) > 0
			},
			merge: mergeDefaultModels,
		}),

		authSession.stringField(
			"salt",
			"Salt",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.Salt
			}),
			func(cfg *structs.AuthSessionConfig) *string {
				return &cfg.Salt
			},
			configPatchFieldOptions[string]{
				merge: mergeAuthSessionSalt,
			},
		),
		authSession.stringField(
			"keyFile",
			"KeyFile",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.KeyFile
			}),
			func(cfg *structs.AuthSessionConfig) *string {
				return &cfg.KeyFile
			},
			configPatchFieldOptions[string]{
				merge: authSession.stringSetIfNonEmptyField(func(cfg *structs.AuthSessionConfig) *string {
					return &cfg.KeyFile
				}),
			},
		),
		authSession.stringField(
			"pwHash",
			"PwHash",
			configPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.PwHash
			}),
			func(cfg *structs.AuthSessionConfig) *string {
				return &cfg.PwHash
			},
			configPatchFieldOptions[string]{
				merge: authSession.stringSetIfNonEmptyField(func(cfg *structs.AuthSessionConfig) *string {
					return &cfg.PwHash
				}),
			},
		),
		configPatchFieldFromBinding(configPatchFieldBinding[map[string]structs.SessionInfo]{
			key:        "authorizedSessions",
			patchField: "AuthorizedSessions",
			target:     configPatchValueAccessor(func(patch *ConfPatch) *map[string]structs.SessionInfo { return &patch.AuthorizedSessions }),
			parse: func(value interface{}) (map[string]structs.SessionInfo, error) {
				return parseSessionMap("authorizedSessions", value)
			},
			apply: func(target configPatchApplyTarget, value map[string]structs.SessionInfo) error {
				if target == nil {
					return nil
				}
				target.AuthSession.Sessions.Authorized = mergeSessions(target.AuthSession.Sessions.Authorized, value)
				return nil
			},
			hasUpdates: func(sessions map[string]structs.SessionInfo) bool {
				return len(sessions) > 0
			},
			merge: mergeAuthSessionAuthorizedSessions,
		}),
		configPatchFieldFromBinding(configPatchFieldBinding[map[string]structs.SessionInfo]{
			key:        "unauthorizedSessions",
			patchField: "UnauthorizedSessions",
			target:     configPatchValueAccessor(func(patch *ConfPatch) *map[string]structs.SessionInfo { return &patch.UnauthorizedSessions }),
			parse: func(value interface{}) (map[string]structs.SessionInfo, error) {
				return parseSessionMap("unauthorizedSessions", value)
			},
			apply: func(target configPatchApplyTarget, value map[string]structs.SessionInfo) error {
				if target == nil {
					return nil
				}
				target.AuthSession.Sessions.Unauthorized = mergeSessions(target.AuthSession.Sessions.Unauthorized, value)
				return nil
			},
			hasUpdates: func(sessions map[string]structs.SessionInfo) bool {
				return len(sessions) > 0
			},
			merge: mergeAuthSessionUnauthorizedSessions,
		}),
	}

	fields = append(fields, unsupportedPatchField(configPatchFieldBinding[string]{})...)
	return fields
}

func unsupportedPatchField(_ configPatchFieldBinding[string]) []configPatchField {
	return []configPatchField{unsupportedConfigPatchField("isEMMCMachine")}
}
