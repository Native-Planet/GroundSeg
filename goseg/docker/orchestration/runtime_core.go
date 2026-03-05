package orchestration

import (
	"groundseg/config"
	"groundseg/internal/seams"
)

type Runtime struct {
	RuntimeContainerOps
	RuntimeUrbitOps
	RuntimeSnapshotOps
	RuntimeHealthOps
	RuntimeStartupOps
	RuntimeStartramOps
}

func (runtime Runtime) UpdateConfig(opts ...config.ConfigUpdateOption) error {
	if runtime.UpdateConfigTypedFn == nil {
		return errConfUpdateMissing
	}
	return runtime.UpdateConfigTypedFn(opts...)
}

func (runtime Runtime) WithWgOn(enabled bool) config.ConfigUpdateOption {
	if runtime.WithWireguardEnabledFn == nil {
		return config.WithWgOn(enabled)
	}
	return runtime.WithWireguardEnabledFn(enabled)
}

type StartupRuntime struct {
	StartupBootstrapOps
	StartupImageOps
	StartupLoadOps
}

func (runtime StartupRuntime) Initialize() error {
	initialize := runtime.StartupBootstrapOps.initializeCallback()
	if initialize == nil {
		return nil
	}
	return initialize()
}

type RuntimeOption func(*Runtime)

func WithContainerOps(ops RuntimeContainerOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeContainerOps = seams.Merge(runtime.RuntimeContainerOps, ops)
	}
}

func WithUrbitOps(ops RuntimeUrbitOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeUrbitOps = seams.Merge(runtime.RuntimeUrbitOps, ops)
	}
}

func WithSnapshotOps(ops RuntimeSnapshotOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeSnapshotOps = seams.Merge(runtime.RuntimeSnapshotOps, ops)
		runtime.RuntimeHealthOps = seams.Merge(runtime.RuntimeHealthOps, RuntimeHealthOps{
			HealthShipSettingsSnapshotFn:     ops.ShipSettingsSnapshotFn,
			HealthCheck502SettingsSnapshotFn: ops.Check502SettingsSnapshotFn,
		})
	}
}

func WithRuntimeStartupOps(ops RuntimeStartupOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeStartupOps = seams.Merge(runtime.RuntimeStartupOps, ops)
	}
}

func WithStartramOps(ops RuntimeStartramOps) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeStartramOps = seams.Merge(runtime.RuntimeStartramOps, ops)
	}
}

func WithRuntimeDependencies(dependencies Runtime) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.RuntimeContainerOps = seams.Merge(runtime.RuntimeContainerOps, dependencies.RuntimeContainerOps)
		runtime.RuntimeUrbitOps = seams.Merge(runtime.RuntimeUrbitOps, dependencies.RuntimeUrbitOps)
		runtime.RuntimeSnapshotOps = seams.Merge(runtime.RuntimeSnapshotOps, dependencies.RuntimeSnapshotOps)
		runtime.RuntimeHealthOps = seams.Merge(runtime.RuntimeHealthOps, dependencies.RuntimeHealthOps)
		runtime.RuntimeStartupOps = seams.Merge(runtime.RuntimeStartupOps, dependencies.RuntimeStartupOps)
		runtime.RuntimeStartramOps = seams.Merge(runtime.RuntimeStartramOps, dependencies.RuntimeStartramOps)
	}
}

