package orchestration

import (
	"crypto/rand"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/network"
	"groundseg/docker/orchestration/container"
	"groundseg/internal/seams"
	"groundseg/startram"
	"groundseg/structs"
	"os"
	"time"
)

var (
	runtimeStartramSettingsSnapshotFn    = config.StartramSettingsSnapshot
	runtimeShipSettingsSnapshotFn        = config.ShipSettingsSnapshot
	runtimeCheck502SettingsSnapshotFn    = config.Check502SettingsSnapshot
	runtimeBasePathFn                    = config.BasePath
	runtimeArchitectureFn                = config.Architecture
	runtimeDebugModeFn                   = config.DebugMode
	runtimeDockerDirFn                   = config.DockerDir
	runtimeShipRuntimeSettingsSnapshotFn = config.ShipRuntimeSettingsSnapshot
)

type Runtime struct {
	RuntimeTransitionOps
	RuntimeHealthOps
	RuntimeStartupOps
}

type RuntimeDependencies struct {
	RuntimeTransitionOps
	RuntimeHealthOps
	RuntimeStartupOps
}

// DockerTransitionRuntime contains the narrow dependencies required for transition-driven
// container workflows like start/restart/stop handling.
type DockerTransitionRuntime struct {
	LoadUrbitConfigFn   func(string) error
	UrbitConfFn         func(string) structs.UrbitDocker
	UpdateUrbitFn       func(string, func(*structs.UrbitDocker) error) error
	ClearLusCodeFn      func(string)
	StartContainerFn    func(string, string) (structs.ContainerState, error)
	GetContainerStateFn func() map[string]structs.ContainerState
	UpdateContainerFn   func(string, structs.ContainerState)
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
		LoadUrbitConfigFn:   runtime.LoadUrbitConfigFn,
		UrbitConfFn:         runtime.UrbitConfFn,
		UpdateUrbitFn:       runtime.UpdateUrbitFn,
		ClearLusCodeFn:      runtime.ClearLusCodeFn,
		StartContainerFn:    runtime.StartContainerFn,
		GetContainerStateFn: runtime.GetContainerStateFn,
		UpdateContainerFn:   runtime.UpdateContainerStateFn,
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

type StartupRuntimeDependencies struct {
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

type RuntimeOption func(*RuntimeDependencies)

func WithContainerOps(ops RuntimeContainerOps) RuntimeOption {
	return func(runtime *RuntimeDependencies) {
		runtime.RuntimeTransitionOps.RuntimeContainerOps = seams.Merge(runtime.RuntimeTransitionOps.RuntimeContainerOps, ops)
	}
}

func WithUrbitOps(ops RuntimeUrbitOps) RuntimeOption {
	return func(runtime *RuntimeDependencies) {
		runtime.RuntimeTransitionOps.RuntimeUrbitOps = seams.Merge(runtime.RuntimeTransitionOps.RuntimeUrbitOps, ops)
	}
}

func WithSnapshotOps(ops RuntimeSnapshotOps) RuntimeOption {
	return func(runtime *RuntimeDependencies) {
		runtime.RuntimeHealthOps.RuntimeSnapshotOps = seams.Merge(runtime.RuntimeHealthOps.RuntimeSnapshotOps, ops)
	}
}

func WithConfigOps(ops RuntimeConfigOps) RuntimeOption {
	return func(runtime *RuntimeDependencies) {
		runtime.RuntimeStartupOps.RuntimeConfigOps = seams.Merge(runtime.RuntimeStartupOps.RuntimeConfigOps, ops)
	}
}

func WithLoadOps(ops RuntimeLoadOps) RuntimeOption {
	return func(runtime *RuntimeDependencies) {
		runtime.RuntimeStartupOps.RuntimeLoadOps = seams.Merge(runtime.RuntimeStartupOps.RuntimeLoadOps, ops)
	}
}

func WithServiceOps(ops RuntimeServiceOps) RuntimeOption {
	return func(runtime *RuntimeDependencies) {
		runtime.RuntimeStartupOps.RuntimeServiceOps = seams.Merge(runtime.RuntimeStartupOps.RuntimeServiceOps, ops)
	}
}

func WithRuntimeDependencies(dependencies RuntimeDependencies) RuntimeOption {
	return func(runtime *RuntimeDependencies) {
		runtime.RuntimeTransitionOps = seams.Merge(runtime.RuntimeTransitionOps, dependencies.RuntimeTransitionOps)
		runtime.RuntimeHealthOps = seams.Merge(runtime.RuntimeHealthOps, dependencies.RuntimeHealthOps)
		runtime.RuntimeStartupOps = seams.Merge(runtime.RuntimeStartupOps, dependencies.RuntimeStartupOps)
	}
}

func NewRuntimeWithDependencies(overrides RuntimeDependencies) Runtime {
	return Runtime{
		RuntimeTransitionOps: seams.Merge(defaultRuntimeTransitionOps(), overrides.RuntimeTransitionOps),
		RuntimeHealthOps:     seams.Merge(defaultRuntimeHealthOps(), overrides.RuntimeHealthOps),
		RuntimeStartupOps:    seams.Merge(defaultRuntimeStartupOps(), overrides.RuntimeStartupOps),
	}
}

func NewRuntime(opts ...RuntimeOption) Runtime {
	overrides := RuntimeDependencies{}
	for _, opt := range opts {
		if opt != nil {
			opt(&overrides)
		}
	}
	return NewRuntimeWithDependencies(overrides)
}

type StartupRuntimeOption func(*StartupRuntimeDependencies)

func WithStartupBootstrapOps(ops StartupBootstrapOps) StartupRuntimeOption {
	return func(runtime *StartupRuntimeDependencies) {
		runtime.StartupBootstrapOps = seams.Merge(runtime.StartupBootstrapOps, ops)
	}
}

func WithStartupImageOps(ops StartupImageOps) StartupRuntimeOption {
	return func(runtime *StartupRuntimeDependencies) {
		runtime.StartupImageOps = seams.Merge(runtime.StartupImageOps, ops)
	}
}

func WithStartupLoadOps(ops StartupLoadOps) StartupRuntimeOption {
	return func(runtime *StartupRuntimeDependencies) {
		runtime.StartupLoadOps = seams.Merge(runtime.StartupLoadOps, ops)
	}
}

func WithStartupRuntimeDependencies(dependencies StartupRuntimeDependencies) StartupRuntimeOption {
	return func(runtime *StartupRuntimeDependencies) {
		runtime.StartupBootstrapOps = seams.Merge(runtime.StartupBootstrapOps, dependencies.StartupBootstrapOps)
		runtime.StartupImageOps = seams.Merge(runtime.StartupImageOps, dependencies.StartupImageOps)
		runtime.StartupLoadOps = seams.Merge(runtime.StartupLoadOps, dependencies.StartupLoadOps)
	}
}

func NewStartupRuntimeWithDependencies(overrides StartupRuntimeDependencies) StartupRuntime {
	return StartupRuntime{
		StartupBootstrapOps: seams.Merge(defaultStartupBootstrap(), overrides.StartupBootstrapOps),
		StartupImageOps:     seams.Merge(defaultStartupImage(), overrides.StartupImageOps),
		StartupLoadOps:      seams.Merge(defaultStartupLoad(), overrides.StartupLoadOps),
	}
}

func NewStartupRuntime(opts ...StartupRuntimeOption) StartupRuntime {
	overrides := StartupRuntimeDependencies{}
	for _, opt := range opts {
		if opt != nil {
			opt(&overrides)
		}
	}
	return NewStartupRuntimeWithDependencies(overrides)
}

type RuntimeTransitionOps struct {
	RuntimeContainerOps
	RuntimeUrbitOps
}

type RuntimeHealthOps struct {
	RuntimeSnapshotOps
}

type RuntimeStartupOps struct {
	RuntimeConfigOps
	RuntimeLoadOps
	RuntimeServiceOps
}

type RuntimeContainerOps struct {
	StartContainerFn       func(name string, ctype string) (structs.ContainerState, error)
	StopContainerByNameFn  func(name string) error
	RestartContainerFn     func(name string) error
	DeleteContainerFn      func(name string) error
	GetContainerStateFn    func() map[string]structs.ContainerState
	UpdateContainerStateFn func(name string, state structs.ContainerState)
	GetShipStatusFn        func([]string) (map[string]string, error)
	WaitForShipExitFn      func(name string, timeout time.Duration) error
}

type RuntimeUrbitOps struct {
	LoadUrbitConfigFn     func(string) error
	UrbitConfFn           func(string) structs.UrbitDocker
	UpdateUrbitFn         func(string, func(*structs.UrbitDocker) error) error
	GetContainerNetworkFn func(string) (string, error)
	GetLusCodeFn          func(string) (string, error)
	ClearLusCodeFn        func(string)
}

type RuntimeSnapshotOps struct {
	StartramSettingsSnapshotFn func() config.StartramSettings
	ShipSettingsSnapshotFn     func() config.ShipSettings
	GetStartramConfigFn        func() structs.StartramRetrieve
	Check502SettingsSnapshotFn func() config.Check502Settings
}

type RuntimeConfigOps struct {
	UpdateConfTypedFn func(...config.ConfUpdateOption) error
	WithWgOnFn        func(bool) config.ConfUpdateOption
	CycleWgKeyFn      func() error
	BarExitFn         func(string) error
}

type RuntimeLoadOps struct {
	LoadWireguardFn func() error
	LoadMCFn        func() error
	LoadMinIOsFn    func() error
	LoadUrbitsFn    func() error
}

type RuntimeServiceOps struct {
	SvcDeleteFn func(patp string, kind string) error
}

type StartupBootstrapOps struct {
	Initialize func() error
}

type StartupImageOps struct {
	GetLatestContainerInfo func(string) (map[string]string, error)
	PullImageIfNotExist    func(string, map[string]string) (bool, error)
}

type StartupLoadOps struct {
	LoadWireguard func() error
	LoadMC        func() error
	LoadMinIOs    func() error
	LoadNetdata   func() error
	LoadUrbits    func() error
	LoadLlama     func() error
}

func defaultRuntimeTransitionOps() RuntimeTransitionOps {
	return RuntimeTransitionOps{
		RuntimeContainerOps: defaultRuntimeContainerOps(),
		RuntimeUrbitOps:     defaultRuntimeUrbit(),
	}
}

func defaultRuntimeHealthOps() RuntimeHealthOps {
	return RuntimeHealthOps{
		RuntimeSnapshotOps: defaultRuntimeSnapshot(),
	}
}

func defaultRuntimeStartupOps() RuntimeStartupOps {
	return RuntimeStartupOps{
		RuntimeConfigOps:  defaultRuntimeConfig(),
		RuntimeLoadOps:    defaultRuntimeLoad(),
		RuntimeServiceOps: defaultRuntimeService(),
	}
}

func defaultRuntimeContainerOps() RuntimeContainerOps {
	return RuntimeContainerOps{
		StartContainerFn:       StartContainer,
		StopContainerByNameFn:  StopContainerByName,
		RestartContainerFn:     RestartContainer,
		DeleteContainerFn:      DeleteContainer,
		GetContainerStateFn:    config.GetContainerState,
		UpdateContainerStateFn: config.UpdateContainerState,
		GetShipStatusFn:        GetShipStatus,
		WaitForShipExitFn:      WaitForShipExit,
	}
}

func defaultRuntimeUrbit() RuntimeUrbitOps {
	return RuntimeUrbitOps{
		LoadUrbitConfigFn: config.LoadUrbitConfig,
		UrbitConfFn:       config.UrbitConf,
		UpdateUrbitFn:     config.UpdateUrbit,
		GetContainerNetworkFn: func(name string) (string, error) {
			return network.NewNetworkRuntime().GetContainerNetwork(name)
		},
		GetLusCodeFn:   click.GetLusCode,
		ClearLusCodeFn: click.ClearLusCode,
	}
}

func defaultRuntimeSnapshot() RuntimeSnapshotOps {
	return RuntimeSnapshotOps{
		StartramSettingsSnapshotFn: runtimeStartramSettingsSnapshotFn,
		ShipSettingsSnapshotFn:     runtimeShipSettingsSnapshotFn,
		GetStartramConfigFn:        config.GetStartramConfig,
		Check502SettingsSnapshotFn: runtimeCheck502SettingsSnapshotFn,
	}
}

func defaultRuntimeConfig() RuntimeConfigOps {
	return RuntimeConfigOps{
		UpdateConfTypedFn: config.UpdateConfTyped,
		WithWgOnFn:        config.WithWgOn,
		CycleWgKeyFn:      config.CycleWgKey,
		BarExitFn:         click.BarExit,
	}
}

func defaultRuntimeLoad() RuntimeLoadOps {
	return RuntimeLoadOps{
		LoadWireguardFn: func() error {
			return wireguardRuntimeFromDocker(newDockerRuntime()).LoadWireguard()
		},
		LoadMCFn:     LoadMC,
		LoadMinIOsFn: LoadMinIOs,
		LoadUrbitsFn: LoadUrbits,
	}
}

func defaultRuntimeService() RuntimeServiceOps {
	return RuntimeServiceOps{
		SvcDeleteFn: startram.SvcDelete,
	}
}

func defaultStartupBootstrap() StartupBootstrapOps {
	return StartupBootstrapOps{
		Initialize: Initialize,
	}
}

func defaultStartupImage() StartupImageOps {
	return StartupImageOps{
		GetLatestContainerInfo: GetLatestContainerInfo,
		PullImageIfNotExist:    PullImageIfNotExist,
	}
}

func defaultStartupLoad() StartupLoadOps {
	return StartupLoadOps{
		LoadWireguard: func() error {
			return wireguardRuntimeFromDocker(newDockerRuntime()).LoadWireguard()
		},
		LoadMC:      LoadMC,
		LoadMinIOs:  LoadMinIOs,
		LoadNetdata: LoadNetdata,
		LoadUrbits:  LoadUrbits,
		LoadLlama:   LoadLlama,
	}
}

var (
	defaultGetStartramServices = func() error {
		return fmt.Errorf("orchestration startram service loader is not configured")
	}
	defaultLoadStartramRegions = func() error {
		return fmt.Errorf("orchestration startram region loader is not configured")
	}
)

// StartramRuntime captures only the orchestration and state dependencies
// required by startram workflows.
type StartramRuntime struct {
	GetStartramServicesFn      func() error
	LoadStartramRegionsFn      func() error
	StartramSettingsSnapshotFn func() config.StartramSettings
	ShipSettingsSnapshotFn     func() config.ShipSettings
	GetStartramConfigFn        func() structs.StartramRetrieve
	UpdateConfTypedFn          func(...config.ConfUpdateOption) error
	WithWgOnFn                 func(bool) config.ConfUpdateOption
	CycleWgKeyFn               func() error
	StopContainerByNameFn      func(name string) error
	UrbitConfFn                func(string) structs.UrbitDocker
	StartContainerFn           func(name string, ctype string) (structs.ContainerState, error)
	DeleteContainerFn          func(name string) error
	LoadWireguardFn            func() error
	LoadMCFn                   func() error
	LoadMinIOsFn               func() error
	SvcDeleteFn                func(patp string, kind string) error
	UpdateUrbitFn              func(patp string, update func(*structs.UrbitDocker) error) error
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
		GetStartramServicesFn:      defaultGetStartramServices,
		LoadStartramRegionsFn:      defaultLoadStartramRegions,
		StartramSettingsSnapshotFn: rt.StartramSettingsSnapshotFn,
		ShipSettingsSnapshotFn:     rt.ShipSettingsSnapshotFn,
		GetStartramConfigFn:        rt.GetStartramConfigFn,
		UpdateConfTypedFn:          rt.UpdateConfTypedFn,
		WithWgOnFn:                 rt.WithWgOnFn,
		CycleWgKeyFn:               rt.CycleWgKeyFn,
		StopContainerByNameFn:      rt.StopContainerByNameFn,
		UrbitConfFn:                rt.UrbitConfFn,
		StartContainerFn:           rt.StartContainerFn,
		DeleteContainerFn:          rt.DeleteContainerFn,
		LoadWireguardFn:            rt.LoadWireguardFn,
		LoadMCFn:                   rt.LoadMCFn,
		LoadMinIOsFn:               rt.LoadMinIOsFn,
		SvcDeleteFn:                rt.SvcDeleteFn,
		UpdateUrbitFn:              rt.UpdateUrbitFn,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&sr)
		}
	}
	return sr
}

