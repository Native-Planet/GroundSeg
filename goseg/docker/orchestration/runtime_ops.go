package orchestration

import (
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/network"
	"groundseg/startram"
	"groundseg/structs"
	"os"
	"time"
)

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

type RuntimeContainerLifecycleOps struct {
	StartContainerFn            func(name string, ctype string) (structs.ContainerState, error)
	StopContainerByNameFn       func(name string) error
	CreateContainerFn           func(name string, ctype string) (structs.ContainerState, error)
	RestartContainerFn          func(name string) error
	DeleteContainerFn           func(name string) error
}

type RuntimeContainerStateOps struct {
	GetContainerStateFn         func() map[string]structs.ContainerState
	UpdateContainerStateFn      func(name string, state structs.ContainerState)
}

type RuntimeContainerNetworkOps struct {
	AddOrGetNetworkFn           func(networkName string) (string, error)
}

type RuntimeContainerObservationOps struct {
	GetContainerRunningStatusFn func(name string) (string, error)
}

type RuntimeContainerLifecycleStatusOps struct {
	GetShipStatusFn             func([]string) (map[string]string, error)
	WaitForShipExitFn           func(name string, timeout time.Duration) error
}

type RuntimeContainerOps struct {
	RuntimeContainerLifecycleOps
	RuntimeContainerStateOps
	RuntimeContainerNetworkOps
	RuntimeContainerObservationOps
	RuntimeContainerLifecycleStatusOps
}

type RuntimeUrbitConfigOps struct {
	LoadUrbitConfigFn           func(string) error
	UrbitConfFn                 func(string) structs.UrbitDocker
	UrbitConfAllFn              func() map[string]structs.UrbitDocker
	UpdateUrbitFn               func(string, func(*structs.UrbitDocker) error) error
	UpdateUrbitRuntimeConfigFn  func(string, func(*structs.UrbitRuntimeConfig) error) error
	UpdateUrbitNetworkConfigFn  func(string, func(*structs.UrbitNetworkConfig) error) error
	UpdateUrbitScheduleConfigFn func(string, func(*structs.UrbitScheduleConfig) error) error
	UpdateUrbitFeatureConfigFn  func(string, func(*structs.UrbitFeatureConfig) error) error
	UpdateUrbitWebConfigFn      func(string, func(*structs.UrbitWebConfig) error) error
 	UpdateUrbitBackupConfigFn   func(string, func(*structs.UrbitBackupConfig) error) error
}

type RuntimeUrbitWorkflowOps struct {
	GetContainerNetworkFn       func(string) (string, error)
	GetLusCodeFn                func(string) (string, error)
	ClearLusCodeFn              func(string)
}

type RuntimeUrbitOps struct {
	RuntimeUrbitConfigOps
	RuntimeUrbitWorkflowOps
}

type RuntimeSnapshotOps struct {
	ConfFn                    func() structs.SysConfig
	StartramSettingsSnapshotFn func() config.StartramSettings
	ShipSettingsSnapshotFn     func() config.ShipSettings
	ShipRuntimeSettingsSnapshotFn func() config.ShipRuntimeSettings
	GetStartramConfigFn        func() structs.StartramRetrieve
	Check502SettingsSnapshotFn func() config.Check502Settings
}

type RuntimeContextOps struct {
	BasePathFn     func() string
	ArchitectureFn func() string
	DebugModeFn    func() bool
	DockerDirFn    func() string
}

type RuntimeFileOps struct {
	OpenFn     func(string) (*os.File, error)
	ReadFileFn func(string) ([]byte, error)
	WriteFileFn func(string, []byte, os.FileMode) error
	MkdirAllFn  func(string, os.FileMode) error
}

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
	CopyFileToVolumeFn     func(string, string, string, string, volumeWriterImageSelector) error
	WriteWgConfFn          func() error
}

type RuntimeNetdataOps struct {
	CreateDefaultNetdataConfFn func() error
	WriteNDConfFn             func() error
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
	networkRuntime := network.NewNetworkRuntime()
	return RuntimeContainerOps{
		RuntimeContainerLifecycleOps: RuntimeContainerLifecycleOps{
			StartContainerFn:      StartContainer,
			StopContainerByNameFn: StopContainerByName,
			CreateContainerFn:     CreateContainer,
			RestartContainerFn:    RestartContainer,
			DeleteContainerFn:     DeleteContainer,
		},
		RuntimeContainerStateOps: RuntimeContainerStateOps{
			GetContainerStateFn:    config.GetContainerState,
			UpdateContainerStateFn: config.UpdateContainerState,
		},
		RuntimeContainerNetworkOps: RuntimeContainerNetworkOps{
			AddOrGetNetworkFn: networkRuntime.AddOrGetNetwork,
		},
		RuntimeContainerObservationOps: RuntimeContainerObservationOps{
			GetContainerRunningStatusFn: GetContainerRunningStatus,
		},
		RuntimeContainerLifecycleStatusOps: RuntimeContainerLifecycleStatusOps{
			GetShipStatusFn:   GetShipStatus,
			WaitForShipExitFn: WaitForShipExit,
		},
	}
}

func defaultRuntimeUrbit() RuntimeUrbitOps {
	networkRuntime := network.NewNetworkRuntime()
	return RuntimeUrbitOps{
		RuntimeUrbitConfigOps: RuntimeUrbitConfigOps{
			LoadUrbitConfigFn:           config.LoadUrbitConfig,
			UrbitConfFn:                 config.UrbitConf,
			UrbitConfAllFn:              config.UrbitConfAll,
			UpdateUrbitFn:               config.UpdateUrbit,
			UpdateUrbitRuntimeConfigFn:  config.UpdateUrbitRuntimeConfig,
			UpdateUrbitNetworkConfigFn:  config.UpdateUrbitNetworkConfig,
			UpdateUrbitScheduleConfigFn: config.UpdateUrbitScheduleConfig,
			UpdateUrbitFeatureConfigFn:  config.UpdateUrbitFeatureConfig,
			UpdateUrbitWebConfigFn:      config.UpdateUrbitWebConfig,
			UpdateUrbitBackupConfigFn:   config.UpdateUrbitBackupConfig,
		},
		RuntimeUrbitWorkflowOps: RuntimeUrbitWorkflowOps{
			GetContainerNetworkFn: networkRuntime.GetContainerNetwork,
			GetLusCodeFn:         click.GetLusCode,
			ClearLusCodeFn:       click.ClearLusCode,
		},
	}
}

func defaultRuntimeSnapshot() RuntimeSnapshotOps {
	return RuntimeSnapshotOps{
		ConfFn:                    config.Conf,
		StartramSettingsSnapshotFn: config.StartramSettingsSnapshot,
		ShipSettingsSnapshotFn:     config.ShipSettingsSnapshot,
		ShipRuntimeSettingsSnapshotFn: config.ShipRuntimeSettingsSnapshot,
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
