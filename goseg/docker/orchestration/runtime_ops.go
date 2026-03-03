package orchestration

import (
	"errors"
	"fmt"

	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/network"
	"groundseg/docker/orchestration/container"
	"groundseg/docker/registry"
	"groundseg/internal/seams"
	"groundseg/startram"
	"groundseg/structs"
	"os"
	"time"
)

type RuntimeTransitionOps struct {
	RuntimeContainerOps
	RuntimeUrbitOps
}


type RuntimeContainerOps struct {
	StartContainerFn      func(name string, ctype string) (structs.ContainerState, error)
	StopContainerByNameFn func(name string) error
	CreateContainerFn     func(name string, ctype string) (structs.ContainerState, error)
	RestartContainerFn    func(name string) error
	DeleteContainerFn     func(name string) error
	GetContainerStateFn    func() map[string]structs.ContainerState
	UpdateContainerStateFn func(name string, state structs.ContainerState)
	AddOrGetNetworkFn func(networkName string) (string, error)
	GetContainerRunningStatusFn func(name string) (string, error)
	GetShipStatusFn   func([]string) (map[string]string, error)
	WaitForShipExitFn func(name string, timeout time.Duration) error
}

type RuntimeSnapshotOps struct {
	ConfFn                        func() structs.SysConfig
	StartramSettingsSnapshotFn    func() config.StartramSettings
	ShipSettingsSnapshotFn        func() config.ShipSettings
	ShipRuntimeSettingsSnapshotFn func() config.ShipRuntimeSettings
	GetStartramConfigFn           func() structs.StartramRetrieve
	Check502SettingsSnapshotFn    func() config.Check502Settings
}

type RuntimeHealthOps = RuntimeSnapshotOps

type RuntimeStartramOps struct {
	GetStartramServicesFn   func() error
	LoadStartramRegionsFn   func() error
	DispatchUrbitPayloadFn  func(payload structs.WsUrbitPayload) error
	PublishEventFn          func(event structs.Event)
	RecoverWireguardFleetFn func(piers []string, deleteMinioClient bool) error
}

type RuntimeStartupOps struct {
	UpdateConfTypedFn func(...config.ConfUpdateOption) error
	WithWgOnFn        func(bool) config.ConfUpdateOption
	CycleWgKeyFn      func() error
	BarExitFn         func(string) error
	LoadWireguardFn   func() error
	LoadMCFn          func() error
	LoadMinIOsFn      func() error
	LoadUrbitsFn      func() error
	SvcDeleteFn       func(patp string, kind string) error
}

func (ops RuntimeSnapshotOps) StartramSettingsSnapshot() (config.StartramSettings, error) {
	if ops.StartramSettingsSnapshotFn == nil {
		return config.StartramSettings{}, errors.New("runtime startram settings snapshot callback is not configured")
	}
	return ops.StartramSettingsSnapshotFn(), nil
}

func (ops RuntimeSnapshotOps) StartramConfig() (structs.StartramRetrieve, error) {
	if ops.GetStartramConfigFn == nil {
		return structs.StartramRetrieve{}, errors.New("runtime startram config snapshot callback is not configured")
	}
	return ops.GetStartramConfigFn(), nil
}

func (ops RuntimeSnapshotOps) Check502SettingsSnapshot() (config.Check502Settings, error) {
	if ops.Check502SettingsSnapshotFn == nil {
		return config.Check502Settings{}, errors.New("runtime check 502 settings snapshot callback is not configured")
	}
	return ops.Check502SettingsSnapshotFn(), nil
}

func (ops RuntimeSnapshotOps) ShipSettingsSnapshot() (config.ShipSettings, error) {
	if ops.ShipSettingsSnapshotFn == nil {
		return config.ShipSettings{}, errors.New("runtime ship settings snapshot callback is not configured")
	}
	return ops.ShipSettingsSnapshotFn(), nil
}