type dockerRuntime struct {
	contextOps   dockerRuntimeContextOps
	fileOps      dockerRuntimeFileOps
	containerOps dockerRuntimeContainerOps
	imageOps     dockerRuntimeImageOps
	configOps    dockerRuntimeConfigOps
	urbitOps     dockerRuntimeUrbitOps
	wireguardOps dockerRuntimeWireguardOps
	netdataOps   dockerRuntimeNetdataOps
	minioOps     dockerRuntimeMinioOps
	commandOps   dockerRuntimeCommandOps
	volumeOps    dockerRuntimeVolumeOps
	timerOps     dockerRuntimeTimerOps
}

type WireguardRuntime struct {
	BasePathFn                func() string
	DockerDirFn               func() string
	OpenFn                    func(string) (*os.File, error)
	ReadFileFn                func(string) ([]byte, error)
	WriteFileFn               func(string, []byte, os.FileMode) error
	MkdirAllFn                func(string, os.FileMode) error
	StartContainerFn          func(string, string) (structs.ContainerState, error)
	UpdateContainerFn         func(string, structs.ContainerState)
	GetWgConfFn               func() (structs.WgConfig, error)
	GetWgConfBlobFn           func() (string, error)
	GetWgPrivkeyFn            func() string
	GetLatestContainerInfoFn  func(string) (map[string]string, error)
	GetLatestContainerImageFn func(string) (string, error)
	CopyFileToVolumeFn        func(string, string, string, string, volumeWriterImageSelector) error
	VolumeExistsFn            func(string) (bool, error)
	CreateVolumeFn            func(string) error
	CreateDefaultWGConfFn     func() error
	WriteWgConfFn             func(WireguardRuntime) error
}

