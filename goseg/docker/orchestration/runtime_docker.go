package orchestration

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"groundseg/config"
	"groundseg/docker/registry"
	"groundseg/structs"
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
	runtime := wireguardRuntimeFromConfig()
	runtime.StartContainerFn = StartContainer
	runtime.UpdateContainerStateFn = config.UpdateContainerState
	return runtime
}

func newWireguardRuntimeForTests() WireguardRuntime {
	runtime := newWireguardRuntime()
	runtime.BasePathFn = func() string { return "/tmp" }
	runtime.DockerDirFn = func() string { return "/tmp/docker" }
	runtime.OpenFn = func(string) (*os.File, error) { return nil, nil }
	runtime.ReadFileFn = func(string) ([]byte, error) { return nil, os.ErrNotExist }
	runtime.WriteFileFn = func(string, []byte, os.FileMode) error { return nil }
	runtime.MkdirAllFn = func(string, os.FileMode) error { return nil }
	runtime.StartContainerFn = func(string, string) (structs.ContainerState, error) { return structs.ContainerState{}, nil }
	runtime.UpdateContainerStateFn = func(string, structs.ContainerState) {}
	runtime.GetLatestContainerInfoFn = func(string) (map[string]string, error) {
		return map[string]string{"repo": "repo/wg", "tag": "latest", "hash": "hash"}, nil
	}
	runtime.GetLatestContainerImageFn = func(string) (string, error) { return "wg:latest", nil }
	runtime.CreateDefaultWGConfFn = func() error { return nil }
	runtime.GetWgConfFn = func() (structs.WgConfig, error) { return structs.WgConfig{}, nil }
	runtime.GetWgConfBlobFn = func() (string, error) {
		return base64.StdEncoding.EncodeToString([]byte("blob")), nil
	}
	runtime.GetWgPrivkeyFn = func() string { return "k1" }
	runtime.CopyFileToVolumeFn = func(string, string, string, string, volumeWriterImageSelector) error { return nil }
	runtime.VolumeExistsFn = func(string) (bool, error) { return false, nil }
	runtime.CreateVolumeFn = func(string) error { return nil }
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
