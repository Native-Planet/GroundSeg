package orchestration

import (
	"errors"
	"fmt"

	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/network"
	"groundseg/docker/registry"
	"groundseg/startram"
	"groundseg/structs"
	"os"
	"sync"
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
	StartContainerFn      func(name string, ctype string) (structs.ContainerState, error)
	StopContainerByNameFn func(name string) error
	CreateContainerFn     func(name string, ctype string) (structs.ContainerState, error)
	RestartContainerFn    func(name string) error
	DeleteContainerFn     func(name string) error
}

type RuntimeContainerStateOps struct {
	GetContainerStateFn    func() map[string]structs.ContainerState
	UpdateContainerStateFn func(name string, state structs.ContainerState)
}

type RuntimeContainerNetworkOps struct {
	AddOrGetNetworkFn func(networkName string) (string, error)
}

type RuntimeContainerObservationOps struct {
	GetContainerRunningStatusFn func(name string) (string, error)
}

type RuntimeContainerLifecycleStatusOps struct {
	GetShipStatusFn   func([]string) (map[string]string, error)
	WaitForShipExitFn func(name string, timeout time.Duration) error
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

type UrbitConfigSection = config.UrbitConfigSection

const (
	UrbitConfigSectionRuntime  = config.UrbitConfigSectionRuntime
	UrbitConfigSectionNetwork  = config.UrbitConfigSectionNetwork
	UrbitConfigSectionSchedule = config.UrbitConfigSectionSchedule
	UrbitConfigSectionFeature  = config.UrbitConfigSectionFeature
	UrbitConfigSectionWeb      = config.UrbitConfigSectionWeb
	UrbitConfigSectionBackup   = config.UrbitConfigSectionBackup
)

var (
	errUrbitConfLoadMissing             = errors.New("orchestration urbit config loader is not configured")
	errUrbitConfReadMissing             = errors.New("orchestration urbit config reader is not configured")
	errUrbitConfSnapshotWriterMissing   = errors.New("orchestration urbit config persistence callback is not configured")
	errUrbitRuntimeConfigWriterMissing  = errors.New("orchestration urbit runtime config persistence callback is not configured")
	errUrbitNetworkConfigWriterMissing  = errors.New("orchestration urbit network config persistence callback is not configured")
	errUrbitFeatureConfigWriterMissing  = errors.New("orchestration urbit feature config persistence callback is not configured")
	errUrbitWebConfigWriterMissing      = errors.New("orchestration urbit web config persistence callback is not configured")
	errUrbitBackupConfigWriterMissing   = errors.New("orchestration urbit backup config persistence callback is not configured")
	errUrbitScheduleConfigWriterMissing = errors.New("orchestration urbit schedule config persistence callback is not configured")
)

func (ops RuntimeUrbitConfigOps) LoadUrbitConfig(patp string) error {
	if ops.LoadUrbitConfigFn == nil {
		return fmt.Errorf("%w: %s", errUrbitConfLoadMissing, patp)
	}
	return ops.LoadUrbitConfigFn(patp)
}

func (ops RuntimeUrbitConfigOps) UrbitConf(patp string) (structs.UrbitDocker, error) {
	if ops.UrbitConfFn == nil {
		return structs.UrbitDocker{}, errUrbitConfReadMissing
	}
	return ops.UrbitConfFn(patp), nil
}

func (ops RuntimeUrbitConfigOps) LoadAndReadUrbitConfig(patp string) (structs.UrbitDocker, error) {
	if err := ops.LoadUrbitConfig(patp); err != nil {
		return structs.UrbitDocker{}, err
	}
	return ops.UrbitConf(patp)
}

func (ops RuntimeUrbitConfigOps) UpdateUrbitSnapshot(patp string, mutate func(*structs.UrbitDocker) error) error {
	if ops.UpdateUrbitFn == nil {
		return errUrbitConfSnapshotWriterMissing
	}
	return ops.UpdateUrbitFn(patp, mutate)
}

func (ops RuntimeUrbitConfigOps) UpdateUrbitRuntimeConfig(patp string, mutate func(*structs.UrbitRuntimeConfig) error) error {
	if mutate == nil {
		return fmt.Errorf("mutate function is required")
	}
	if ops.UpdateUrbitRuntimeConfigFn != nil {
		return ops.UpdateUrbitRuntimeConfigFn(patp, mutate)
	}
	if ops.UpdateUrbitFn == nil {
		return errUrbitRuntimeConfigWriterMissing
	}
	return ops.UpdateUrbitFn(patp, func(conf *structs.UrbitDocker) error {
		conf.UpdateRuntimeConfig(func(runtimeConf *structs.UrbitRuntimeConfig) {
			mutate(runtimeConf)
		})
		return nil
	})
}

func (ops RuntimeUrbitConfigOps) UpdateUrbitNetworkConfig(patp string, mutate func(*structs.UrbitNetworkConfig) error) error {
	if mutate == nil {
		return fmt.Errorf("mutate function is required")
	}
	if ops.UpdateUrbitNetworkConfigFn != nil {
		return ops.UpdateUrbitNetworkConfigFn(patp, mutate)
	}
	if ops.UpdateUrbitFn == nil {
		return errUrbitNetworkConfigWriterMissing
	}
	return ops.UpdateUrbitFn(patp, func(conf *structs.UrbitDocker) error {
		conf.UpdateNetworkConfig(func(networkConf *structs.UrbitNetworkConfig) {
			mutate(networkConf)
		})
		return nil
	})
}

func (ops RuntimeUrbitConfigOps) UpdateUrbitScheduleConfig(patp string, mutate func(*structs.UrbitScheduleConfig) error) error {
	if mutate == nil {
		return fmt.Errorf("mutate function is required")
	}
	if ops.UpdateUrbitScheduleConfigFn != nil {
		return ops.UpdateUrbitScheduleConfigFn(patp, mutate)
	}
	if ops.UpdateUrbitFn == nil {
		return errUrbitScheduleConfigWriterMissing
	}
	return ops.UpdateUrbitFn(patp, func(conf *structs.UrbitDocker) error {
		conf.UpdateScheduleConfig(func(scheduleConf *structs.UrbitScheduleConfig) {
			mutate(scheduleConf)
		})
		return nil
	})
}

func (ops RuntimeUrbitConfigOps) UpdateUrbitFeatureConfig(patp string, mutate func(*structs.UrbitFeatureConfig) error) error {
	if mutate == nil {
		return fmt.Errorf("mutate function is required")
	}
	if ops.UpdateUrbitFeatureConfigFn != nil {
		return ops.UpdateUrbitFeatureConfigFn(patp, mutate)
	}
	if ops.UpdateUrbitFn == nil {
		return errUrbitFeatureConfigWriterMissing
	}
	return ops.UpdateUrbitFn(patp, func(conf *structs.UrbitDocker) error {
		conf.UpdateFeatureConfig(func(featureConf *structs.UrbitFeatureConfig) {
			mutate(featureConf)
		})
		return nil
	})
}

func (ops RuntimeUrbitConfigOps) UpdateUrbitWebConfig(patp string, mutate func(*structs.UrbitWebConfig) error) error {
	if mutate == nil {
		return fmt.Errorf("mutate function is required")
	}
	if ops.UpdateUrbitWebConfigFn != nil {
		return ops.UpdateUrbitWebConfigFn(patp, mutate)
	}
	if ops.UpdateUrbitFn == nil {
		return errUrbitWebConfigWriterMissing
	}
	return ops.UpdateUrbitFn(patp, func(conf *structs.UrbitDocker) error {
		conf.UpdateWebConfig(func(webConf *structs.UrbitWebConfig) {
			mutate(webConf)
		})
		return nil
	})
}

func (ops RuntimeUrbitConfigOps) UpdateUrbitBackupConfig(patp string, mutate func(*structs.UrbitBackupConfig) error) error {
	if mutate == nil {
		return fmt.Errorf("mutate function is required")
	}
	if ops.UpdateUrbitBackupConfigFn != nil {
		return ops.UpdateUrbitBackupConfigFn(patp, mutate)
	}
	if ops.UpdateUrbitFn == nil {
		return errUrbitBackupConfigWriterMissing
	}
	return ops.UpdateUrbitFn(patp, func(conf *structs.UrbitDocker) error {
		conf.UpdateBackupConfig(func(backupConf *structs.UrbitBackupConfig) {
			mutate(backupConf)
		})
		return nil
	})
}

type RuntimeUrbitWorkflowOps struct {
	GetContainerNetworkFn func(string) (string, error)
	GetLusCodeFn          func(string) (string, error)
	ClearLusCodeFn        func(string)
}

type RuntimeUrbitOps struct {
	RuntimeUrbitConfigOps
	RuntimeUrbitWorkflowOps
}

type RuntimeSnapshotOps struct {
	ConfFn                        func() structs.SysConfig
	StartramSettingsSnapshotFn    func() config.StartramSettings
	ShipSettingsSnapshotFn        func() config.ShipSettings
	ShipRuntimeSettingsSnapshotFn func() config.ShipRuntimeSettings
	GetStartramConfigFn           func() structs.StartramRetrieve
	Check502SettingsSnapshotFn    func() config.Check502Settings
}

type RuntimeContextOps struct {
	BasePathFn     func() string
	ArchitectureFn func() string
	DebugModeFn    func() bool
	DockerDirFn    func() string
}

type RuntimeFileOps struct {
	OpenFn      func(string) (*os.File, error)
	ReadFileFn  func(string) ([]byte, error)
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

var (
	defaultRuntimeSeams runtimeSeamRegistry
	runtimeSeamsOnce    sync.Once
)

func runtimeSeams() runtimeSeamRegistry {
	runtimeSeamsOnce.Do(func() {
		defaultRuntimeSeams = buildRuntimeSeamBundle()
	})
	return defaultRuntimeSeams
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
	return RuntimeTransitionOps{
		RuntimeContainerOps: defaultRuntimeContainerOpsWithLifecycle(),
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
	}
}

func defaultRuntimeContainerOpsWithLifecycle() RuntimeContainerOps {
	ops := defaultRuntimeContainerOps()
	ops.RuntimeContainerLifecycleStatusOps = RuntimeContainerLifecycleStatusOps{
		GetShipStatusFn:   GetShipStatus,
		WaitForShipExitFn: WaitForShipExit,
	}
	return ops
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
			GetLusCodeFn:          click.GetLusCode,
			ClearLusCodeFn:        click.ClearLusCode,
		},
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
