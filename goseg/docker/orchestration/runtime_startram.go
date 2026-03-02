package orchestration

import (
	"fmt"
	"groundseg/config"
	"groundseg/structs"
)

// StartramRuntime captures only the orchestration and state dependencies
// required by startram workflows.
type StartramRuntime struct {
	GetStartramServicesFn       func() error
	LoadStartramRegionsFn       func() error
	StartramSettingsSnapshotFn  func() config.StartramSettings
	ShipSettingsSnapshotFn      func() config.ShipSettings
	GetStartramConfigFn         func() structs.StartramRetrieve
	UpdateConfTypedFn           func(...config.ConfUpdateOption) error
	WithWgOnFn                  func(bool) config.ConfUpdateOption
	CycleWgKeyFn                func() error
	StopContainerByNameFn       func(name string) error
	UrbitConfFn                 func(string) structs.UrbitDocker
	StartContainerFn            func(name string, ctype string) (structs.ContainerState, error)
	DeleteContainerFn           func(name string) error
	LoadWireguardFn             func() error
	LoadMCFn                    func() error
	LoadMinIOsFn                func() error
	SvcDeleteFn                 func(patp string, kind string) error
	UpdateUrbitFn               func(patp string, update func(*structs.UrbitDocker) error) error
	UpdateUrbitRuntimeConfigFn  func(patp string, update func(*structs.UrbitRuntimeConfig) error) error
	UpdateUrbitNetworkConfigFn  func(patp string, update func(*structs.UrbitNetworkConfig) error) error
	UpdateUrbitScheduleConfigFn func(patp string, update func(*structs.UrbitScheduleConfig) error) error
	UpdateUrbitFeatureConfigFn  func(patp string, update func(*structs.UrbitFeatureConfig) error) error
	UpdateUrbitWebConfigFn      func(patp string, update func(*structs.UrbitWebConfig) error) error
	UpdateUrbitBackupConfigFn   func(patp string, update func(*structs.UrbitBackupConfig) error) error
}

type StartramRuntimeOption func(*StartramRuntime)

func WithStartramServiceLoaders(getStartramServicesFn, loadStartramRegionsFn func() error) StartramRuntimeOption {
	return func(rt *StartramRuntime) {
		if getStartramServicesFn != nil {
			rt.GetStartramServicesFn = getStartramServicesFn
		}
		if loadStartramRegionsFn != nil {
			rt.LoadStartramRegionsFn = loadStartramRegionsFn
		}
	}
}

func NewStartramRuntime(opts ...StartramRuntimeOption) StartramRuntime {
	rt := NewRuntime()
	sr := StartramRuntime{
		GetStartramServicesFn:       defaultGetStartramServices,
		LoadStartramRegionsFn:       defaultLoadStartramRegions,
		StartramSettingsSnapshotFn:  rt.StartramSettingsSnapshotFn,
		ShipSettingsSnapshotFn:      rt.ShipSettingsSnapshotFn,
		GetStartramConfigFn:         rt.GetStartramConfigFn,
		UpdateConfTypedFn:           rt.UpdateConfTypedFn,
		WithWgOnFn:                  rt.WithWgOnFn,
		CycleWgKeyFn:                rt.CycleWgKeyFn,
		StopContainerByNameFn:       rt.StopContainerByNameFn,
		UrbitConfFn:                 rt.UrbitConfFn,
		StartContainerFn:            rt.StartContainerFn,
		DeleteContainerFn:           rt.DeleteContainerFn,
		LoadWireguardFn:             rt.LoadWireguardFn,
		LoadMCFn:                    rt.LoadMCFn,
		LoadMinIOsFn:                rt.LoadMinIOsFn,
		SvcDeleteFn:                 rt.SvcDeleteFn,
		UpdateUrbitFn:               rt.UpdateUrbitFn,
		UpdateUrbitRuntimeConfigFn:  rt.UpdateUrbitRuntimeConfigFn,
		UpdateUrbitNetworkConfigFn:  rt.UpdateUrbitNetworkConfigFn,
		UpdateUrbitScheduleConfigFn: rt.UpdateUrbitScheduleConfigFn,
		UpdateUrbitFeatureConfigFn:  rt.UpdateUrbitFeatureConfigFn,
		UpdateUrbitWebConfigFn:      rt.UpdateUrbitWebConfigFn,
		UpdateUrbitBackupConfigFn:   rt.UpdateUrbitBackupConfigFn,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&sr)
		}
	}
	return sr
}

var (
	defaultGetStartramServices = func() error {
		return fmt.Errorf("orchestration startram service loader is not configured")
	}
	defaultLoadStartramRegions = func() error {
		return fmt.Errorf("orchestration startram region loader is not configured")
	}
)