type RuntimeUrbitOps struct {
	LoadUrbitConfigFn    func(string) error
	UrbitConfFn          func(string) structs.UrbitDocker
	UrbitConfAllFn       func() map[string]structs.UrbitDocker
	UpdateUrbitFn        func(string, func(*structs.UrbitDocker) error) error
	UpdateUrbitSectionFn func(patp string, section UrbitConfigSection, mutateFn any) error
	GetContainerNetworkFn func(string) (string, error)
	GetLusCodeFn          func(string) (string, error)
	ClearLusCodeFn        func(string)
}

type UrbitConfigSection = config.UrbitConfigSection

const (
	UrbitConfigSectionRuntime  = config.UrbitConfigSectionRuntime
	UrbitConfigSectionNetwork  = config.UrbitConfigSectionNetwork
	UrbitConfigSectionSchedule = config.UrbitConfigSectionSchedule
	UrbitConfigSectionFeature  = config.UrbitConfigSectionFeature
	UrbitConfigSectionWeb      = config.UrbitConfigSectionWeb
	UrbitConfigSectionBackup   = config.UrbitConfigSectionBackup
)

type RuntimeContextOps struct {
	BasePathFn     func() string
	ArchitectureFn func() string
	DebugModeFn    func() bool
	DockerDirFn    func() string
}

type RuntimeFileOps = container.RuntimeFileOps

type RuntimeImageOps struct {
	GetLatestContainerInfoFn  func(string) (map[string]string, error)
	GetLatestContainerImageFn func(string) (string, error)
}

type RuntimeVolumeOps struct {
	VolumeExistsFn func(string) (bool, error)
	CreateVolumeFn func(string) error
}

type RuntimeCommandOps struct {
	ExecDockerCommandFn     func(string, []string) (string, error)
	ExecDockerCommandExitFn func(string, []string) (string, int, error)
	RandReadFn              func([]byte) (int, error)
	CopyFileToVolumeFn      func(string, string, string, string, volumeWriterImageSelector) error
}

type RuntimeTimerOps struct {
	SleepFn        func(time.Duration)
	PollIntervalFn func() time.Duration
}

type RuntimeWireguardOps struct {
	CreateDefaultWGConfFn func() error
	GetWgConfFn           func() (structs.WgConfig, error)
	GetWgConfBlobFn       func() (string, error)
	GetWgPrivkeyFn        func() string
	CopyFileToVolumeFn    func(string, string, string, string, volumeWriterImageSelector) error
	WriteWgConfFn         func() error
}

type RuntimeNetdataOps struct {
	CreateDefaultNetdataConfFn func() error
	WriteNDConfFn              func() error
}

type RuntimeMinioOps struct {
	CreateDefaultMcConfFn func() error
	SetMinIOPasswordFn    func(string, string) error
	GetMinIOPasswordFn    func(string) (string, error)
}

type RuntimeConfigOps struct {
	UpdateConfTypedFn func(...config.ConfUpdateOption) error
	WithWgOnFn        func(bool) config.ConfUpdateOption
	CycleWgKeyFn      func() error
	BarExitFn         func(string) error
}

var (
	errConfUpdateMissing = errors.New("orchestration config updater is not configured")
)

func (ops RuntimeConfigOps) UpdateConfig(opts ...config.ConfUpdateOption) error {
	if ops.UpdateConfTypedFn == nil {
		return errConfUpdateMissing
	}
	return ops.UpdateConfTypedFn(opts...)
}