func NewRuntimeWithDependencies(overrides Runtime) Runtime {
	return Runtime{
		RuntimeContainerOps: seams.Merge(defaultRuntimeContainerOps(), overrides.RuntimeContainerOps),
		RuntimeUrbitOps:     seams.Merge(defaultRuntimeUrbit(), overrides.RuntimeUrbitOps),
		RuntimeSnapshotOps:  seams.Merge(defaultRuntimeSnapshot(), overrides.RuntimeSnapshotOps),
		RuntimeHealthOps:    seams.Merge(defaultRuntimeHealthOps(), overrides.RuntimeHealthOps),
		RuntimeStartupOps:   seams.Merge(defaultRuntimeStartupOps(), overrides.RuntimeStartupOps),
		RuntimeStartramOps:  seams.Merge(defaultRuntimeStartramOps(), overrides.RuntimeStartramOps),
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
	return normalizeStartupRuntimeCallbacks(StartupRuntime{
		StartupBootstrapOps: seams.Merge(defaultStartupBootstrap(), overrides.StartupBootstrapOps),
		StartupImageOps:     seams.Merge(defaultStartupImage(), overrides.StartupImageOps),
		StartupLoadOps:      seams.Merge(defaultStartupLoad(), overrides.StartupLoadOps),
	})
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

func normalizeStartupRuntimeCallbacks(runtime StartupRuntime) StartupRuntime {
	runtime.StartupBootstrapOps = normalizeStartupBootstrapOps(runtime.StartupBootstrapOps)
	runtime.StartupImageOps = normalizeStartupImageOps(runtime.StartupImageOps)
	runtime.StartupLoadOps = normalizeStartupLoadOps(runtime.StartupLoadOps)
	return runtime
}

func normalizeStartupBootstrapOps(ops StartupBootstrapOps) StartupBootstrapOps {
	if ops.Initialize != nil {
		ops.InitializeFn = ops.Initialize
	}
	if ops.InitializeFn == nil {
		ops.InitializeFn = ops.Initialize
	}
	if ops.Initialize == nil {
		ops.Initialize = ops.InitializeFn
	}
	return ops
}

func normalizeStartupImageOps(ops StartupImageOps) StartupImageOps {
	if ops.GetLatestContainerInfo != nil {
		ops.GetLatestContainerInfoFn = ops.GetLatestContainerInfo
	}
	if ops.GetLatestContainerInfoFn == nil {
		ops.GetLatestContainerInfoFn = ops.GetLatestContainerInfo
	}
	if ops.GetLatestContainerInfo == nil {
		ops.GetLatestContainerInfo = ops.GetLatestContainerInfoFn
	}
	if ops.PullImageIfNotExist != nil {
		ops.PullImageIfNotExistFn = ops.PullImageIfNotExist
	}
	if ops.PullImageIfNotExistFn == nil {
		ops.PullImageIfNotExistFn = ops.PullImageIfNotExist
	}
	if ops.PullImageIfNotExist == nil {
		ops.PullImageIfNotExist = ops.PullImageIfNotExistFn
	}
	return ops
}

func normalizeStartupLoadOps(ops StartupLoadOps) StartupLoadOps {
	if ops.LoadWireguard != nil {
		ops.LoadWireguardFn = ops.LoadWireguard
	}
	if ops.LoadWireguardFn == nil {
		ops.LoadWireguardFn = ops.LoadWireguard
	}
	if ops.LoadWireguard == nil {
		ops.LoadWireguard = ops.LoadWireguardFn
	}
	if ops.LoadMC != nil {
		ops.LoadMCFn = ops.LoadMC
	}
	if ops.LoadMCFn == nil {
		ops.LoadMCFn = ops.LoadMC
	}
	if ops.LoadMC == nil {
		ops.LoadMC = ops.LoadMCFn
	}
	if ops.LoadMinIOs != nil {
		ops.LoadMinIOsFn = ops.LoadMinIOs
	}
	if ops.LoadMinIOsFn == nil {
		ops.LoadMinIOsFn = ops.LoadMinIOs
	}
	if ops.LoadMinIOs == nil {
		ops.LoadMinIOs = ops.LoadMinIOsFn
	}
	if ops.LoadNetdata != nil {
		ops.LoadNetdataFn = ops.LoadNetdata
	}
	if ops.LoadNetdataFn == nil {
		ops.LoadNetdataFn = ops.LoadNetdata
	}
	if ops.LoadNetdata == nil {
		ops.LoadNetdata = ops.LoadNetdataFn
	}
	if ops.LoadUrbits != nil {
		ops.LoadUrbitsFn = ops.LoadUrbits
	}
	if ops.LoadUrbitsFn == nil {
		ops.LoadUrbitsFn = ops.LoadUrbits
	}
	if ops.LoadUrbits == nil {
		ops.LoadUrbits = ops.LoadUrbitsFn
	}
	if ops.LoadLlama != nil {
		ops.LoadLlamaFn = ops.LoadLlama
	}
	if ops.LoadLlamaFn == nil {
		ops.LoadLlamaFn = ops.LoadLlama
	}
	if ops.LoadLlama == nil {
		ops.LoadLlama = ops.LoadLlamaFn
	}
	return ops
}
