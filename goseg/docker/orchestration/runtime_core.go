package orchestration

import (
	"groundseg/config"
	"groundseg/internal/seams"
	"groundseg/structs"
)

type Runtime struct {
	RuntimeTransitionOps
	RuntimeHealthOps
	RuntimeStartupOps
}

// DockerTransitionRuntime contains the narrow dependencies required for transition-driven
// container workflows like start/restart/stop handling.
type DockerTransitionRuntime struct {
	LoadUrbitConfigFn           func(string) error
	UrbitConfFn                 func(string) structs.UrbitDocker
	UpdateUrbitFn               func(string, func(*structs.UrbitDocker) error) error
	UpdateUrbitRuntimeConfigFn  func(string, func(*structs.UrbitRuntimeConfig) error) error
	UpdateUrbitNetworkConfigFn  func(string, func(*structs.UrbitNetworkConfig) error) error
	UpdateUrbitScheduleConfigFn func(string, func(*structs.UrbitScheduleConfig) error) error
	UpdateUrbitFeatureConfigFn  func(string, func(*structs.UrbitFeatureConfig) error) error
	UpdateUrbitWebConfigFn      func(string, func(*structs.UrbitWebConfig) error) error
	UpdateUrbitBackupConfigFn   func(string, func(*structs.UrbitBackupConfig) error) error
	ClearLusCodeFn              func(string)
	StartContainerFn            func(string, string) (structs.ContainerState, error)
	GetContainerStateFn         func() map[string]structs.ContainerState
	UpdateContainerFn           func(string, structs.ContainerState)
}

// DockerHealthRuntime contains the narrow dependencies required for health checks and
// 502 recovery loops.
type DockerHealthRuntime struct {
	Check502SettingsSnapshotFn func() config.Check502Settings
	GetShipStatusFn            func([]string) (map[string]string, error)
	GetContainerNetworkFn      func(string) (string, error)
	GetLusCodeFn               func(string) (string, error)
	ShipSettingsSnapshotFn     func() config.ShipSettings
}

// NewDockerTransitionRuntime builds transition-focused dependencies from the general runtime
// seam, limiting coupling between lifecycle handlers and unrelated concerns.
func NewDockerTransitionRuntime(runtime Runtime) DockerTransitionRuntime {
	return DockerTransitionRuntime{
		LoadUrbitConfigFn:           runtime.LoadUrbitConfigFn,
		UrbitConfFn:                 runtime.UrbitConfFn,
		UpdateUrbitFn:               runtime.UpdateUrbitFn,
		UpdateUrbitRuntimeConfigFn:  runtime.UpdateUrbitRuntimeConfigFn,
		UpdateUrbitNetworkConfigFn:  runtime.UpdateUrbitNetworkConfigFn,
		UpdateUrbitScheduleConfigFn: runtime.UpdateUrbitScheduleConfigFn,
		UpdateUrbitFeatureConfigFn:  runtime.UpdateUrbitFeatureConfigFn,
		UpdateUrbitWebConfigFn:      runtime.UpdateUrbitWebConfigFn,
		UpdateUrbitBackupConfigFn:   runtime.UpdateUrbitBackupConfigFn,
		ClearLusCodeFn:              runtime.ClearLusCodeFn,
		StartContainerFn:            runtime.StartContainerFn,
		GetContainerStateFn:         runtime.GetContainerStateFn,
		UpdateContainerFn:           runtime.UpdateContainerStateFn,
	}
}

// NewDockerHealthRuntime builds health-loop focused dependencies from the general runtime seam.
func NewDockerHealthRuntime(runtime Runtime) DockerHealthRuntime {
	return DockerHealthRuntime{
		Check502SettingsSnapshotFn: runtime.Check502SettingsSnapshotFn,
		GetShipStatusFn:            runtime.GetShipStatusFn,
		GetContainerNetworkFn:      runtime.GetContainerNetworkFn,
		GetLusCodeFn:               runtime.GetLusCodeFn,
		ShipSettingsSnapshotFn:     runtime.ShipSettingsSnapshotFn,
	}
}

type StartupRuntime struct {
	StartupBootstrapOps
	StartupImageOps
	StartupLoadOps
}

func (runtime StartupRuntime) Initialize() error {
	if runtime.StartupBootstrapOps.Initialize == nil {
		return nil
	}
	return runtime.StartupBootstrapOps.Initialize()
}

type RuntimeOption func(*Runtime)

