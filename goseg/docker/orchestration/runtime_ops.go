package orchestration

import (
	"errors"
	"fmt"

	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/network"
	"groundseg/docker/orchestration/container"
	"groundseg/docker/registry"
	"groundseg/startram"
	"groundseg/structs"
	"os"
	"time"
)

type RuntimeContainerOps struct {
	StartContainerFn            func(name string, ctype string) (structs.ContainerState, error) `runtime:"importer-provision,workflow,rectify" runtime_name:"start container callback"`
	StopContainerByNameFn       func(name string) error                                         `runtime:"workflow" runtime_name:"stop container callback"`
	CreateContainerFn           func(name string, ctype string) (structs.ContainerState, error)
	RestartContainerFn          func(name string) error
	DeleteContainerFn           func(name string) error                         `runtime:"importer-provision,workflow" runtime_name:"delete container callback"`
	GetContainerStateFn         func() map[string]structs.ContainerState        `runtime:"rectify" runtime_name:"get container state callback"`
	UpdateContainerStateFn      func(name string, state structs.ContainerState) `runtime:"importer-provision" runtime_name:"update container callback"`
	AddOrGetNetworkFn           func(networkName string) (string, error)
	GetContainerRunningStatusFn func(name string) (string, error)
	GetShipStatusFn             func([]string) (map[string]string, error) `runtime:"workflow" runtime_name:"get ship status callback"`
	WaitForShipExitFn           func(name string, timeout time.Duration) error
}

type RuntimeSnapshotOps struct {
	StartramSettingsSnapshotFn    func() config.StartramSettings
	PenpaiSettingsSnapshotFn      func() config.PenpaiSettings
	ShipSettingsSnapshotFn        func() config.ShipSettings        `runtime:"workflow,rectify" runtime_name:"ship settings snapshot callback"`
	ShipRuntimeSettingsSnapshotFn func() config.ShipRuntimeSettings `runtime:"workflow,rectify" runtime_name:"ship runtime settings snapshot callback"`
	GetStartramConfigFn           func() structs.StartramRetrieve   `runtime:"workflow,rectify" runtime_name:"startram config callback"`
	Check502SettingsSnapshotFn    func() config.Check502Settings    `runtime:"workflow,rectify" runtime_name:"check 502 snapshot callback"`
}

type RuntimeHealthOps struct {
	HealthCheck502SettingsSnapshotFn func() config.Check502Settings
	HealthShipSettingsSnapshotFn     func() config.ShipSettings
}

type RuntimeStartramOps struct {
	GetStartramServicesFn   func() error
	LoadStartramRegionsFn   func() error
	DispatchUrbitPayloadFn  func(payload structs.WsUrbitPayload) error
	PublishEventFn          func(event structs.Event)
	RecoverWireguardFleetFn func(piers []string, deleteMinioClient bool) error
}

type RuntimeStartupOps struct {
	UpdateConfTypedFn func(...config.ConfUpdateOption) error `runtime:"rectify" runtime_name:"update config callback"`
	WithWgOnFn        func(bool) config.ConfUpdateOption
	CycleWgKeyFn      func() error
	BarExitFn         func(string) error
	LoadWireguardFn   func() error
	LoadMCFn          func() error
	LoadMinIOsFn      func() error
	LoadUrbitsFn      func() error
	SvcDeleteFn       func(patp string, kind string) error
}

type RuntimeUrbitOps struct {
	LoadUrbitConfigFn           func(string) error                                                         `runtime:"rectify" runtime_name:"load urbit config callback"`
	UrbitConfFn                 func(string) structs.UrbitDocker                                           `runtime:"rectify" runtime_name:"urbit config callback"`
	UrbitConfAllFn              func() map[string]structs.UrbitDocker                                      `runtime:"rectify" runtime_name:"urbit config all callback"`
	UpdateUrbitFn               func(string, func(*structs.UrbitDocker) error) error                       `runtime:"workflow,rectify" runtime_name:"update urbit callback"`
	UpdateUrbitRuntimeConfigFn  func(patp string, mutateFn func(*structs.UrbitRuntimeConfig) error) error  `runtime:"workflow,rectify" runtime_name:"update urbit runtime config callback"`
	UpdateUrbitNetworkConfigFn  func(patp string, mutateFn func(*structs.UrbitNetworkConfig) error) error  `runtime:"workflow,rectify" runtime_name:"update urbit network config callback"`
	UpdateUrbitScheduleConfigFn func(patp string, mutateFn func(*structs.UrbitScheduleConfig) error) error `runtime:"workflow,rectify" runtime_name:"update urbit schedule config callback"`
	UpdateUrbitFeatureConfigFn  func(patp string, mutateFn func(*structs.UrbitFeatureConfig) error) error  `runtime:"workflow,rectify" runtime_name:"update urbit feature config callback"`
	UpdateUrbitWebConfigFn      func(patp string, mutateFn func(*structs.UrbitWebConfig) error) error      `runtime:"workflow,rectify" runtime_name:"update urbit web config callback"`
	UpdateUrbitBackupConfigFn   func(patp string, mutateFn func(*structs.UrbitBackupConfig) error) error   `runtime:"workflow,rectify" runtime_name:"update urbit backup config callback"`
	GetContainerNetworkFn       func(string) (string, error)
	GetLusCodeFn                func(string) (string, error) `runtime:"workflow" runtime_name:"LUS code callback"`
	ClearLusCodeFn              func(string)
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

var errConfUpdateMissing = errors.New("orchestration config updater is not configured")

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
		contextOps:   runtimeContextOps(),
		fileOps:      runtimeFileOps(),
		imageOps:     runtimeImageOps(),
		snapshotOps:  defaultRuntimeSnapshot(),
		urbitOps:     defaultRuntimeUrbit(),
		wireguardOps: runtimeWireguardOps(),
		netdataOps:   runtimeNetdataOps(),
		minioOps:     runtimeMinioOps(),
		volumeOps:    runtimeVolumeOps(networkRuntime),
	}
}

