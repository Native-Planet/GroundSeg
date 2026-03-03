package orchestration

import (
	"crypto/rand"
	"fmt"
	"groundseg/config"
	"groundseg/docker/registry"
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
	runtime := wireguardRuntimeFromConfig()
	runtime.StartContainerFn = StartContainer
	runtime.UpdateContainerStateFn = config.UpdateContainerState
	return runtime
}

func wireguardRuntimeFromConfig() WireguardRuntime {
	seams := runtimeSeams()
	return WireguardRuntime{
		RuntimeContextOps:   seams.contextOps,
		RuntimeFileOps:      seams.fileOps,
		RuntimeImageOps:     seams.imageOps,
		RuntimeWireguardOps: seams.wireguardOps,
		RuntimeVolumeOps:    seams.volumeOps,
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
	runtime := urbitRuntimeFromConfig()
	runtime.StartContainerFn = StartContainer
	runtime.CreateContainerFn = CreateContainer
	runtime.UpdateContainerStateFn = config.UpdateContainerState
	return runtime
}

func urbitRuntimeFromConfig() UrbitRuntime {
	seams := runtimeSeams()
	return UrbitRuntime{
		RuntimeSnapshotOps: seams.snapshotOps,
		RuntimeUrbitOps:    seams.urbitOps,
		RuntimeContextOps:  seams.contextOps,
		RuntimeFileOps:     RuntimeFileOps{WriteFileFn: seams.fileOps.WriteFileFn},
		RuntimeImageOps:    seams.imageOps,
	}
}

func newUrbitRuntimeForContainerConfig() UrbitRuntime {
	return urbitRuntimeFromConfig()
}

func newWireguardRuntimeForContainerConfig() WireguardRuntime {
	return wireguardRuntimeFromConfig()
}

func newDockerRuntime() dockerRuntime {
	seams := runtimeSeams()
	return dockerRuntime{
		contextOps:   seams.contextOps,
		fileOps:      seams.fileOps,
		containerOps: defaultRuntimeContainerOps(),
		imageOps:     seams.imageOps,
		configOps:    seams.snapshotOps,
		urbitOps:     seams.urbitOps,
		wireguardOps: seams.wireguardOps,
		netdataOps:   seams.netdataOps,
		minioOps:     seams.minioOps,
		commandOps: RuntimeCommandOps{
			ExecDockerCommandFn: func(name string, cmd []string) (string, error) {
				out, _, err := ExecDockerCommand(name, cmd)
				return out, err
			},
			ExecDockerCommandExitFn: ExecDockerCommand,
			RandReadFn:              rand.Read,
			CopyFileToVolumeFn:      copyFileToVolumeWithTempContainer,
		},
		volumeOps: seams.volumeOps,
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