func wireguardRuntimeFromDocker(rt dockerRuntime) WireguardRuntime {
	return WireguardRuntime{
		BasePathFn:                rt.contextOps.BasePathFn,
		DockerDirFn:               rt.contextOps.DockerDirFn,
		OpenFn:                    rt.fileOps.OpenFn,
		ReadFileFn:                rt.fileOps.ReadFileFn,
		WriteFileFn:               rt.fileOps.WriteFileFn,
		MkdirAllFn:                rt.fileOps.MkdirAllFn,
		StartContainerFn:          rt.containerOps.StartContainerFn,
		UpdateContainerFn:         rt.containerOps.UpdateContainerStateFn,
		GetWgConfFn:               rt.wireguardOps.GetWgConfFn,
		GetWgConfBlobFn:           rt.wireguardOps.GetWgConfBlobFn,
		GetWgPrivkeyFn:            rt.wireguardOps.GetWgPrivkeyFn,
		GetLatestContainerInfoFn:  rt.imageOps.GetLatestContainerInfoFn,
		GetLatestContainerImageFn: rt.imageOps.GetLatestContainerImageFn,
		CopyFileToVolumeFn:        rt.commandOps.CopyFileToVolumeFn,
		VolumeExistsFn:            rt.volumeOps.VolumeExistsFn,
		CreateVolumeFn:            rt.volumeOps.CreateVolumeFn,
		CreateDefaultWGConfFn:     rt.wireguardOps.CreateDefaultWGConfFn,
		WriteWgConfFn:             rt.wireguardOps.WriteWgConfFn,
	}
}