func runtimeContextOps() RuntimeContextOps {
	return RuntimeContextOps{
		BasePathFn:     config.BasePath,
		ArchitectureFn: config.Architecture,
		DebugModeFn:    config.DebugMode,
		DockerDirFn:    config.DockerDir,
	}
}

func runtimeFileOps() RuntimeFileOps {
	return RuntimeFileOps{
		OpenFn:      os.Open,
		ReadFileFn:  os.ReadFile,
		WriteFileFn: os.WriteFile,
		MkdirAllFn:  os.MkdirAll,
	}
}

func runtimeImageOps() RuntimeImageOps {
	return RuntimeImageOps{
		GetLatestContainerInfoFn:  registry.GetLatestContainerInfo,
		GetLatestContainerImageFn: latestContainerImage,
	}
}

func runtimeWireguardOps() RuntimeWireguardOps {
	return RuntimeWireguardOps{
		CreateDefaultWGConfFn: config.CreateDefaultWGConf,
		GetWgConfFn:           config.GetWgConf,
		GetWgConfBlobFn:       getConfiguredStartramWGConfig,
		GetWgPrivkeyFn:        config.GetWgPrivkey,
		CopyFileToVolumeFn:    copyFileToVolumeWithTempContainer,
	}
}

func runtimeNetdataOps() RuntimeNetdataOps {
	return RuntimeNetdataOps{
		CreateDefaultNetdataConfFn: config.CreateDefaultNetdataConf,
	}
}

func runtimeMinioOps() RuntimeMinioOps {
	return RuntimeMinioOps{
		CreateDefaultMcConfFn: config.CreateDefaultMcConf,
		SetMinIOPasswordFn:    config.SetMinIOPassword,
		GetMinIOPasswordFn:    config.GetMinIOPassword,
	}
}

func runtimeVolumeOps(networkRuntime interface {
	VolumeExists(string) (bool, error)
	CreateVolume(string) error
}) RuntimeVolumeOps {
	return RuntimeVolumeOps{
		VolumeExistsFn: networkRuntime.VolumeExists,
		CreateVolumeFn: networkRuntime.CreateVolume,
	}
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
	return RuntimeHealthOps{
		HealthShipSettingsSnapshotFn:     config.ShipSettingsSnapshot,
		HealthCheck502SettingsSnapshotFn: config.Check502SettingsSnapshot,
	}
}

func defaultRuntimeStartupOps() RuntimeStartupOps {
	wireguardRuntime := newWireguardRuntime()
	return RuntimeStartupOps{
		UpdateConfTypedFn: config.UpdateConfTyped,
		WithWgOnFn:        config.WithWgOn,
		CycleWgKeyFn:      config.CycleWgKey,
		BarExitFn:         click.BarExit,
		LoadWireguardFn:   wireguardRuntime.LoadWireguard,
		LoadMCFn:          LoadMC,
		LoadMinIOsFn:      LoadMinIOs,
		LoadUrbitsFn:      LoadUrbits,
		SvcDeleteFn:       startram.SvcDelete,
	}
}

func defaultRuntimeContainerOps() RuntimeContainerOps {
	networkRuntime := network.NewNetworkRuntime()
	return RuntimeContainerOps{
		StartContainerFn:            StartContainer,
		StopContainerByNameFn:       StopContainerByName,
		CreateContainerFn:           CreateContainer,
		RestartContainerFn:          RestartContainer,
		DeleteContainerFn:           DeleteContainer,
		GetContainerStateFn:         config.GetContainerState,
		UpdateContainerStateFn:      config.UpdateContainerState,
		AddOrGetNetworkFn:           networkRuntime.AddOrGetNetwork,
		GetContainerRunningStatusFn: GetContainerRunningStatus,
		GetShipStatusFn:             GetShipStatus,
		WaitForShipExitFn:           WaitForShipExit,
	}
}

func defaultRuntimeUrbit() RuntimeUrbitOps {
	networkRuntime := network.NewNetworkRuntime()
	return RuntimeUrbitOps{
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
		GetContainerNetworkFn:       networkRuntime.GetContainerNetwork,
		GetLusCodeFn:                click.GetLusCode,
		ClearLusCodeFn:              click.ClearLusCode,
	}
}

func defaultRuntimeSnapshot() RuntimeSnapshotOps {
	return RuntimeSnapshotOps{
		StartramSettingsSnapshotFn:    config.StartramSettingsSnapshot,
		PenpaiSettingsSnapshotFn:      config.PenpaiSettingsSnapshot,
		ShipSettingsSnapshotFn:        config.ShipSettingsSnapshot,
		ShipRuntimeSettingsSnapshotFn: config.ShipRuntimeSettingsSnapshot,
		GetStartramConfigFn:           config.GetStartramConfig,
		Check502SettingsSnapshotFn:    config.Check502SettingsSnapshot,
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
