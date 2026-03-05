package orchestration

import (
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/network"
	"groundseg/startram"
)

func defaultRuntimeWireguardOps() RuntimeWireguardOps {
	return RuntimeWireguardOps{
		CreateDefaultWGConfFn: config.CreateDefaultWGConf,
		GetWgConfFn:           config.GetWgConf,
		GetWgConfBlobFn:       getConfiguredStartramWGConfig,
		GetWgPrivkeyFn:        config.GetWgPrivkey,
		CopyFileToVolumeFn:    copyFileToVolumeWithTempContainer,
	}
}

func defaultRuntimeStartramOps() RuntimeStartramOps {
	return RuntimeStartramOps{
		GetStartramServicesFn: func() error {
			return errStartramServicesLoaderMissing
		},
		LoadStartramRegionsFn: func() error {
			return errStartramRegionsLoaderMissing
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
		UpdateConfigTypedFn:    config.UpdateConfigTyped,
		WithWireguardEnabledFn: config.WithWgOn,
		CycleWgKeyFn:           config.CycleWgKey,
		BarExitFn:              click.BarExit,
		LoadWireguardFn:        wireguardRuntime.LoadWireguard,
		LoadMCFn:               LoadMC,
		LoadMinIOsFn:           LoadMinIOs,
		LoadUrbitsFn:           LoadUrbits,
		SvcDeleteFn:            startram.SvcDelete,
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
		LoadUrbitConfigFn:     config.LoadUrbitConfig,
		UrbitConfFn:           config.UrbitConf,
		UrbitConfAllFn:        config.UrbitConfAll,
		UpdateUrbitFn:         config.UpdateUrbit,
		UpdateUrbitSectionFn:  config.UpdateUrbitSection,
		GetContainerNetworkFn: networkRuntime.GetContainerNetwork,
		GetLusCodeFn:          click.GetLusCode,
		ClearLusCodeFn:        click.ClearLusCode,
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
		InitializeFn: Initialize,
	}
}

func defaultStartupImage() StartupImageOps {
	return StartupImageOps{
		GetLatestContainerInfoFn: GetLatestContainerInfo,
		PullImageIfNotExistFn:    PullImageIfNotExist,
	}
}

func defaultStartupLoad() StartupLoadOps {
	loadWireguard := newWireguardRuntime().LoadWireguard
	return StartupLoadOps{
		LoadWireguardFn: loadWireguard,
		LoadMCFn:        LoadMC,
		LoadMinIOsFn:    LoadMinIOs,
		LoadNetdataFn:   LoadNetdata,
		LoadUrbitsFn:    LoadUrbits,
		LoadLlamaFn:     LoadLlama,
	}
}