type UrbitRuntime struct {
	ShipSettingsSnapshotFn    func() config.ShipSettings
	RuntimeSettingsSnapshotFn func() config.ShipRuntimeSettings
	LoadUrbitConfigFn         func(string) error
	UrbitConfFn               func(string) structs.UrbitDocker
	UpdateUrbitFn             func(string, func(*structs.UrbitDocker) error) error
	StartContainerFn          func(string, string) (structs.ContainerState, error)
	CreateContainerFn         func(string, string) (structs.ContainerState, error)
	UpdateContainerStateFn    func(string, structs.ContainerState)
	GetLatestContainerInfoFn  func(string) (map[string]string, error)
	ArchitectureFn            func() string
	DockerDirFn               func() string
	WriteFileFn               func(string, []byte, os.FileMode) error
}

func urbitRuntimeFromDocker(rt dockerRuntime) UrbitRuntime {
	return UrbitRuntime{
		ShipSettingsSnapshotFn:    rt.configOps.ShipSettingsSnapshotFn,
		RuntimeSettingsSnapshotFn: rt.configOps.RuntimeSettingsSnapshotFn,
		LoadUrbitConfigFn:         rt.urbitOps.LoadUrbitConfigFn,
		UrbitConfFn:               rt.urbitOps.UrbitConfFn,
		UpdateUrbitFn:             rt.urbitOps.UpdateUrbitFn,
		StartContainerFn:          rt.containerOps.StartContainerFn,
		CreateContainerFn:         rt.containerOps.CreateContainerFn,
		UpdateContainerStateFn:    rt.containerOps.UpdateContainerStateFn,
		GetLatestContainerInfoFn:  rt.imageOps.GetLatestContainerInfoFn,
		ArchitectureFn:            rt.contextOps.ArchitectureFn,
		DockerDirFn:               rt.contextOps.DockerDirFn,
		WriteFileFn:               rt.fileOps.WriteFileFn,
	}
}