func WithContainerOps(ops RuntimeContainerOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeTransitionOps.RuntimeContainerOps = seams.Merge(runtime.RuntimeTransitionOps.RuntimeContainerOps, ops)
	}
}

func WithUrbitOps(ops RuntimeUrbitOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeTransitionOps.RuntimeUrbitOps = seams.Merge(runtime.RuntimeTransitionOps.RuntimeUrbitOps, ops)
	}
}

func WithSnapshotOps(ops RuntimeSnapshotOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeHealthOps.RuntimeSnapshotOps = seams.Merge(runtime.RuntimeHealthOps.RuntimeSnapshotOps, ops)
	}
}

func WithConfigOps(ops RuntimeConfigOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeStartupOps.RuntimeConfigOps = seams.Merge(runtime.RuntimeStartupOps.RuntimeConfigOps, ops)
	}
}

func WithLoadOps(ops RuntimeLoadOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeStartupOps.RuntimeLoadOps = seams.Merge(runtime.RuntimeStartupOps.RuntimeLoadOps, ops)
	}
}

func WithServiceOps(ops RuntimeServiceOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeStartupOps.RuntimeServiceOps = seams.Merge(runtime.RuntimeStartupOps.RuntimeServiceOps, ops)
	}
}

func WithRuntimeDependencies(dependencies Runtime) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeTransitionOps = seams.Merge(runtime.RuntimeTransitionOps, dependencies.RuntimeTransitionOps)
		runtime.RuntimeHealthOps = seams.Merge(runtime.RuntimeHealthOps, dependencies.RuntimeHealthOps)
		runtime.RuntimeStartupOps = seams.Merge(runtime.RuntimeStartupOps, dependencies.RuntimeStartupOps)
	}
}

func NewRuntimeWithDependencies(overrides Runtime) Runtime {
	return Runtime{
		RuntimeTransitionOps: seams.Merge(defaultRuntimeTransitionOps(), overrides.RuntimeTransitionOps),
		RuntimeHealthOps:     seams.Merge(defaultRuntimeHealthOps(), overrides.RuntimeHealthOps),
		RuntimeStartupOps:    seams.Merge(defaultRuntimeStartupOps(), overrides.RuntimeStartupOps),
	}
}

func NewRuntime(opts ...RuntimeOption) Runtime {
	overrides := Runtime{}
	for _, opt := range opts {
		if opt != nil {
			opt(&overrides)
		}
	}
	return NewRuntimeWithDependencies(overrides)
}

type StartupRuntimeOption func(*StartupRuntime)

func WithStartupBootstrapOps(ops StartupBootstrapOps) StartupRuntimeOption {
	return func(runtime *StartupRuntime) {
		runtime.StartupBootstrapOps = seams.Merge(runtime.StartupBootstrapOps, ops)
	}
}

func WithStartupImageOps(ops StartupImageOps) StartupRuntimeOption {
	return func(runtime *StartupRuntime) {
		runtime.StartupImageOps = seams.Merge(runtime.StartupImageOps, ops)
	}
}

func WithStartupLoadOps(ops StartupLoadOps) StartupRuntimeOption {
	return func(runtime *StartupRuntime) {
		runtime.StartupLoadOps = seams.Merge(runtime.StartupLoadOps, ops)
	}
}

func WithStartupRuntimeDependencies(dependencies StartupRuntime) StartupRuntimeOption {
	return func(runtime *StartupRuntime) {
		runtime.StartupBootstrapOps = seams.Merge(runtime.StartupBootstrapOps, dependencies.StartupBootstrapOps)
		runtime.StartupImageOps = seams.Merge(runtime.StartupImageOps, dependencies.StartupImageOps)
		runtime.StartupLoadOps = seams.Merge(runtime.StartupLoadOps, dependencies.StartupLoadOps)
	}
}

func NewStartupRuntimeWithDependencies(overrides StartupRuntime) StartupRuntime {
	return StartupRuntime{
		StartupBootstrapOps: seams.Merge(defaultStartupBootstrap(), overrides.StartupBootstrapOps),
		StartupImageOps:     seams.Merge(defaultStartupImage(), overrides.StartupImageOps),
		StartupLoadOps:      seams.Merge(defaultStartupLoad(), overrides.StartupLoadOps),
	}
}

func NewStartupRuntime(opts ...StartupRuntimeOption) StartupRuntime {
	overrides := StartupRuntime{}
	for _, opt := range opts {
		if opt != nil {
			opt(&overrides)
		}
	}
	return NewStartupRuntimeWithDependencies(overrides)
}
