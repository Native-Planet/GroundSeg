package orchestration

import (
	"crypto/rand"
	"fmt"
	"groundseg/config"
	"groundseg/docker/network"
	"groundseg/docker/registry"
	"os"
	"time"
)

type dockerRuntime struct {
	contextOps   RuntimeContextOps
	fileOps      RuntimeFileOps
	containerOps RuntimeContainerOps
	imageOps     RuntimeImageOps
	configOps    RuntimeSnapshotOps
	urbitOps     RuntimeUrbitOps
	wireguardOps RuntimeWireguardOps
	netdataOps   RuntimeNetdataOps
	minioOps     RuntimeMinioOps
	commandOps   RuntimeCommandOps
	volumeOps    RuntimeVolumeOps
	timerOps     RuntimeTimerOps
}

type WireguardRuntime struct {
	RuntimeContextOps
	RuntimeFileOps
	RuntimeImageOps
	RuntimeContainerOps
	RuntimeWireguardOps
	RuntimeVolumeOps
}

func newWireguardRuntime() WireguardRuntime {
	wireguardRuntime := wireguardRuntimeFromConfig()
	wireguardRuntime.StartContainerFn = StartContainer
	wireguardRuntime.UpdateContainerStateFn = config.UpdateContainerState
	return wireguardRuntime
}

func wireguardRuntimeFromConfig() WireguardRuntime {
	networkRuntime := network.NewNetworkRuntime()
	return WireguardRuntime{
		RuntimeContextOps: RuntimeContextOps{
			BasePathFn:  config.BasePath,
			DockerDirFn: config.DockerDir,
		},
		RuntimeFileOps: RuntimeFileOps{
			OpenFn:     os.Open,
			ReadFileFn: os.ReadFile,
			WriteFileFn: os.WriteFile,
			MkdirAllFn: os.MkdirAll,
		},
		RuntimeImageOps: RuntimeImageOps{
			GetLatestContainerInfoFn:  registry.GetLatestContainerInfo,
			GetLatestContainerImageFn: latestContainerImage,
		},
		RuntimeWireguardOps: RuntimeWireguardOps{
			CreateDefaultWGConfFn: config.CreateDefaultWGConf,
			GetWgConfFn:         config.GetWgConf,
			GetWgConfBlobFn:     getConfiguredStartramWGConfig,
			GetWgPrivkeyFn:      config.GetWgPrivkey,
			CopyFileToVolumeFn:   copyFileToVolumeWithTempContainer,
		},
		RuntimeVolumeOps: RuntimeVolumeOps{
			VolumeExistsFn: networkRuntime.VolumeExists,
			CreateVolumeFn: networkRuntime.CreateVolume,
		},
	}
}

func getConfiguredStartramWGConfig() (string, error) {
	return config.GetStartramConfig().Conf, nil
}

type UrbitRuntime struct {
	RuntimeSnapshotOps
	RuntimeUrbitOps
	RuntimeContainerOps
	RuntimeImageOps
	RuntimeContextOps
	RuntimeFileOps
}

func newUrbitRuntime() UrbitRuntime {
	urbitRuntime := urbitRuntimeFromConfig()
	urbitRuntime.StartContainerFn = StartContainer
	urbitRuntime.CreateContainerFn = CreateContainer
	urbitRuntime.UpdateContainerStateFn = config.UpdateContainerState
	return urbitRuntime
}

func urbitRuntimeFromConfig() UrbitRuntime {
	return UrbitRuntime{
		RuntimeSnapshotOps: RuntimeSnapshotOps{
			ConfFn:                    config.Conf,
			StartramSettingsSnapshotFn: config.StartramSettingsSnapshot,
			ShipSettingsSnapshotFn:     config.ShipSettingsSnapshot,
			ShipRuntimeSettingsSnapshotFn: config.ShipRuntimeSettingsSnapshot,
			GetStartramConfigFn:        config.GetStartramConfig,
			Check502SettingsSnapshotFn: config.Check502SettingsSnapshot,
		},
		RuntimeUrbitOps: RuntimeUrbitOps{
			LoadUrbitConfigFn:           config.LoadUrbitConfig,
			UrbitConfFn:                 config.UrbitConf,
			UpdateUrbitFn:               config.UpdateUrbit,
			UpdateUrbitRuntimeConfigFn:  config.UpdateUrbitRuntimeConfig,
			UpdateUrbitNetworkConfigFn:  config.UpdateUrbitNetworkConfig,
			UpdateUrbitScheduleConfigFn: config.UpdateUrbitScheduleConfig,
			UpdateUrbitFeatureConfigFn:  config.UpdateUrbitFeatureConfig,
			UpdateUrbitWebConfigFn:      config.UpdateUrbitWebConfig,
			UpdateUrbitBackupConfigFn:   config.UpdateUrbitBackupConfig,
		},
		RuntimeContextOps: RuntimeContextOps{
			ArchitectureFn: config.Architecture,
			DockerDirFn:    config.DockerDir,
		},
		RuntimeFileOps: RuntimeFileOps{
			WriteFileFn: os.WriteFile,
		},
		RuntimeImageOps: RuntimeImageOps{
			GetLatestContainerInfoFn:  registry.GetLatestContainerInfo,
			GetLatestContainerImageFn: latestContainerImage,
		},
	}
}