type dockerRuntimeContextOps struct {
	BasePathFn     func() string
	ArchitectureFn func() string
	DebugModeFn    func() bool
	DockerDirFn    func() string
}

type dockerRuntimeFileOps struct {
	OpenFn      func(string) (*os.File, error)
	ReadFileFn  func(string) ([]byte, error)
	WriteFileFn func(string, []byte, os.FileMode) error
	MkdirAllFn  func(string, os.FileMode) error
}

type dockerRuntimeContainerOps struct {
	StartContainerFn            func(string, string) (structs.ContainerState, error)
	StopContainerByNameFn       func(string) error
	CreateContainerFn           func(string, string) (structs.ContainerState, error)
	UpdateContainerStateFn      func(string, structs.ContainerState)
	GetContainerRunningStatusFn func(string) (string, error)
	AddOrGetNetworkFn           func(string) (string, error)
}

type dockerRuntimeImageOps struct {
	GetLatestContainerInfoFn  func(string) (map[string]string, error)
	GetLatestContainerImageFn func(string) (string, error)
}

type dockerRuntimeConfigOps struct {
	ConfFn                    func() structs.SysConfig
	ShipSettingsSnapshotFn    func() config.ShipSettings
	RuntimeSettingsSnapshotFn func() config.ShipRuntimeSettings
}

