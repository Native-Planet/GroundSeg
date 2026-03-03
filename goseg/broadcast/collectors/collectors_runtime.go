package collectors

import (
	"fmt"
	"groundseg/backupsvc"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/network"
	"groundseg/docker/orchestration"
	"groundseg/startram"
	"groundseg/structs"
	"reflect"
	"time"
	"unsafe"
)

var defaultCollectorRuntimeValue = NewCollectorRuntime()

type collectorRuntime struct {
	collectorUrbitRuntime
	collectorConfigRuntime
}

type collectorUrbitRuntime struct {
	LoadUrbitConfigFn        func(string) error
	UrbitConfFn              func(string) structs.UrbitDocker
	GetContainerStatsFn      func(string) structs.ContainerStats
	GetContainerImageTagFn   func(string) (string, error)
	GetMinIOLinkedStatusFn   func(string) bool
	GetMinIOPasswordFn       func(string) (string, error)
	GetContainerNetworkFn    func(string) (string, error)
	GetContainerShipStatusFn func([]string) (map[string]string, error)
	LusCodeFn                func(string) (string, error)
	GetDeskFn                func(string, string, bool) (string, error)
}

type collectorConfigRuntime struct {
	StartramSettingsFn          func() config.StartramSettings
	StartramConfigFn            func() structs.StartramRetrieve
	StartramServicesRetrieverFn func() (structs.StartramRetrieve, error)
	PenpaiSettingsFn            func() config.PenpaiSettings
	SwapSettingsFn              func() config.SwapSettings
	BackupRootFn                func() string
	BackupTimeFn                func() time.Time
}

func NewCollectorRuntime() collectorRuntime {
	networkRuntime := network.NewNetworkRuntime()
	return collectorRuntime{
		collectorUrbitRuntime: collectorUrbitRuntime{
			LoadUrbitConfigFn:        config.LoadUrbitConfig,
			UrbitConfFn:              config.UrbitConf,
			GetContainerStatsFn:      orchestration.GetContainerStats,
			GetContainerImageTagFn:   orchestration.GetContainerImageTag,
			GetMinIOLinkedStatusFn:   config.GetMinIOLinkedStatus,
			GetMinIOPasswordFn:       config.GetMinIOPassword,
			GetContainerNetworkFn:    networkRuntime.GetContainerNetwork,
			GetContainerShipStatusFn: orchestration.GetShipStatus,
			LusCodeFn:                click.GetLusCode,
			GetDeskFn:                click.GetDesk,
		},
		collectorConfigRuntime: collectorConfigRuntime{
			StartramSettingsFn:          config.StartramSettingsSnapshot,
			StartramConfigFn:            config.GetStartramConfig,
			StartramServicesRetrieverFn: startram.Retrieve,
			PenpaiSettingsFn:            config.PenpaiSettingsSnapshot,
			SwapSettingsFn:              config.SwapSettingsSnapshot,
			BackupRootFn:                func() string { return backupsvc.ResolveBackupRoot(config.BasePath()) },
			BackupTimeFn:                func() time.Time { return config.BackupTime },
		},
	}
}

func defaultCollectorRuntime() collectorRuntime {
	return DefaultCollectorRuntime()
}

func DefaultCollectorRuntime() collectorRuntime {
	return defaultCollectorRuntimeValue
}

func collectorRuntimeWithDefaults(runtime collectorRuntime) collectorRuntime {
	defaultRuntime := NewCollectorRuntime()
	defaultRuntime.collectorUrbitRuntime = mergeRuntimeCallbacks(defaultRuntime.collectorUrbitRuntime, runtime.collectorUrbitRuntime)
	defaultRuntime.collectorConfigRuntime = mergeRuntimeCallbacks(defaultRuntime.collectorConfigRuntime, runtime.collectorConfigRuntime)
	return defaultRuntime
}

func mergeRuntimeCallbacks[T any](defaults, overrides T) T {
	defaultValue := reflect.ValueOf(&defaults).Elem()
	overrideValue := reflect.ValueOf(overrides)
	if !defaultValue.IsValid() || !overrideValue.IsValid() {
		return defaults
	}
	if defaultValue.Kind() != reflect.Struct || overrideValue.Kind() != reflect.Struct {
		return defaults
	}
	for i := 0; i < overrideValue.NumField(); i++ {
		overrideField := overrideValue.Field(i)
		if !isNonZeroValue(overrideField) {
			continue
		}
		defaultField := settableValue(defaultValue.Field(i))
		overridden := settableValue(overrideField)
		if defaultField.CanSet() {
			defaultField.Set(overridden)
		}
	}
	return defaults
}

func settableValue(v reflect.Value) reflect.Value {
	if v.CanSet() || !v.CanAddr() {
		return v
	}
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

func isNonZeroValue(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return !v.IsNil()
	default:
		return !v.IsZero()
	}
}

func collectorRuntimeOrDefault(runtime ...collectorRuntime) collectorRuntime {
	if len(runtime) > 0 {
		return collectorRuntimeWithDefaults(runtime[0])
	}
	return defaultCollectorRuntime()
}

// ConstructPierInfo builds the urbit entries for broadcast state.
func ConstructPierInfo(existingUrbits map[string]structs.Urbit, scheduled func(string) time.Time) (map[string]structs.Urbit, error) {
	return constructPierInfo(defaultCollectorRuntime(), existingUrbits, scheduled)
}

func constructPierInfo(runtime collectorRuntime, existingUrbits map[string]structs.Urbit, scheduled func(string) time.Time) (map[string]structs.Urbit, error) {
	runtime = collectorRuntimeWithDefaults(runtime)
	startramSettings := runtime.StartramSettingsFn()
	startramConfig := runtime.StartramConfigFn()
	settings := startramSettings
	piers := settings.Piers
	sgContext := wireguardContext{
		registered: settings.WgRegistered,
		on:         settings.WgOn,
	}

	backups := backupSnapshotForRuntime(runtime, piers, startramConfig.Backups)
	rtSnapshot, err := runtimeSnapshotForPiersWithRuntime(runtime, piers, existingUrbits)
	if err != nil {
		return nil, fmt.Errorf("constructing pier info: %w", err)
	}
	startramSnapshot := startramSnapshotForPiers(startramConfig.Subdomains)
	deploymentInputs := collectUrbitDeploymentInputsForPiers(
		piers,
		rtSnapshot.hostName,
		sgContext,
		startramSnapshot,
		runtime,
	)
	runtimeInputs := collectUrbitRuntimeInputsForPiers(
		rtSnapshot.pierStatus,
		urbitRuntimeContext{
			existingUrbits: rtSnapshot.currentState,
			shipNetworks:   rtSnapshot.shipNetworks,
		},
		scheduled,
		runtime,
	)
	return composeUrbitViews(piers, runtimeInputs, deploymentInputs, backups), nil
}

func (runtime collectorRuntime) backupRoot() (string, bool) {
	if runtime.BackupRootFn == nil {
		return "", false
	}
	return runtime.BackupRootFn(), true
}
