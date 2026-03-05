package orchestration

import (
	"groundseg/config"
	"groundseg/docker/registry"
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
	if runtime.InitializeFn == nil {
		return nil
	}
	return runtime.InitializeFn()
}

func (runtime StartupRuntime) GetLatestContainerInfo(containerType string) (registry.ImageDescriptor, error) {
	if runtime.GetLatestContainerInfoFn == nil {
		return registry.ImageDescriptor{}, seams.MissingRuntimeDependency("startup image metadata callback", "")
	}
	return runtime.GetLatestContainerInfoFn(containerType)
}

func (runtime StartupRuntime) PullImageIfNotExist(containerType string, image registry.ImageDescriptor) (bool, error) {
	if runtime.PullImageIfNotExistFn == nil {
		return false, seams.MissingRuntimeDependency("startup image pull callback", "")
	}
	return runtime.PullImageIfNotExistFn(containerType, image)
}

func (runtime StartupRuntime) LoadWireguard() error {
	if runtime.LoadWireguardFn == nil {
		return seams.MissingRuntimeDependency("startup wireguard loader callback", "")
	}
	return runtime.LoadWireguardFn()
}

func (runtime StartupRuntime) LoadMC() error {
	if runtime.LoadMCFn == nil {
		return seams.MissingRuntimeDependency("startup mc loader callback", "")
	}
	return runtime.LoadMCFn()
}

func (runtime StartupRuntime) LoadMinIOs() error {
	if runtime.LoadMinIOsFn == nil {
		return seams.MissingRuntimeDependency("startup minio loader callback", "")
	}
	return runtime.LoadMinIOsFn()
}

func (runtime StartupRuntime) LoadNetdata() error {
	if runtime.LoadNetdataFn == nil {
		return seams.MissingRuntimeDependency("startup netdata loader callback", "")
	}
	return runtime.LoadNetdataFn()
}

func (runtime StartupRuntime) LoadUrbits() error {
	if runtime.LoadUrbitsFn == nil {
		return seams.MissingRuntimeDependency("startup urbits loader callback", "")
	}
	return runtime.LoadUrbitsFn()
}

func (runtime StartupRuntime) LoadLlama() error {
	if runtime.LoadLlamaFn == nil {
		return seams.MissingRuntimeDependency("startup llama loader callback", "")
	}
	return runtime.LoadLlamaFn()
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
		*runtime = mergeRuntimeDependencies(*runtime, dependencies)
	}
}

func defaultRuntimeDependencies() Runtime {
	return Runtime{
		RuntimeContainerOps: defaultRuntimeContainerOps(),
		RuntimeUrbitOps:     defaultRuntimeUrbit(),
		RuntimeSnapshotOps:  defaultRuntimeSnapshot(),
		RuntimeHealthOps:    defaultRuntimeHealthOps(),
		RuntimeStartupOps:   defaultRuntimeStartupOps(),
		RuntimeStartramOps:  defaultRuntimeStartramOps(),
	}
}

func mergeRuntimeDependencies(base Runtime, overrides Runtime) Runtime {
	return Runtime{
		RuntimeContainerOps: seams.Merge(base.RuntimeContainerOps, overrides.RuntimeContainerOps),
		RuntimeUrbitOps:     seams.Merge(base.RuntimeUrbitOps, overrides.RuntimeUrbitOps),
		RuntimeSnapshotOps:  seams.Merge(base.RuntimeSnapshotOps, overrides.RuntimeSnapshotOps),
		RuntimeHealthOps:    seams.Merge(base.RuntimeHealthOps, overrides.RuntimeHealthOps),
		RuntimeStartupOps:   seams.Merge(base.RuntimeStartupOps, overrides.RuntimeStartupOps),
		RuntimeStartramOps:  seams.Merge(base.RuntimeStartramOps, overrides.RuntimeStartramOps),
	}
}

func NewRuntimeWithDependencies(overrides Runtime) Runtime {
	return mergeRuntimeDependencies(defaultRuntimeDependencies(), overrides)
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
		*runtime = mergeStartupRuntimeDependencies(*runtime, dependencies)
	}
}

func defaultStartupRuntimeDependencies() StartupRuntime {
	return StartupRuntime{
		StartupBootstrapOps: defaultStartupBootstrap(),
		StartupImageOps:     defaultStartupImage(),
		StartupLoadOps:      defaultStartupLoad(),
	}
}

func mergeStartupRuntimeDependencies(base StartupRuntime, overrides StartupRuntime) StartupRuntime {
	return StartupRuntime{
		StartupBootstrapOps: seams.Merge(base.StartupBootstrapOps, overrides.StartupBootstrapOps),
		StartupImageOps:     seams.Merge(base.StartupImageOps, overrides.StartupImageOps),
		StartupLoadOps:      seams.Merge(base.StartupLoadOps, overrides.StartupLoadOps),
	}
}

func NewStartupRuntimeWithDependencies(overrides StartupRuntime) StartupRuntime {
	return mergeStartupRuntimeDependencies(defaultStartupRuntimeDependencies(), overrides)
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