type dockerRuntimeUrbitOps struct {
	LoadUrbitConfigFn func(string) error
	UrbitConfFn       func(string) structs.UrbitDocker
	UpdateUrbitFn     func(string, func(*structs.UrbitDocker) error) error
	UrbitConfAllFn    func() map[string]structs.UrbitDocker
}

type dockerRuntimeWireguardOps struct {
	CreateDefaultWGConfFn func() error
	GetWgConfFn           func() (structs.WgConfig, error)
	GetWgConfBlobFn       func() (string, error)
	GetWgPrivkeyFn        func() string
	WriteWgConfFn         func(WireguardRuntime) error
}

type dockerRuntimeNetdataOps struct {
	CreateDefaultNetdataConfFn func() error
	WriteNDConfFn              func(container.NetdataRuntime) error
}

type dockerRuntimeMinioOps struct {
	CreateDefaultMcConfFn func() error
	SetMinIOPasswordFn    func(string, string) error
	GetMinIOPasswordFn    func(string) (string, error)
}

type dockerRuntimeCommandOps struct {
	ExecDockerCommandFn     func(string, []string) (string, error)
	ExecDockerCommandExitFn func(string, []string) (string, int, error)
	RandReadFn              func([]byte) (int, error)
	CopyFileToVolumeFn      func(string, string, string, string, volumeWriterImageSelector) error
}

type dockerRuntimeVolumeOps struct {
	VolumeExistsFn func(string) (bool, error)
	CreateVolumeFn func(string) error
}

