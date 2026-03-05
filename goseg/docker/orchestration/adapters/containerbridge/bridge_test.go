package containerbridge

import (
	"os"
	"testing"
	"time"

	"groundseg/config"
	"groundseg/docker/orchestration/container"
	"groundseg/docker/registry"
	"groundseg/structs"
)

func sampleInputs() RuntimeInputs {
	return RuntimeInputs{
		ShipSettingsSnapshotFn:      func() config.ShipSettings { return config.ShipSettings{} },
		StartramSettingsSnapshotFn:  func() config.StartramSettings { return config.StartramSettings{} },
		PenpaiSettingsSnapshotFn:    func() config.PenpaiSettings { return config.PenpaiSettings{} },
		BasePathFn:                  func() string { return "/tmp/base" },
		DockerDirFn:                 func() string { return "/tmp/docker/" },
		StopContainerByNameFn:       func(string) error { return nil },
		GetContainerRunningStatusFn: func(string) (string, error) { return "running", nil },
		AddOrGetNetworkFn:           func(string) (string, error) { return "network", nil },
		GetUrbitConfAllFn:           func() map[string]structs.UrbitDocker { return map[string]structs.UrbitDocker{} },
		CreateDefaultNetdataFn:      func() error { return nil },
		WriteNDConfFn:               func() error { return nil },
		MinioRuntime: container.MinioRuntime{
			BasePathFn:                func() string { return "/tmp/base" },
			DockerDirFn:               func() string { return "/tmp/docker/" },
			OpenFn:                    func(string) (*os.File, error) { return nil, nil },
			ReadFileFn:                func(string) ([]byte, error) { return nil, nil },
			WriteFileFn:               func(string, []byte, os.FileMode) error { return nil },
			MkdirAllFn:                func(string, os.FileMode) error { return nil },
			StartContainerFn:          func(string, string) (structs.ContainerState, error) { return structs.ContainerState{}, nil },
			UpdateContainerStateFn:    func(string, structs.ContainerState) {},
			GetLatestContainerInfoFn:  func(string) (registry.ImageDescriptor, error) { return registry.ImageDescriptor{}, nil },
			GetLatestContainerImageFn: func(string) (string, error) { return "image:latest", nil },
			ExecCommandFn:             func(string, []string) (string, error) { return "", nil },
			ExecCommandExitFn:         func(string, []string) (string, int, error) { return "", 0, nil },
			CopyFileToVolumeFn:        func(string, string, string, string, func() (string, error)) error { return nil },
			VolumeExistsFn:            func(string) (bool, error) { return true, nil },
			CreateVolumeFn:            func(string) error { return nil },
			SleepFn:                   func(time.Duration) {},
			PollIntervalFn:            func() time.Duration { return time.Second },
		},
	}
}

func TestLlamaRuntimeBuildsFromInputs(t *testing.T) {
	inputs := sampleInputs()

	rt := LlamaRuntime(inputs, container.LlamaRuntime{})
	if rt.StartContainerFn == nil {
		t.Fatal("expected llama runtime start container callback")
	}
	if rt.StopContainerByNameFn == nil {
		t.Fatal("expected llama runtime stop callback")
	}
	if rt.GetLatestContainerImageFn == nil {
		t.Fatal("expected llama runtime image callback")
	}
}

func TestNetdataRuntimeBuildsFromInputs(t *testing.T) {
	inputs := sampleInputs()

	rt := NetdataRuntime(inputs, container.NetdataRuntime{})
	if rt.StartContainerFn == nil {
		t.Fatal("expected netdata runtime start container callback")
	}
	if rt.WriteNDConfFn == nil {
		t.Fatal("expected netdata runtime write config callback")
	}
	if rt.DockerDirFn == nil {
		t.Fatal("expected netdata runtime docker dir callback")
	}
}

func TestMinioRuntimeMergesInputsAndOverridesCopyToVolume(t *testing.T) {
	inputs := sampleInputs()
	copyToVolume := func(string, string, string, string, func() (string, error)) error { return nil }

	rt := MinioRuntime(inputs, container.MinioRuntime{}, copyToVolume)
	if rt.StartContainerFn == nil {
		t.Fatal("expected minio runtime start container callback")
	}
	if rt.ExecCommandExitFn == nil {
		t.Fatal("expected minio runtime exec exit callback")
	}
	if rt.CopyFileToVolumeFn == nil {
		t.Fatal("expected minio runtime copy-to-volume callback")
	}
}
