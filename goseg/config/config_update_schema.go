package config

import (
	"encoding/json"
	"fmt"

	"groundseg/structs"
)

type ConfUpdateOption func(*ConfPatch)

type ConnectivityPatch struct {
	Piers                *[]string
	WgOn                 *bool
	NetCheck             *string
	UpdateMode           *string
	UpdateUrl            *string
	UpdateBranch         *string
	RemoteBackupPassword *string
	C2cInterval          *int
	EndpointURL          *string
	ApiVersion           *string
	DiskWarning          *map[string]structs.DiskWarning
	WgRegistered         *bool
}

type RuntimePatch struct {
	GracefulExit  *bool
	SwapVal       *int
	SwapFile      *string
	Setup         *string
	LastKnownMDNS *string
	LinuxUpdates  *struct {
		Value    int    `json:"value"`
		Interval string `json:"interval"`
	}
	DockerData     *string
	GSVersion      *string
	CfgDir         *string
	UpdateInterval *int
	BinHash        *string
	Disable502     *bool
	SnapTime       *int
}

type StartramPatch struct {
	StartramReminderOne   *bool
	StartramReminderThree *bool
	StartramReminderSeven *bool
	Pubkey                *string
	Privkey               *string
	DisableSlsa           *bool
}

type PenpaiPatch struct {
	PenpaiAllow   *bool
	PenpaiRunning *bool
	PenpaiCores   *int
	PenpaiActive  *string
	PenpaiModels  []structs.Penpai
}

type AuthSessionPatch struct {
	PwHash               *string
	Salt                 *string
	KeyFile              *string
	AuthorizedSessions   map[string]structs.SessionInfo
	UnauthorizedSessions map[string]structs.SessionInfo
}

type ConfPatch struct {
	ConnectivityPatch
	RuntimePatch
	StartramPatch
	PenpaiPatch
	AuthSessionPatch
}

type confPatchParser func(*ConfPatch, interface{}) error
type confPatchHasUpdateFn func(*ConfPatch) bool
type confPatchApplyTarget interface {
	UpdateConnectivityConfig(func(*structs.ConnectivityConfig))
	UpdateRuntimeConfig(func(*structs.RuntimeConfig))
	UpdateStartramConfig(func(*structs.StartramConfig))
	UpdatePenpaiConfig(func(*structs.PenpaiConfig))
	UpdateAuthSessionConfig(func(*structs.AuthSessionConfig))
}

type confPatchApplyFn func(confPatchApplyTarget, *ConfPatch)
type confPatchApplyValueFn[T any] func(confPatchApplyTarget, T)

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

func (field confPatchField) apply(target confPatchApplyTarget, patch *ConfPatch) {
	if field.applyUpdates == nil || !field.has(patch) {
		return
	}
	field.applyUpdates(target, patch)
}

type confPatchValueParser[T any] func(interface{}) (T, error)
type confPatchValueRef[T any] struct {
	getTarget func(*ConfPatch) **T
	getValue  func(*ConfPatch) (T, bool)
	setValue  func(*ConfPatch, T)
}

func (target confPatchValueRef[T]) get(patch *ConfPatch) (T, bool) {
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

func (target confPatchValueRef[T]) set(patch *ConfPatch, value T) {
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

func confPatchPointerRef[T any](getTarget func(*ConfPatch) **T) confPatchValueRef[T] {
	var zero T
	return confPatchValueRef[T]{
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

func confPatchValueAccessor[T any](getTarget func(*ConfPatch) *T) confPatchValueRef[T] {
	var zero T
	return confPatchValueRef[T]{
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

func newConfPatchField[T any](
	key, patchField string,
	parseValue confPatchValueParser[T],
	target confPatchValueRef[T],
	applyValue confPatchApplyValueFn[T],
	hasUpdates func(T) bool,
) confPatchField {
	if hasUpdates == nil {
		hasUpdates = func(_ T) bool { return true }
	}
	return confPatchField{
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
		applyUpdates: func(patchTarget confPatchApplyTarget, patch *ConfPatch) {
			parsed, ok := target.get(patch)
			if !ok || !hasUpdates(parsed) {
				return
			}
			applyValue(patchTarget, parsed)
		},
	}
}

func unsupportedConfPatchField(key string) confPatchField {
	return confPatchField{
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
	confPatchByKey    = map[string]confPatchField{}
	confPatchByKeyErr error
)

func init() {
	confPatchByKey, confPatchByKeyErr = buildConfPatchByKey(confPatchRegistry)
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

func parseBoolValue(name string, value interface{}) (bool, error) {
	parsed, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("invalid %s value: %T", name, value)
	}
	return parsed, nil
}

func parseIntValue(name string, value interface{}) (int, error) {
	switch parsed := value.(type) {
	case int:
		return parsed, nil
	case float64:
		return int(parsed), nil
	default:
		return 0, fmt.Errorf("invalid %s value: %T", name, value)
	}
}

func parseStringValue(name string, value interface{}) (string, error) {
	parsed, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("invalid %s value: %T", name, value)
	}
	return parsed, nil
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
		parsed = append(parsed, fmt.Sprint(item))
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

func parseLinuxUpdatesValue(name string, value interface{}) (struct {
	Value    int    `json:"value"`
	Interval string `json:"interval"`
}, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return struct {
			Value    int    `json:"value"`
			Interval string `json:"interval"`
		}{}, fmt.Errorf("invalid %s value: %w", name, err)
	}

	var parsed struct {
		Value    int    `json:"value"`
		Interval string `json:"interval"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return struct {
			Value    int    `json:"value"`
			Interval string `json:"interval"`
		}{}, fmt.Errorf("invalid %s value: %w", name, err)
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