type dockerRuntimeTimerOps struct {
	SleepFn        func(time.Duration)
	PollIntervalFn func() time.Duration
}

func newDockerRuntime() dockerRuntime {
	return dockerRuntime{
		contextOps: dockerRuntimeContextOps{
			BasePathFn: func() string {
				return runtimeBasePathFn()
			},
			ArchitectureFn: func() string {
				return runtimeArchitectureFn()
			},
			DebugModeFn: func() bool {
				return runtimeDebugModeFn()
			},
			DockerDirFn: func() string {
				return runtimeDockerDirFn()
			},
		},
		fileOps: dockerRuntimeFileOps{
			OpenFn:      os.Open,
			ReadFileFn:  os.ReadFile,
			WriteFileFn: os.WriteFile,
			MkdirAllFn:  os.MkdirAll,
		},
		containerOps: dockerRuntimeContainerOps{
			StartContainerFn:            StartContainer,
			StopContainerByNameFn:       StopContainerByName,
			CreateContainerFn:           CreateContainer,
			UpdateContainerStateFn:      config.UpdateContainerState,
			GetContainerRunningStatusFn: GetContainerRunningStatus,
			AddOrGetNetworkFn: func(networkName string) (string, error) {
				return network.NewNetworkRuntime().AddOrGetNetwork(networkName)
			},
		},
		imageOps: dockerRuntimeImageOps{
			GetLatestContainerInfoFn:  GetLatestContainerInfo,
			GetLatestContainerImageFn: latestContainerImage,
		},
		configOps: dockerRuntimeConfigOps{
			ConfFn: config.Conf,
			ShipSettingsSnapshotFn: func() config.ShipSettings {
				return runtimeShipSettingsSnapshotFn()
			},
			RuntimeSettingsSnapshotFn: func() config.ShipRuntimeSettings {
				return runtimeShipRuntimeSettingsSnapshotFn()
			},
		},
		urbitOps: dockerRuntimeUrbitOps{
			LoadUrbitConfigFn: config.LoadUrbitConfig,
			UrbitConfFn:       config.UrbitConf,
			UpdateUrbitFn:     config.UpdateUrbit,
			UrbitConfAllFn:    config.UrbitConfAll,
		},
		wireguardOps: dockerRuntimeWireguardOps{
			CreateDefaultWGConfFn: config.CreateDefaultWGConf,
			GetWgConfFn:           config.GetWgConf,
			GetWgConfBlobFn: func() (string, error) {
				return config.GetStartramConfig().Conf, nil
			},
			GetWgPrivkeyFn: config.GetWgPrivkey,
			WriteWgConfFn:  func(rt WireguardRuntime) error { return rt.WriteWgConf() },
		},
		netdataOps: dockerRuntimeNetdataOps{
			CreateDefaultNetdataConfFn: config.CreateDefaultNetdataConf,
			WriteNDConfFn:              writeNDConfWithRuntime,
		},
		minioOps: dockerRuntimeMinioOps{
			CreateDefaultMcConfFn: config.CreateDefaultMcConf,
			SetMinIOPasswordFn:    config.SetMinIOPassword,
			GetMinIOPasswordFn:    config.GetMinIOPassword,
		},
		commandOps: dockerRuntimeCommandOps{
			ExecDockerCommandFn: func(name string, cmd []string) (string, error) {
				out, _, err := ExecDockerCommand(name, cmd)
				return out, err
			},
			ExecDockerCommandExitFn: ExecDockerCommand,
			RandReadFn:              rand.Read,
			CopyFileToVolumeFn:      copyFileToVolumeWithTempContainer,
		},
		volumeOps: dockerRuntimeVolumeOps{
			VolumeExistsFn: func(volumeName string) (bool, error) {
				return network.NewNetworkRuntime().VolumeExists(volumeName)
			},
			CreateVolumeFn: func(volumeName string) error {
				return network.NewNetworkRuntime().CreateVolume(volumeName)
			},
		},
		timerOps: dockerRuntimeTimerOps{
			SleepFn:        time.Sleep,
			PollIntervalFn: runtimePollInterval,
		},
	}
}

func runtimePollInterval() time.Duration {
	return 500 * time.Millisecond
}
