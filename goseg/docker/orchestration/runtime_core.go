package orchestration

import (
	"groundseg/config"
	"groundseg/internal/seams"
)

type Runtime struct {
	RuntimeTransitionOps
	RuntimeHealthOps
	RuntimeStartupOps
	RuntimeStartramOps
}

func (runtime Runtime) UpdateConfig(opts ...config.ConfUpdateOption) error {
	if runtime.UpdateConfTypedFn == nil {
		return errConfUpdateMissing
	}
	return runtime.UpdateConfTypedFn(opts...)
}

func (runtime Runtime) WithWgOn(enabled bool) config.ConfUpdateOption {
	if runtime.WithWgOnFn == nil {
		return config.WithWgOn(enabled)
	}
	return runtime.WithWgOnFn(enabled)
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
		runtime.RuntimeTransitionOps = seams.Merge(runtime.RuntimeTransitionOps, RuntimeTransitionOpsFromContainerOps(ops))
	}
}

func WithUrbitOps(ops RuntimeUrbitOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeTransitionOps = seams.Merge(runtime.RuntimeTransitionOps, RuntimeTransitionOpsFromUrbitOps(ops))
	}
}

func WithSnapshotOps(ops RuntimeSnapshotOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeHealthOps = seams.Merge(runtime.RuntimeHealthOps, RuntimeHealthOps{
			ConfFn:                        ops.ConfFn,
			StartramSettingsSnapshotFn:    ops.StartramSettingsSnapshotFn,
			ShipSettingsSnapshotFn:        ops.ShipSettingsSnapshotFn,
			ShipRuntimeSettingsSnapshotFn: ops.ShipRuntimeSettingsSnapshotFn,
			GetStartramConfigFn:           ops.GetStartramConfigFn,
			Check502SettingsSnapshotFn:    ops.Check502SettingsSnapshotFn,
		})
	}
}

func WithConfigOps(ops RuntimeConfigOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeStartupOps = seams.Merge(runtime.RuntimeStartupOps, RuntimeStartupOps{
			UpdateConfTypedFn: ops.UpdateConfTypedFn,
			WithWgOnFn:        ops.WithWgOnFn,
			CycleWgKeyFn:      ops.CycleWgKeyFn,
			BarExitFn:         ops.BarExitFn,
		})
	}
}

func WithLoadOps(ops RuntimeLoadOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeStartupOps = seams.Merge(runtime.RuntimeStartupOps, RuntimeStartupOps{
			LoadWireguardFn: ops.LoadWireguardFn,
			LoadMCFn:        ops.LoadMCFn,
			LoadMinIOsFn:    ops.LoadMinIOsFn,
			LoadUrbitsFn:    ops.LoadUrbitsFn,
		})
	}
}

func WithServiceOps(ops RuntimeServiceOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeStartupOps = seams.Merge(runtime.RuntimeStartupOps, RuntimeStartupOps{
			SvcDeleteFn: ops.SvcDeleteFn,
		})
	}
}

func WithStartramOps(ops RuntimeStartramOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeStartramOps = seams.Merge(runtime.RuntimeStartramOps, ops)
	}
}

func WithRuntimeDependencies(dependencies Runtime) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeTransitionOps = seams.Merge(runtime.RuntimeTransitionOps, dependencies.RuntimeTransitionOps)
		runtime.RuntimeHealthOps = seams.Merge(runtime.RuntimeHealthOps, dependencies.RuntimeHealthOps)
		runtime.RuntimeStartupOps = seams.Merge(runtime.RuntimeStartupOps, dependencies.RuntimeStartupOps)
		runtime.RuntimeStartramOps = seams.Merge(runtime.RuntimeStartramOps, dependencies.RuntimeStartramOps)
	}
}

func NewRuntimeWithDependencies(overrides Runtime) Runtime {
	return Runtime{
		RuntimeTransitionOps: seams.Merge(defaultRuntimeTransitionOps(), overrides.RuntimeTransitionOps),
		RuntimeHealthOps:     seams.Merge(defaultRuntimeHealthOps(), overrides.RuntimeHealthOps),
		RuntimeStartupOps:    seams.Merge(defaultRuntimeStartupOps(), overrides.RuntimeStartupOps),
		RuntimeStartramOps:   seams.Merge(defaultRuntimeStartramOps(), overrides.RuntimeStartramOps),
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
