package config

import (
	"fmt"
	"groundseg/structs"
	"strings"
)

type confPatchFieldSpec[T any] struct {
	key        string
	patchField string
	parse      confPatchValueParser[T]
	target     confPatchValueRef[T]
	apply      confPatchApplyValueFn[T]
	hasUpdates func(T) bool
	merge      confPatchMergeFn
}

func configPatchFieldFromSpec[T any](spec confPatchFieldSpec[T]) confPatchField {
	apply := spec.apply
	if apply == nil {
		return confPatchField{
			key:        spec.key,
			patchField: spec.patchField,
			initErr:    fmt.Errorf("patch field spec requires explicit apply handler for %q", spec.key),
			hasUpdates: func(_ *ConfPatch) bool {
				return false
			},
		}
	}
	if spec.target.setValue == nil && spec.target.getValue == nil {
		return confPatchField{
			key:        spec.key,
			patchField: spec.patchField,
			initErr:    fmt.Errorf("patch field spec requires explicit target accessor for %q", spec.key),
			hasUpdates: func(_ *ConfPatch) bool {
				return false
			},
		}
	}
	return newConfPatchField(
		spec.key,
		spec.patchField,
		spec.parse,
		spec.target,
		apply,
		spec.hasUpdates,
		spec.merge,
	)
}

type confPatchConfigFieldRef[T any] func(*structs.SysConfig) *T

type confPatchFieldBinding[T any] struct {
	key         string
	patchField  string
	parse       confPatchValueParser[T]
	target      confPatchValueRef[T]
	configField confPatchConfigFieldRef[T]
	apply       confPatchApplyValueFn[T]
	hasUpdates  func(T) bool
	merge       confPatchMergeFn
}

func sectionField[Section any, T any](section func(*structs.SysConfig) *Section, field func(*Section) *T) confPatchConfigFieldRef[T] {
	return func(cfg *structs.SysConfig) *T {
		return field(section(cfg))
	}
}

