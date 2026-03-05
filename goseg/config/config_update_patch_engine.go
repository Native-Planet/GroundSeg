package config

import (
	"encoding/json"
	"fmt"
	"sync"

	"groundseg/internal/configparse"
	"groundseg/structs"
)

type configPatchParser func(*ConfPatch, interface{}) error
type configPatchHasUpdateFn func(*ConfPatch) bool
type configPatchApplyTarget = *structs.SysConfig

type configPatchApplyFn func(configPatchApplyTarget, *ConfPatch) error
type configPatchApplyValueFn[T any] func(configPatchApplyTarget, T) error
type configPatchMergeFn func(defaultConfig, customConfig structs.SysConfig, mergedConfig *structs.SysConfig)

func applyPatchValueToSection[T any, S any](
	section func(*structs.SysConfig) *S,
	setValue func(*S, T),
) configPatchApplyValueFn[T] {
	return func(target configPatchApplyTarget, value T) error {
		if target == nil {
			return nil
		}
		setValue(section(target), value)
		return nil
	}
}

type configPatchField struct {
	key          string
	patchField   string
	parse        configPatchParser
	initErr      error
	hasUpdates   configPatchHasUpdateFn
	applyUpdates configPatchApplyFn
	mergeConfig  configPatchMergeFn
}

func (patch *ConfPatch) hasUpdates() bool {
	for _, field := range allConfigPatchFields() {
		if field.hasUpdates(patch) {
			return true
		}
	}
	return false
}

func (field configPatchField) has(patch *ConfPatch) bool {
	if field.initErr != nil {
		return false
	}
	if field.hasUpdates == nil {
		return false
	}
	return field.hasUpdates(patch)
}

func (field configPatchField) parsePatch(patch *ConfPatch, value interface{}) error {
	if field.initErr != nil {
		return field.initErr
	}
	if field.parse == nil {
		return nil
	}
	return field.parse(patch, value)
}

func (field configPatchField) apply(target configPatchApplyTarget, patch *ConfPatch) error {
	if field.initErr != nil {
		return field.initErr
	}
	if field.applyUpdates == nil || !field.has(patch) {
		return nil
	}
	return field.applyUpdates(target, patch)
}

func (field configPatchField) merge(defaultConfig, customConfig structs.SysConfig, mergedConfig *structs.SysConfig) {
	if field.initErr != nil {
		return
	}
	if field.mergeConfig == nil {
		return
	}
	field.mergeConfig(defaultConfig, customConfig, mergedConfig)
}

type configPatchValueParser[T any] func(interface{}) (T, error)

type configPatchValueRef[T any] struct {
	getTarget func(*ConfPatch) **T
	getValue  func(*ConfPatch) (T, bool)
	setValue  func(*ConfPatch, T)
}

func (target configPatchValueRef[T]) get(patch *ConfPatch) (T, bool) {
	var zero T
	if target.getValue != nil {
		return target.getValue(patch)
	}
	if target.getTarget == nil {
		return zero, false
	}
	holder := target.getTarget(patch)
	if holder == nil || *holder == nil {
		return zero, false
	}
	return **holder, true
}

func (target configPatchValueRef[T]) set(patch *ConfPatch, value T) {
	if target.setValue != nil {
		target.setValue(patch, value)
		return
	}
	if target.getTarget != nil {
		holder := target.getTarget(patch)
		if holder != nil {
			*holder = &value
		}
	}
}

func configPatchPointerRef[T any](getTarget func(*ConfPatch) **T) configPatchValueRef[T] {
	var zero T
	return configPatchValueRef[T]{
		getValue: func(patch *ConfPatch) (T, bool) {
			if getTarget == nil {
				return zero, false
			}
			holder := getTarget(patch)
			if holder == nil || *holder == nil {
				return zero, false
			}
			return **holder, true
		},
		setValue: func(patch *ConfPatch, value T) {
			holder := getTarget(patch)
			if holder != nil {
				*holder = &value
			}
		},
	}
}

func configPatchValueAccessor[T any](getTarget func(*ConfPatch) *T) configPatchValueRef[T] {
	var zero T
	return configPatchValueRef[T]{
		getValue: func(patch *ConfPatch) (T, bool) {
			if getTarget == nil {
				return zero, false
			}
			return *getTarget(patch), true
		},
		setValue: func(patch *ConfPatch, value T) {
			*getTarget(patch) = value
		},
	}
}