func newUrbitRuntimeForContainerConfig() UrbitRuntime {
	return urbitRuntimeFromConfig()
}

func newWireguardRuntimeForContainerConfig() WireguardRuntime {
	return wireguardRuntimeFromConfig()
}

func newDockerRuntime() dockerRuntime {
	networkRuntime := network.NewNetworkRuntime()
	return dockerRuntime{
		contextOps: RuntimeContextOps{
			BasePathFn: func() string {
				return config.BasePath()
			},
			ArchitectureFn: config.Architecture,
			DebugModeFn: func() bool {
				return config.DebugMode()
			},
			DockerDirFn: func() string {
				return config.DockerDir()
			},
		},
		fileOps: RuntimeFileOps{
			OpenFn:      os.Open,
			ReadFileFn:  os.ReadFile,
			WriteFileFn: os.WriteFile,
			MkdirAllFn:  os.MkdirAll,
		},
		containerOps: RuntimeContainerOps{
			StartContainerFn:            StartContainer,
			StopContainerByNameFn:       StopContainerByName,
			CreateContainerFn:           CreateContainer,
			RestartContainerFn:          RestartContainer,
			DeleteContainerFn:           DeleteContainer,
			GetContainerRunningStatusFn: GetContainerRunningStatus,
			AddOrGetNetworkFn:           networkRuntime.AddOrGetNetwork,
			GetContainerStateFn:         config.GetContainerState,
			UpdateContainerStateFn:      config.UpdateContainerState,
			GetShipStatusFn:             GetShipStatus,
		},
		imageOps: RuntimeImageOps{
			GetLatestContainerInfoFn:  GetLatestContainerInfo,
			GetLatestContainerImageFn: latestContainerImage,
		},
		configOps:    defaultRuntimeSnapshot(),
		urbitOps: defaultRuntimeUrbit(),
		wireguardOps: RuntimeWireguardOps{
			CreateDefaultWGConfFn: config.CreateDefaultWGConf,
			GetWgConfFn:           config.GetWgConf,
			GetWgConfBlobFn: func() (string, error) {
				return config.GetStartramConfig().Conf, nil
			},
			GetWgPrivkeyFn: config.GetWgPrivkey,
			CopyFileToVolumeFn: copyFileToVolumeWithTempContainer,
		},
		netdataOps: RuntimeNetdataOps{
			CreateDefaultNetdataConfFn: config.CreateDefaultNetdataConf,
		},
		minioOps: RuntimeMinioOps{
			CreateDefaultMcConfFn: config.CreateDefaultMcConf,
			SetMinIOPasswordFn:    config.SetMinIOPassword,
			GetMinIOPasswordFn:    config.GetMinIOPassword,
		},
		commandOps: RuntimeCommandOps{
			ExecDockerCommandFn: func(name string, cmd []string) (string, error) {
				out, _, err := ExecDockerCommand(name, cmd)
				return out, err
			},
			ExecDockerCommandExitFn: ExecDockerCommand,
			RandReadFn:              rand.Read,
			CopyFileToVolumeFn:      copyFileToVolumeWithTempContainer,
		},
		volumeOps: RuntimeVolumeOps{
			VolumeExistsFn: networkRuntime.VolumeExists,
			CreateVolumeFn: networkRuntime.CreateVolume,
		},
		timerOps: RuntimeTimerOps{
			SleepFn:        time.Sleep,
			PollIntervalFn: runtimePollInterval,
		},
	}
}

func latestContainerImage(containerType string) (string, error) {
	containerInfo, err := registry.GetLatestContainerInfo(containerType)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"]), nil
}

func runtimePollInterval() time.Duration {
	return 500 * time.Millisecond
}