func applyPatchField[T any](field confPatchConfigFieldRef[T]) confPatchApplyValueFn[T] {
	return func(target confPatchApplyTarget, value T) error {
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

func parseStringField(key string) confPatchValueParser[string] {
	return func(value interface{}) (string, error) {
		return parseStringValue(key, value)
	}
}

func parseBoolField(key string) confPatchValueParser[bool] {
	return func(value interface{}) (bool, error) {
		return parseBoolValue(key, value)
	}
}

func parseIntField(key string) confPatchValueParser[int] {
	return func(value interface{}) (int, error) {
		return parseIntValue(key, value)
	}
}

func configPatchFieldFromBinding[T any](spec confPatchFieldBinding[T]) confPatchField {
	apply := spec.apply
	if apply == nil {
		apply = applyPatchField(spec.configField)
	}
	return configPatchFieldFromSpec(confPatchFieldSpec[T]{
		key:        spec.key,
		patchField: spec.patchField,
		parse:      spec.parse,
		target:     spec.target,
		apply:      apply,
		hasUpdates: spec.hasUpdates,
		merge:      spec.merge,
	})
}

type confPatchFieldOptions[T any] struct {
	apply      confPatchApplyValueFn[T]
	hasUpdates func(T) bool
	merge      confPatchMergeFn
}

type confPatchSectionBuilder[Section any] struct {
	configSection func(*structs.SysConfig) *Section
}

func newConfPatchSection[Section any](section func(*structs.SysConfig) *Section) confPatchSectionBuilder[Section] {
	return confPatchSectionBuilder[Section]{
		configSection: section,
	}
}

func patchConfigFieldFromSection[Section any, T any](
	builder confPatchSectionBuilder[Section],
	key string,
	patchField string,
	target confPatchValueRef[T],
	parse confPatchValueParser[T],
	cfgField func(*Section) *T,
	options confPatchFieldOptions[T],
) confPatchField {
	configField := sectionField(builder.configSection, cfgField)
	return configPatchFieldFromBinding(confPatchFieldBinding[T]{
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

func (builder confPatchSectionBuilder[Section]) boolField(
	key string,
	patchField string,
	target confPatchValueRef[bool],
	cfgField func(*Section) *bool,
	options confPatchFieldOptions[bool],
) confPatchField {
	return patchConfigFieldFromSection(builder, key, patchField, target, parseBoolField(key), cfgField, options)
}

func (builder confPatchSectionBuilder[Section]) intField(
	key string,
	patchField string,
	target confPatchValueRef[int],
	cfgField func(*Section) *int,
	options confPatchFieldOptions[int],
) confPatchField {
	return patchConfigFieldFromSection(builder, key, patchField, target, parseIntField(key), cfgField, options)
}

func (builder confPatchSectionBuilder[Section]) stringField(
	key string,
	patchField string,
	target confPatchValueRef[string],
	cfgField func(*Section) *string,
	options confPatchFieldOptions[string],
) confPatchField {
	return patchConfigFieldFromSection(builder, key, patchField, target, parseStringField(key), cfgField, options)
}

func (builder confPatchSectionBuilder[Section]) boolOrConfigField(cfgField func(*Section) *bool) confPatchMergeFn {
	return mergeBoolOrConfigField(sectionField(builder.configSection, cfgField))
}

func (builder confPatchSectionBuilder[Section]) boolSetField(cfgField func(*Section) *bool) confPatchMergeFn {
	return mergeBoolSetRef(sectionField(builder.configSection, cfgField))
}

func (builder confPatchSectionBuilder[Section]) stringSetIfNonEmptyField(cfgField func(*Section) *string) confPatchMergeFn {
	return mergeStringSetIfNonEmptyRef(sectionField(builder.configSection, cfgField))
}

func (builder confPatchSectionBuilder[Section]) stringSetDirectField(cfgField func(*Section) *string) confPatchMergeFn {
	return mergeStringSetDirectRef(sectionField(builder.configSection, cfgField))
}

func (builder confPatchSectionBuilder[Section]) intSetIfNonZeroField(cfgField func(*Section) *int) confPatchMergeFn {
	return mergeIntSetIfNonZeroRef(sectionField(builder.configSection, cfgField))
}

func (builder confPatchSectionBuilder[Section]) stringSliceIfNonEmptyField(cfgField func(*Section) *[]string) confPatchMergeFn {
	return mergeStringSliceSetIfNonEmptyRef(sectionField(builder.configSection, cfgField))
}

func mergeIfString(configField confPatchConfigFieldRef[string], shouldMerge func(string) bool) confPatchMergeFn {
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

func mergeIfInt(configField confPatchConfigFieldRef[int], shouldMerge func(int) bool) confPatchMergeFn {
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

func mergeIfBool(configField confPatchConfigFieldRef[bool], shouldMerge func(bool) bool) confPatchMergeFn {
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
			*destination = *value
		}
	}
}

func mergeBoolOrConfigField(configField confPatchConfigFieldRef[bool]) confPatchMergeFn {
	return func(_ structs.SysConfig, customConfig structs.SysConfig, mergedConfig *structs.SysConfig) {
		value := configField(&customConfig)
		if value == nil || !*value {
			return
		}
		destination := configField(mergedConfig)
		if destination != nil {
			*destination = true
		}
	}
}

func mergeSliceWhen[T any](configField confPatchConfigFieldRef[[]T], shouldMerge func([]T) bool) confPatchMergeFn {
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

func mergeStringSetIfNonEmptyRef(configField confPatchConfigFieldRef[string]) confPatchMergeFn {
	return mergeIfString(configField, func(value string) bool { return value != "" })
}

func mergeStringSetDirectRef(configField confPatchConfigFieldRef[string]) confPatchMergeFn {
	return mergeIfString(configField, nil)
}

func mergeIntSetIfNonZeroRef(configField confPatchConfigFieldRef[int]) confPatchMergeFn {
	return mergeIfInt(configField, func(value int) bool { return value != 0 })
}

func mergeBoolSetRef(configField confPatchConfigFieldRef[bool]) confPatchMergeFn {
	return mergeIfBool(configField, nil)
}

func mergeStringSliceSetIfNonEmptyRef(configField confPatchConfigFieldRef[[]string]) confPatchMergeFn {
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
		mergedConfig.AuthSession.Salt = RandString(32)
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

func allConfPatchFields() []confPatchField {
	connectivity := newConfPatchSection(func(cfg *structs.SysConfig) *structs.ConnectivityConfig {
		return &cfg.Connectivity
	})
	startram := newConfPatchSection(func(cfg *structs.SysConfig) *structs.StartramConfig {
		return &cfg.Startram
	})
	runtime := newConfPatchSection(func(cfg *structs.SysConfig) *structs.RuntimeConfig {
		return &cfg.Runtime
	})
	penpai := newConfPatchSection(func(cfg *structs.SysConfig) *structs.PenpaiConfig {
		return &cfg.Penpai
	})
	authSession := newConfPatchSection(func(cfg *structs.SysConfig) *structs.AuthSessionConfig {
		return &cfg.AuthSession
	})

	fields := []confPatchField{
		patchConfigFieldFromSection(
			connectivity,
			"piers",
			"Piers",
			confPatchPointerRef(func(patch *ConfPatch) **[]string {
				return &patch.Piers
			}),
			func(value interface{}) ([]string, error) {
				return parseStringSliceValue("piers", value)
			},
			func(cfg *structs.ConnectivityConfig) *[]string {
				return &cfg.Piers
			},
			confPatchFieldOptions[[]string]{
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
			confPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.WgOn
			}),
			func(cfg *structs.ConnectivityConfig) *bool {
				return &cfg.WgOn
			},
			confPatchFieldOptions[bool]{
				merge: connectivity.boolOrConfigField(func(cfg *structs.ConnectivityConfig) *bool {
					return &cfg.WgOn
				}),
			},
		),
		connectivity.stringField(
			"endpointUrl",
			"EndpointURL",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.EndpointURL
			}),
			func(cfg *structs.ConnectivityConfig) *string {
				return &cfg.EndpointUrl
			},
			confPatchFieldOptions[string]{
				merge: connectivity.stringSetIfNonEmptyField(func(cfg *structs.ConnectivityConfig) *string {
					return &cfg.EndpointUrl
				}),
			},
		),
		connectivity.boolField(
			"wgRegistered",
			"WgRegistered",
			confPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.WgRegistered
			}),
			func(cfg *structs.ConnectivityConfig) *bool {
				return &cfg.WgRegistered
			},
			confPatchFieldOptions[bool]{
				merge: connectivity.boolOrConfigField(func(cfg *structs.ConnectivityConfig) *bool {
					return &cfg.WgRegistered
				}),
			},
		),
		connectivity.stringField(
			"remoteBackupPassword",
			"RemoteBackupPassword",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.RemoteBackupPassword
			}),
			func(cfg *structs.ConnectivityConfig) *string {
				return &cfg.RemoteBackupPassword
			},
			confPatchFieldOptions[string]{
				merge: connectivity.stringSetDirectField(func(cfg *structs.ConnectivityConfig) *string {
					return &cfg.RemoteBackupPassword
				}),
			},
		),
		connectivity.stringField(
			"netCheck",
			"NetCheck",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.NetCheck
			}),
			func(cfg *structs.ConnectivityConfig) *string {
				return &cfg.NetCheck
			},
			confPatchFieldOptions[string]{
				merge: connectivity.stringSetIfNonEmptyField(func(cfg *structs.ConnectivityConfig) *string {
					return &cfg.NetCheck
				}),
			},
		),
		connectivity.stringField(
			"updateMode",
			"UpdateMode",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.UpdateMode
			}),
			func(cfg *structs.ConnectivityConfig) *string {
				return &cfg.UpdateMode
			},
			confPatchFieldOptions[string]{
				merge: connectivity.stringSetIfNonEmptyField(func(cfg *structs.ConnectivityConfig) *string {
					return &cfg.UpdateMode
				}),
			},
		),
		connectivity.stringField(
			"updateUrl",
			"UpdateUrl",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.UpdateUrl
			}),
			func(cfg *structs.ConnectivityConfig) *string {
				return &cfg.UpdateUrl
			},
			confPatchFieldOptions[string]{
				merge: connectivity.stringSetIfNonEmptyField(func(cfg *structs.ConnectivityConfig) *string {
					return &cfg.UpdateUrl
				}),
			},
		),
		connectivity.stringField(
			"updateBranch",
			"UpdateBranch",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.UpdateBranch
			}),
			func(cfg *structs.ConnectivityConfig) *string {
				return &cfg.UpdateBranch
			},
			confPatchFieldOptions[string]{
				merge: connectivity.stringSetIfNonEmptyField(func(cfg *structs.ConnectivityConfig) *string {
					return &cfg.UpdateBranch
				}),
			},
		),
		connectivity.stringField(
			"apiVersion",
			"ApiVersion",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.ApiVersion
			}),
			func(cfg *structs.ConnectivityConfig) *string {
				return &cfg.ApiVersion
			},
			confPatchFieldOptions[string]{
				merge: connectivity.stringSetIfNonEmptyField(func(cfg *structs.ConnectivityConfig) *string {
					return &cfg.ApiVersion
				}),
			},
		),
		connectivity.intField(
			"c2cInterval",
			"C2cInterval",
			confPatchPointerRef(func(patch *ConfPatch) **int {
				return &patch.C2cInterval
			}),
			func(cfg *structs.ConnectivityConfig) *int {
				return &cfg.C2cInterval
			},
			confPatchFieldOptions[int]{
				merge: connectivity.intSetIfNonZeroField(func(cfg *structs.ConnectivityConfig) *int {
					return &cfg.C2cInterval
				}),
			},
		),
		configPatchFieldFromBinding(confPatchFieldBinding[map[string]structs.DiskWarning]{
			key:        "diskWarning",
			patchField: "DiskWarning",
			target:     confPatchPointerRef(func(patch *ConfPatch) **map[string]structs.DiskWarning { return &patch.DiskWarning }),
			parse: func(value interface{}) (map[string]structs.DiskWarning, error) {
				return parseDiskWarningMap("diskWarning", value)
			},
			hasUpdates: func(warnings map[string]structs.DiskWarning) bool {
				return len(warnings) > 0
			},
			apply: func(target confPatchApplyTarget, value map[string]structs.DiskWarning) error {
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
			confPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.StartramReminderOne
			}),
			func(cfg *structs.StartramConfig) *bool {
				return &cfg.StartramSetReminder.One
			},
			confPatchFieldOptions[bool]{
				merge: startram.boolOrConfigField(func(cfg *structs.StartramConfig) *bool {
					return &cfg.StartramSetReminder.One
				}),
			},
		),
		startram.boolField(
			"startramSetReminderThree",
			"StartramReminderThree",
			confPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.StartramReminderThree
			}),
			func(cfg *structs.StartramConfig) *bool {
				return &cfg.StartramSetReminder.Three
			},
			confPatchFieldOptions[bool]{
				merge: startram.boolOrConfigField(func(cfg *structs.StartramConfig) *bool {
					return &cfg.StartramSetReminder.Three
				}),
			},
		),
		startram.boolField(
			"startramSetReminderSeven",
			"StartramReminderSeven",
			confPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.StartramReminderSeven
			}),
			func(cfg *structs.StartramConfig) *bool {
				return &cfg.StartramSetReminder.Seven
			},
			confPatchFieldOptions[bool]{
				merge: startram.boolOrConfigField(func(cfg *structs.StartramConfig) *bool {
					return &cfg.StartramSetReminder.Seven
				}),
			},
		),
		startram.stringField(
			"pubkey",
			"Pubkey",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.Pubkey
			}),
			func(cfg *structs.StartramConfig) *string {
				return &cfg.Pubkey
			},
			confPatchFieldOptions[string]{
				merge: startram.stringSetIfNonEmptyField(func(cfg *structs.StartramConfig) *string {
					return &cfg.Pubkey
				}),
			},
		),
		startram.stringField(
			"privkey",
			"Privkey",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.Privkey
			}),
			func(cfg *structs.StartramConfig) *string {
				return &cfg.Privkey
			},
			confPatchFieldOptions[string]{
				merge: startram.stringSetIfNonEmptyField(func(cfg *structs.StartramConfig) *string {
					return &cfg.Privkey
				}),
			},
		),
		startram.boolField(
			"disableSlsa",
			"DisableSlsa",
			confPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.DisableSlsa
			}),
			func(cfg *structs.StartramConfig) *bool {
				return &cfg.DisableSlsa
			},
			confPatchFieldOptions[bool]{
				merge: startram.boolSetField(func(cfg *structs.StartramConfig) *bool {
					return &cfg.DisableSlsa
				}),
			},
		),

		runtime.boolField(
			"gracefulExit",
			"GracefulExit",
			confPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.GracefulExit
			}),
			func(cfg *structs.RuntimeConfig) *bool {
				return &cfg.GracefulExit
			},
			confPatchFieldOptions[bool]{
				merge: runtime.boolOrConfigField(func(cfg *structs.RuntimeConfig) *bool {
					return &cfg.GracefulExit
				}),
			},
		),
		runtime.intField(
			"swapVal",
			"SwapVal",
			confPatchPointerRef(func(patch *ConfPatch) **int {
				return &patch.SwapVal
			}),
			func(cfg *structs.RuntimeConfig) *int {
				return &cfg.SwapVal
			},
			confPatchFieldOptions[int]{
				merge: runtime.intSetIfNonZeroField(func(cfg *structs.RuntimeConfig) *int {
					return &cfg.SwapVal
				}),
			},
		),
		runtime.stringField(
			"swapFile",
			"SwapFile",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.SwapFile
			}),
			func(cfg *structs.RuntimeConfig) *string {
				return &cfg.SwapFile
			},
			confPatchFieldOptions[string]{
				merge: runtime.stringSetIfNonEmptyField(func(cfg *structs.RuntimeConfig) *string {
					return &cfg.SwapFile
				}),
			},
		),
		configPatchFieldFromBinding(confPatchFieldBinding[linuxUpdatesPatch]{
			key:        "linuxUpdates",
			patchField: "LinuxUpdates",
			target: confPatchPointerRef(func(patch *ConfPatch) **linuxUpdatesPatch {
				return &patch.LinuxUpdates
			}),
			parse: func(value interface{}) (linuxUpdatesPatch, error) {
				return parseLinuxUpdatesValue("linuxUpdates", value)
			},
			apply: func(target confPatchApplyTarget, value linuxUpdatesPatch) error {
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
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.DockerData
			}),
			func(cfg *structs.RuntimeConfig) *string {
				return &cfg.DockerData
			},
			confPatchFieldOptions[string]{
				merge: runtime.stringSetIfNonEmptyField(func(cfg *structs.RuntimeConfig) *string {
					return &cfg.DockerData
				}),
			},
		),
		runtime.stringField(
			"setup",
			"Setup",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.Setup
			}),
			func(cfg *structs.RuntimeConfig) *string {
				return &cfg.Setup
			},
			confPatchFieldOptions[string]{
				merge: mergeSetupFromConfig,
			},
		),
		runtime.stringField(
			"cfgDir",
			"CfgDir",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.CfgDir
			}),
			func(cfg *structs.RuntimeConfig) *string {
				return &cfg.CfgDir
			},
			confPatchFieldOptions[string]{
				merge: runtime.stringSetIfNonEmptyField(func(cfg *structs.RuntimeConfig) *string {
					return &cfg.CfgDir
				}),
			},
		),
		runtime.intField(
			"updateInterval",
			"UpdateInterval",
			confPatchPointerRef(func(patch *ConfPatch) **int {
				return &patch.UpdateInterval
			}),
			func(cfg *structs.RuntimeConfig) *int {
				return &cfg.UpdateInterval
			},
			confPatchFieldOptions[int]{
				merge: runtime.intSetIfNonZeroField(func(cfg *structs.RuntimeConfig) *int {
					return &cfg.UpdateInterval
				}),
			},
		),
		runtime.boolField(
			"disable502",
			"Disable502",
			confPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.Disable502
			}),
			func(cfg *structs.RuntimeConfig) *bool {
				return &cfg.Disable502
			},
			confPatchFieldOptions[bool]{
				merge: runtime.boolOrConfigField(func(cfg *structs.RuntimeConfig) *bool {
					return &cfg.Disable502
				}),
			},
		),
		runtime.intField(
			"snapTime",
			"SnapTime",
			confPatchPointerRef(func(patch *ConfPatch) **int {
				return &patch.SnapTime
			}),
			func(cfg *structs.RuntimeConfig) *int {
				return &cfg.SnapTime
			},
			confPatchFieldOptions[int]{
				merge: runtime.intSetIfNonZeroField(func(cfg *structs.RuntimeConfig) *int {
					return &cfg.SnapTime
				}),
			},
		),
		runtime.stringField(
			"lastKnownMDNS",
			"LastKnownMDNS",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.LastKnownMDNS
			}),
			func(cfg *structs.RuntimeConfig) *string {
				return &cfg.LastKnownMDNS
			},
			confPatchFieldOptions[string]{
				merge: runtime.stringSetIfNonEmptyField(func(cfg *structs.RuntimeConfig) *string {
					return &cfg.LastKnownMDNS
				}),
			},
		),
		runtime.stringField(
			"gsVersion",
			"GSVersion",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.GSVersion
			}),
			func(cfg *structs.RuntimeConfig) *string {
				return &cfg.GsVersion
			},
			confPatchFieldOptions[string]{
				merge: runtime.stringSetIfNonEmptyField(func(cfg *structs.RuntimeConfig) *string {
					return &cfg.GsVersion
				}),
			},
		),
		runtime.stringField(
			"binHash",
			"BinHash",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.BinHash
			}),
			func(cfg *structs.RuntimeConfig) *string {
				return &cfg.BinHash
			},
			confPatchFieldOptions[string]{
				merge: runtime.stringSetIfNonEmptyField(func(cfg *structs.RuntimeConfig) *string {
					return &cfg.BinHash
				}),
			},
		),

		penpai.boolField(
			"penpaiAllow",
			"PenpaiAllow",
			confPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.PenpaiAllow
			}),
			func(cfg *structs.PenpaiConfig) *bool {
				return &cfg.PenpaiAllow
			},
			confPatchFieldOptions[bool]{
				merge: penpai.boolOrConfigField(func(cfg *structs.PenpaiConfig) *bool {
					return &cfg.PenpaiAllow
				}),
			},
		),
		penpai.boolField(
			"penpaiRunning",
			"PenpaiRunning",
			confPatchPointerRef(func(patch *ConfPatch) **bool {
				return &patch.PenpaiRunning
			}),
			func(cfg *structs.PenpaiConfig) *bool {
				return &cfg.PenpaiRunning
			},
			confPatchFieldOptions[bool]{
				merge: penpai.boolSetField(func(cfg *structs.PenpaiConfig) *bool {
					return &cfg.PenpaiRunning
				}),
			},
		),
		penpai.intField(
			"penpaiCores",
			"PenpaiCores",
			confPatchPointerRef(func(patch *ConfPatch) **int {
				return &patch.PenpaiCores
			}),
			func(cfg *structs.PenpaiConfig) *int {
				return &cfg.PenpaiCores
			},
			confPatchFieldOptions[int]{
				merge: penpai.intSetIfNonZeroField(func(cfg *structs.PenpaiConfig) *int {
					return &cfg.PenpaiCores
				}),
			},
		),
		penpai.stringField(
			"penpaiActive",
			"PenpaiActive",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.PenpaiActive
			}),
			func(cfg *structs.PenpaiConfig) *string {
				return &cfg.PenpaiActive
			},
			confPatchFieldOptions[string]{
				merge: mergePenpaiActive,
			},
		),
		configPatchFieldFromBinding(confPatchFieldBinding[[]structs.Penpai]{
			key:        "penpaiModels",
			patchField: "PenpaiModels",
			target:     confPatchValueAccessor(func(patch *ConfPatch) *[]structs.Penpai { return &patch.PenpaiModels }),
			parse: func(value interface{}) ([]structs.Penpai, error) {
				return parsePenpaiModels("penpaiModels", value)
			},
			apply: func(target confPatchApplyTarget, value []structs.Penpai) error {
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
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.Salt
			}),
			func(cfg *structs.AuthSessionConfig) *string {
				return &cfg.Salt
			},
			confPatchFieldOptions[string]{
				merge: mergeAuthSessionSalt,
			},
		),
		authSession.stringField(
			"keyFile",
			"KeyFile",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.KeyFile
			}),
			func(cfg *structs.AuthSessionConfig) *string {
				return &cfg.KeyFile
			},
			confPatchFieldOptions[string]{
				merge: authSession.stringSetIfNonEmptyField(func(cfg *structs.AuthSessionConfig) *string {
					return &cfg.KeyFile
				}),
			},
		),
		authSession.stringField(
			"pwHash",
			"PwHash",
			confPatchPointerRef(func(patch *ConfPatch) **string {
				return &patch.PwHash
			}),
			func(cfg *structs.AuthSessionConfig) *string {
				return &cfg.PwHash
			},
			confPatchFieldOptions[string]{
				merge: authSession.stringSetIfNonEmptyField(func(cfg *structs.AuthSessionConfig) *string {
					return &cfg.PwHash
				}),
			},
		),
		configPatchFieldFromBinding(confPatchFieldBinding[map[string]structs.SessionInfo]{
			key:        "authorizedSessions",
			patchField: "AuthorizedSessions",
			target:     confPatchValueAccessor(func(patch *ConfPatch) *map[string]structs.SessionInfo { return &patch.AuthorizedSessions }),
			parse: func(value interface{}) (map[string]structs.SessionInfo, error) {
				return parseSessionMap("authorizedSessions", value)
			},
			apply: func(target confPatchApplyTarget, value map[string]structs.SessionInfo) error {
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
		configPatchFieldFromBinding(confPatchFieldBinding[map[string]structs.SessionInfo]{
			key:        "unauthorizedSessions",
			patchField: "UnauthorizedSessions",
			target:     confPatchValueAccessor(func(patch *ConfPatch) *map[string]structs.SessionInfo { return &patch.UnauthorizedSessions }),
			parse: func(value interface{}) (map[string]structs.SessionInfo, error) {
				return parseSessionMap("unauthorizedSessions", value)
			},
			apply: func(target confPatchApplyTarget, value map[string]structs.SessionInfo) error {
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

	fields = append(fields, unsupportedPatchField(confPatchFieldBinding[string]{})...)
	return fields
}

func unsupportedPatchField(_ confPatchFieldBinding[string]) []confPatchField {
	return []confPatchField{unsupportedConfPatchField("isEMMCMachine")}
}