func (ops RuntimeConfigOps) WithWgOn(enabled bool) config.ConfUpdateOption {
	if ops.WithWgOnFn == nil {
		return config.WithWgOn(enabled)
	}
	return ops.WithWgOnFn(enabled)
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

type runtimeSeamRegistry struct {
	contextOps   RuntimeContextOps
	fileOps      RuntimeFileOps
	imageOps     RuntimeImageOps
	snapshotOps  RuntimeSnapshotOps
	urbitOps     RuntimeUrbitOps
	wireguardOps RuntimeWireguardOps
	netdataOps   RuntimeNetdataOps
	minioOps     RuntimeMinioOps
	volumeOps    RuntimeVolumeOps
}

func runtimeSeams() runtimeSeamRegistry {
	return buildRuntimeSeamBundle()
}

func buildRuntimeSeamBundle() runtimeSeamRegistry {
	networkRuntime := network.NewNetworkRuntime()
	return runtimeSeamRegistry{
		contextOps: RuntimeContextOps{
			BasePathFn:     config.BasePath,
			ArchitectureFn: config.Architecture,
			DebugModeFn:    config.DebugMode,
			DockerDirFn:    config.DockerDir,
		},
		fileOps: RuntimeFileOps{
			OpenFn:      os.Open,
			ReadFileFn:  os.ReadFile,
			WriteFileFn: os.WriteFile,
			MkdirAllFn:  os.MkdirAll,
		},
		imageOps: RuntimeImageOps{
			GetLatestContainerInfoFn:  registry.GetLatestContainerInfo,
			GetLatestContainerImageFn: latestContainerImage,
		},
		snapshotOps: defaultRuntimeSnapshot(),
		urbitOps:    defaultRuntimeUrbit(),
		wireguardOps: RuntimeWireguardOps{
			CreateDefaultWGConfFn: config.CreateDefaultWGConf,
			GetWgConfFn:           config.GetWgConf,
			GetWgConfBlobFn:       getConfiguredStartramWGConfig,
			GetWgPrivkeyFn:        config.GetWgPrivkey,
			CopyFileToVolumeFn:    copyFileToVolumeWithTempContainer,
		},
		netdataOps: RuntimeNetdataOps{
			CreateDefaultNetdataConfFn: config.CreateDefaultNetdataConf,
		},
		minioOps: RuntimeMinioOps{
			CreateDefaultMcConfFn: config.CreateDefaultMcConf,
			SetMinIOPasswordFn:    config.SetMinIOPassword,
			GetMinIOPasswordFn:    config.GetMinIOPassword,
		},
		volumeOps: RuntimeVolumeOps{
			VolumeExistsFn: networkRuntime.VolumeExists,
			CreateVolumeFn: networkRuntime.CreateVolume,
		},
	}
}

func defaultRuntimeTransitionOps() RuntimeTransitionOps {
	return RuntimeTransitionOpsFromContainerOps(defaultRuntimeContainerOpsWithLifecycle(), RuntimeTransitionOpsFromUrbitOps(defaultRuntimeUrbit()))
}

func urbitConfigOps() RuntimeUrbitOps {
	return RuntimeUrbitOps{
		LoadUrbitConfigFn:    config.LoadUrbitConfig,
		UrbitConfFn:          config.UrbitConf,
		UrbitConfAllFn:       config.UrbitConfAll,
		UpdateUrbitFn:        config.UpdateUrbit,
		UpdateUrbitSectionFn: config.UpdateUrbitSectionConfig,
	}
}

func RuntimeTransitionOpsFromContainerOps(containerOps RuntimeContainerOps, overrides ...RuntimeTransitionOps) RuntimeTransitionOps {
	return seams.MergeAll(RuntimeTransitionOps{
		RuntimeContainerOps: containerOps,
	}, overrides...)
}

func RuntimeTransitionOpsFromUrbitOps(urbitOps RuntimeUrbitOps, overrides ...RuntimeTransitionOps) RuntimeTransitionOps {
	return seams.MergeAll(RuntimeTransitionOps{
		RuntimeUrbitOps: urbitOps,
	}, overrides...)
}

func defaultRuntimeStartramOps() RuntimeStartramOps {
	return RuntimeStartramOps{
		GetStartramServicesFn: func() error {
			return fmt.Errorf("orchestration startram service loader is not configured")
		},
		LoadStartramRegionsFn: func() error {
			return fmt.Errorf("orchestration startram region loader is not configured")
		},
	}
}

func defaultRuntimeHealthOps() RuntimeHealthOps {
	return RuntimeHealthOps(defaultRuntimeSnapshot())
}

func defaultRuntimeStartupOps() RuntimeStartupOps {
	configOps := defaultRuntimeConfig()
	loadOps := defaultRuntimeLoad()
	serviceOps := defaultRuntimeService()
	return RuntimeStartupOps{
		UpdateConfTypedFn: configOps.UpdateConfTypedFn,
		WithWgOnFn:        configOps.WithWgOnFn,
		CycleWgKeyFn:      configOps.CycleWgKeyFn,
		BarExitFn:         configOps.BarExitFn,
		LoadWireguardFn:   loadOps.LoadWireguardFn,
		LoadMCFn:          loadOps.LoadMCFn,
		LoadMinIOsFn:      loadOps.LoadMinIOsFn,
		LoadUrbitsFn:      loadOps.LoadUrbitsFn,
		SvcDeleteFn:       serviceOps.SvcDeleteFn,
	}
}

func defaultRuntimeContainerOps() RuntimeContainerOps {
	networkRuntime := network.NewNetworkRuntime()
	return RuntimeContainerOps{
		StartContainerFn:             StartContainer,
		StopContainerByNameFn:        StopContainerByName,
		CreateContainerFn:            CreateContainer,
		RestartContainerFn:           RestartContainer,
		DeleteContainerFn:            DeleteContainer,
		GetContainerStateFn:          config.GetContainerState,
		UpdateContainerStateFn:       config.UpdateContainerState,
		AddOrGetNetworkFn:            networkRuntime.AddOrGetNetwork,
		GetContainerRunningStatusFn:  GetContainerRunningStatus,
		GetShipStatusFn:              GetShipStatus,
		WaitForShipExitFn:            WaitForShipExit,
	}
}

func defaultRuntimeContainerOpsWithLifecycle() RuntimeContainerOps {
	networkRuntime := network.NewNetworkRuntime()
	return RuntimeContainerOps{
		StartContainerFn:             StartContainer,
		StopContainerByNameFn:        StopContainerByName,
		CreateContainerFn:            CreateContainer,
		RestartContainerFn:           RestartContainer,
		DeleteContainerFn:            DeleteContainer,
		GetContainerStateFn:          config.GetContainerState,
		UpdateContainerStateFn:       config.UpdateContainerState,
		AddOrGetNetworkFn:            networkRuntime.AddOrGetNetwork,
		GetContainerRunningStatusFn:  GetContainerRunningStatus,
		GetShipStatusFn:              GetShipStatus,
		WaitForShipExitFn:            WaitForShipExit,
	}
}

func defaultRuntimeUrbit() RuntimeUrbitOps {
	networkRuntime := network.NewNetworkRuntime()
	return RuntimeUrbitOps{
		LoadUrbitConfigFn:    config.LoadUrbitConfig,
		UrbitConfFn:          config.UrbitConf,
		UrbitConfAllFn:       config.UrbitConfAll,
		UpdateUrbitFn:        config.UpdateUrbit,
		UpdateUrbitSectionFn: config.UpdateUrbitSectionConfig,
		GetContainerNetworkFn: networkRuntime.GetContainerNetwork,
		GetLusCodeFn:          click.GetLusCode,
		ClearLusCodeFn:        click.ClearLusCode,
	}
}

func defaultRuntimeSnapshot() RuntimeSnapshotOps {
	return RuntimeSnapshotOps{
		ConfFn:                        config.Conf,
		StartramSettingsSnapshotFn:    config.StartramSettingsSnapshot,
		ShipSettingsSnapshotFn:        config.ShipSettingsSnapshot,
		ShipRuntimeSettingsSnapshotFn: config.ShipRuntimeSettingsSnapshot,
		GetStartramConfigFn:           config.GetStartramConfig,
		Check502SettingsSnapshotFn:    config.Check502SettingsSnapshot,
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
		LoadWireguardFn: newWireguardRuntime().LoadWireguard,
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
		LoadWireguard: newWireguardRuntime().LoadWireguard,
		LoadMC:        LoadMC,
		LoadMinIOs:    LoadMinIOs,
		LoadNetdata:   LoadNetdata,
		LoadUrbits:    LoadUrbits,
		LoadLlama:     LoadLlama,
	}
}
