package collectors

import (
	"fmt"
	"groundseg/backupsvc"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/network"
	"groundseg/docker/orchestration"
	"groundseg/internal/seams"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/system"
	systemdisk "groundseg/system/disk"
	maintenanceapt "groundseg/system/maintenance/apt"
	"time"
)

var defaultCollectorRuntimeValue = NewCollectorRuntime()

type collectorRuntime struct {
	collectorUrbitRuntime
	collectorConfigRuntime
	collectorSystemRuntime
}

type collectorUrbitRuntime struct {
	LoadUrbitConfigFn        func(string) error
	UrbitConfFn              func(string) structs.UrbitDocker
	GetContainerStatsFn      func(string) structs.ContainerStats
	GetContainerImageTagFn   func(string) (string, error)
	GetMinIOLinkedStatusFn   func(string) bool
	GetMinIOPasswordFn       func(string) (string, error)
	GetContainerNetworkFn    func(string) (string, error)
	GetContainerShipStatusFn func([]string) (map[string]string, error)
	LusCodeFn                func(string) (string, error)
	GetDeskFn                func(string, string, bool) (string, error)
}

type collectorConfigRuntime struct {
	StartramSettingsFn          func() config.StartramSettings
	StartramConfigFn            func() structs.StartramRetrieve
	StartramServicesRetrieverFn func() (structs.StartramRetrieve, error)
	PenpaiSettingsFn            func() config.PenpaiSettings
	SwapSettingsFn              func() config.SwapSettings
	BackupRootFn                func() string
	BackupTimeFn                func() time.Time
}

type collectorSystemRuntime struct {
	SystemUpdatesFn        func() structs.SystemUpdates
	WiFiInfoSnapshotFn     func() structs.SystemWifi
	GetMemoryFn            func() (uint64, uint64, error)
	GetCPUFn               func() (int, error)
	GetTempFn              func() (float64, error)
	GetDiskFn              func() (map[string][2]uint64, error)
	ListHardDisksFn        func() (structs.LSBLKDevice, error)
	IsDevMountedFn         func(structs.BlockDev) bool
	SmartResultsSnapshotFn func() map[string]bool
}

func NewCollectorRuntime() collectorRuntime {
	networkRuntime := network.NewNetworkRuntime()
	return collectorRuntime{
		collectorUrbitRuntime: collectorUrbitRuntime{
			LoadUrbitConfigFn:        config.LoadUrbitConfig,
			UrbitConfFn:              config.UrbitConf,
			GetContainerStatsFn:      orchestration.GetContainerStats,
			GetContainerImageTagFn:   orchestration.GetContainerImageTag,
			GetMinIOLinkedStatusFn:   config.GetMinIOLinkedStatus,
			GetMinIOPasswordFn:       config.GetMinIOPassword,
			GetContainerNetworkFn:    networkRuntime.GetContainerNetwork,
			GetContainerShipStatusFn: orchestration.GetShipStatus,
			LusCodeFn:                click.GetLusCode,
			GetDeskFn:                click.GetDesk,
		},
		collectorConfigRuntime: collectorConfigRuntime{
			StartramSettingsFn:          config.StartramSettingsSnapshot,
			StartramConfigFn:            config.GetStartramConfig,
			StartramServicesRetrieverFn: startram.Retrieve,
			PenpaiSettingsFn:            config.PenpaiSettingsSnapshot,
			SwapSettingsFn:              config.SwapSettingsSnapshot,
			BackupRootFn:                func() string { return backupsvc.ResolveBackupRoot(config.BasePath()) },
			BackupTimeFn:                func() time.Time { return config.BackupTime },
		},
		collectorSystemRuntime: collectorSystemRuntime{
			SystemUpdatesFn:        maintenanceapt.SystemUpdatesSnapshot,
			WiFiInfoSnapshotFn:     func() structs.SystemWifi { return system.WifiInfoSnapshot() },
			GetMemoryFn:            system.GetMemory,
			GetCPUFn:               system.GetCPU,
			GetTempFn:              system.GetTemp,
			GetDiskFn:              system.GetDisk,
			ListHardDisksFn:        systemdisk.ListHardDisks,
			IsDevMountedFn:         systemdisk.IsDevMounted,
			SmartResultsSnapshotFn: systemdisk.SmartResultsSnapshot,
		},
	}
}

func defaultCollectorRuntime() collectorRuntime {
	return DefaultCollectorRuntime()
}

func DefaultCollectorRuntime() collectorRuntime {
	return defaultCollectorRuntimeValue
}

func collectorRuntimeWithDefaults(runtime collectorRuntime) collectorRuntime {
	defaultRuntime := NewCollectorRuntime()
	defaultRuntime.collectorUrbitRuntime = seams.Merge(defaultRuntime.collectorUrbitRuntime, runtime.collectorUrbitRuntime)
	defaultRuntime.collectorConfigRuntime = seams.Merge(defaultRuntime.collectorConfigRuntime, runtime.collectorConfigRuntime)
	defaultRuntime.collectorSystemRuntime = seams.Merge(defaultRuntime.collectorSystemRuntime, runtime.collectorSystemRuntime)
	return defaultRuntime
}

func collectorRuntimeOrDefault(runtime ...collectorRuntime) collectorRuntime {
	if len(runtime) > 0 {
		return collectorRuntimeWithDefaults(runtime[0])
	}
	return defaultCollectorRuntime()
}

// ConstructPierInfo builds the urbit entries for broadcast state.
func ConstructPierInfo(existingUrbits map[string]structs.Urbit, scheduled func(string) time.Time) (map[string]structs.Urbit, error) {
	return constructPierInfo(defaultCollectorRuntime(), existingUrbits, scheduled)
}

func constructPierInfo(runtime collectorRuntime, existingUrbits map[string]structs.Urbit, scheduled func(string) time.Time) (map[string]structs.Urbit, error) {
	runtime = collectorRuntimeWithDefaults(runtime)
	startramSettings := runtime.StartramSettingsFn()
	startramConfig := runtime.StartramConfigFn()
	settings := startramSettings
	piers := settings.Piers
	sgContext := wireguardContext{
		registered: settings.WgRegistered,
		on:         settings.WgOn,
	}

	backups := backupSnapshotForRuntime(runtime, piers, startramConfig.Backups)
	rtSnapshot, err := runtimeSnapshotForPiersWithRuntime(runtime, piers, existingUrbits)
	if err != nil {
		return nil, fmt.Errorf("constructing pier info: %w", err)
	}
	startramSnapshot := startramSnapshotForPiers(startramConfig.Subdomains)
	deploymentInputs := collectUrbitDeploymentInputsForPiers(
		piers,
		rtSnapshot.hostName,
		sgContext,
		startramSnapshot,
		runtime,
	)
	runtimeInputs := collectUrbitRuntimeInputsForPiers(
		rtSnapshot.pierStatus,
		urbitRuntimeContext{
			existingUrbits: rtSnapshot.currentState,
			shipNetworks:   rtSnapshot.shipNetworks,
		},
		scheduled,
		runtime,
	)
	return composeUrbitViews(piers, runtimeInputs, deploymentInputs, backups), nil
}

func (runtime collectorRuntime) backupRoot() (string, bool) {
	if runtime.BackupRootFn == nil {
		return "", false
	}
	return runtime.BackupRootFn(), true
}
