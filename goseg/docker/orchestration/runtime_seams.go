package orchestration

import (
	"crypto/rand"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/orchestration/container"
	"groundseg/startram"
	"groundseg/structs"
	"os"
	"reflect"
	"time"
)

type Runtime struct {
	RuntimeContainerOps
	RuntimeUrbitOps
	RuntimeSnapshotOps
	RuntimeConfigOps
	RuntimeLoadOps
	RuntimeServiceOps
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
		assignNonNilCallbacks(&runtime.RuntimeContainerOps, ops)
	}
}

func WithUrbitOps(ops RuntimeUrbitOps) RuntimeOption {
	return func(runtime *Runtime) {
		assignNonNilCallbacks(&runtime.RuntimeUrbitOps, ops)
	}
}

func WithSnapshotOps(ops RuntimeSnapshotOps) RuntimeOption {
	return func(runtime *Runtime) {
		assignNonNilCallbacks(&runtime.RuntimeSnapshotOps, ops)
	}
}

func WithConfigOps(ops RuntimeConfigOps) RuntimeOption {
	return func(runtime *Runtime) {
		assignNonNilCallbacks(&runtime.RuntimeConfigOps, ops)
	}
}

func WithLoadOps(ops RuntimeLoadOps) RuntimeOption {
	return func(runtime *Runtime) {
		assignNonNilCallbacks(&runtime.RuntimeLoadOps, ops)
	}
}

func WithServiceOps(ops RuntimeServiceOps) RuntimeOption {
	return func(runtime *Runtime) {
		assignNonNilCallbacks(&runtime.RuntimeServiceOps, ops)
	}
}

func NewRuntime(opts ...RuntimeOption) Runtime {
	runtime := Runtime{
		RuntimeContainerOps: defaultRuntimeContainerOps(),
		RuntimeUrbitOps:     defaultRuntimeUrbit(),
		RuntimeSnapshotOps:  defaultRuntimeSnapshot(),
		RuntimeConfigOps:    defaultRuntimeConfig(),
		RuntimeLoadOps:      defaultRuntimeLoad(),
		RuntimeServiceOps:   defaultRuntimeService(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&runtime)
		}
	}
	return runtime
}

type StartupRuntimeOption func(*StartupRuntime)

func WithStartupBootstrapOps(ops StartupBootstrapOps) StartupRuntimeOption {
	return func(runtime *StartupRuntime) {
		assignNonNilCallbacks(&runtime.StartupBootstrapOps, ops)
	}
}

func WithStartupImageOps(ops StartupImageOps) StartupRuntimeOption {
	return func(runtime *StartupRuntime) {
		assignNonNilCallbacks(&runtime.StartupImageOps, ops)
	}
}

func WithStartupLoadOps(ops StartupLoadOps) StartupRuntimeOption {
	return func(runtime *StartupRuntime) {
		assignNonNilCallbacks(&runtime.StartupLoadOps, ops)
	}
}

func assignNonNilCallbacks[T any](target *T, source T) {
	targetVal := reflect.ValueOf(target).Elem()
	sourceVal := reflect.ValueOf(source)
	for i := 0; i < targetVal.NumField(); i++ {
		sourceField := sourceVal.Field(i)
		if sourceField.Kind() != reflect.Func || sourceField.IsNil() {
			continue
		}
		targetVal.Field(i).Set(sourceField)
	}
}

func NewStartupRuntime(opts ...StartupRuntimeOption) StartupRuntime {
	runtime := StartupRuntime{
		StartupBootstrapOps: defaultStartupBootstrap(),
		StartupImageOps:     defaultStartupImage(),
		StartupLoadOps:      defaultStartupLoad(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&runtime)
		}
	}
	return runtime
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
		LoadUrbitConfigFn:     config.LoadUrbitConfig,
		UrbitConfFn:           config.UrbitConf,
		UpdateUrbitFn:         config.UpdateUrbit,
		GetContainerNetworkFn: GetContainerNetwork,
		GetLusCodeFn:          click.GetLusCode,
		ClearLusCodeFn:        click.ClearLusCode,
	}
}

func defaultRuntimeSnapshot() RuntimeSnapshotOps {
	return RuntimeSnapshotOps{
		StartramSettingsSnapshotFn: config.StartramSettingsSnapshot,
		ShipSettingsSnapshotFn:     config.ShipSettingsSnapshot,
		GetStartramConfigFn:        config.GetStartramConfig,
		Check502SettingsSnapshotFn: config.Check502SettingsSnapshot,
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
		LoadWireguardFn: LoadWireguard,
		LoadMCFn:        LoadMC,
		LoadMinIOsFn:    LoadMinIOs,
		LoadUrbitsFn:    LoadUrbits,
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
		LoadWireguard: LoadWireguard,
		LoadMC:        LoadMC,
		LoadMinIOs:    LoadMinIOs,
		LoadNetdata:   LoadNetdata,
		LoadUrbits:    LoadUrbits,
		LoadLlama:     LoadLlama,
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
			BasePathFn:     func() string { return config.RuntimeContextSnapshot().BasePath },
			ArchitectureFn: func() string { return config.RuntimeContextSnapshot().Architecture },
			DebugModeFn:    func() bool { return config.RuntimeContextSnapshot().DebugMode },
			DockerDirFn:    func() string { return config.RuntimeContextSnapshot().DockerDir },
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
			AddOrGetNetworkFn:           AddOrGetNetwork,
		},
		imageOps: dockerRuntimeImageOps{
			GetLatestContainerInfoFn:  GetLatestContainerInfo,
			GetLatestContainerImageFn: latestContainerImage,
		},
		configOps: dockerRuntimeConfigOps{
			ConfFn:                    config.Conf,
			ShipSettingsSnapshotFn:    config.ShipSettingsSnapshot,
			RuntimeSettingsSnapshotFn: config.ShipRuntimeSettingsSnapshot,
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
			WriteWgConfFn:  writeWgConfWithRuntime,
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
			VolumeExistsFn: VolumeExists,
			CreateVolumeFn: CreateVolume,
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