func newConfigPatchField[T any](
	key, patchField string,
	parseValue configPatchValueParser[T],
	target configPatchValueRef[T],
	applyValue configPatchApplyValueFn[T],
	hasUpdates func(T) bool,
	mergeConfig configPatchMergeFn,
) configPatchField {
	if hasUpdates == nil {
		hasUpdates = func(_ T) bool { return true }
	}
	return configPatchField{
		key:        key,
		patchField: patchField,
		parse: func(patch *ConfPatch, value interface{}) error {
			parsed, err := parseValue(value)
			if err != nil {
				return err
			}
			target.set(patch, parsed)
			return nil
		},
		hasUpdates: func(patch *ConfPatch) bool {
			parsed, ok := target.get(patch)
			return ok && hasUpdates(parsed)
		},
		applyUpdates: func(patchTarget configPatchApplyTarget, patch *ConfPatch) error {
			parsed, ok := target.get(patch)
			if !ok || !hasUpdates(parsed) {
				return nil
			}
			return applyValue(patchTarget, parsed)
		},
		mergeConfig: mergeConfig,
	}
}

func unsupportedConfigPatchField(key string) configPatchField {
	return configPatchField{
		key: key,
		parse: func(_ *ConfPatch, _ interface{}) error {
			return fmt.Errorf("unsupported config key: %s", key)
		},
		hasUpdates: func(_ *ConfPatch) bool {
			return false
		},
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

var (
	configPatchByKey    map[string]configPatchField
	configPatchByKeyErr error
	configPatchByKeyMu  sync.Once
)

func configPatchRegistry() (map[string]configPatchField, error) {
	configPatchByKeyMu.Do(func() {
		allFields := allConfigPatchFields()
		configPatchByKey, configPatchByKeyErr = buildConfigPatchByKey(allFields)
		if configPatchByKeyErr == nil {
			configPatchByKeyErr = validateConfigPatchFields(allFields)
		}
	})
	return configPatchByKey, configPatchByKeyErr
}

func validateConfigPatchFields(fields []configPatchField) error {
	for _, field := range fields {
		if field.initErr != nil {
			return field.initErr
		}
	}
	return nil
}

func buildConfigPatchByKey(fields []configPatchField) (map[string]configPatchField, error) {
	registry := make(map[string]configPatchField, len(fields))
	for _, field := range fields {
		if _, exists := registry[field.key]; exists {
			return nil, fmt.Errorf("duplicate config patch key: %s", field.key)
		}
		registry[field.key] = field
	}
	return registry, nil
}

func parseBoolValue(name string, value interface{}) (bool, error) {
	return configparse.Bool(name, value)
}

func parseIntValue(name string, value interface{}) (int, error) {
	return configparse.Int(name, value)
}

func parseStringValue(name string, value interface{}) (string, error) {
	return configparse.String(name, value)
}

func parseStringSliceValue(name string, value interface{}) ([]string, error) {
	if typed, ok := value.([]string); ok {
		return append([]string(nil), typed...), nil
	}
	typedInterface, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid %s value: %T", name, value)
	}
	parsed := make([]string, 0, len(typedInterface))
	for _, item := range typedInterface {
		asString, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("invalid %s item %d value: %T", name, len(parsed), item)
		}
		parsed = append(parsed, asString)
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

func parseDiskWarningMap(name string, value interface{}) (map[string]structs.DiskWarning, error) {
	rawWarnings, ok := value.(map[string]structs.DiskWarning)
	if !ok {
		return nil, fmt.Errorf("invalid %s value: %T", name, value)
	}
	copied := make(map[string]structs.DiskWarning, len(rawWarnings))
	for key, warning := range rawWarnings {
		copied[key] = warning
	}
	return copied, nil
}

func parseLinuxUpdatesValue(name string, value interface{}) (linuxUpdatesPatch, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return linuxUpdatesPatch{}, fmt.Errorf("invalid %s value: %w", name, err)
	}

	var parsed linuxUpdatesPatch
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return linuxUpdatesPatch{}, fmt.Errorf("invalid %s value: %w", name, err)
	}
	return parsed, nil
}

func parsePenpaiModels(name string, value interface{}) ([]structs.Penpai, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s value: %w", name, err)
	}

	var parsed []structs.Penpai
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("invalid %s value: %w", name, err)
	}
	return parsed, nil
}

func buildConfigPatch(values map[string]interface{}) (*ConfPatch, error) {
	registry, registryErr := configPatchRegistry()
	if registryErr != nil {
		return nil, fmt.Errorf("invalid config patch registry: %w", registryErr)
	}
	if registry == nil {
		return nil, fmt.Errorf("invalid config patch registry: registry is nil")
	}
	patch := &ConfPatch{}
	for key, value := range values {
		field, exists := registry[key]
		if !exists {
			return nil, fmt.Errorf("unsupported config key: %s", key)
		}
		if err := field.parsePatch(patch, value); err != nil {
			return nil, err
		}
	}
	return patch, nil
}
