package containerbridge

import (
	"groundseg/config"
	"groundseg/docker/orchestration/container"
	"groundseg/internal/seams"
	"groundseg/structs"
)

type RuntimeInputs struct {
	ShipSettingsSnapshotFn      func() config.ShipSettings
	StartramSettingsSnapshotFn  func() config.StartramSettings
	PenpaiSettingsSnapshotFn    func() config.PenpaiSettings
	BasePathFn                  func() string
	DockerDirFn                 func() string
	StopContainerByNameFn       func(string) error
	GetContainerRunningStatusFn func(string) (string, error)
	AddOrGetNetworkFn           func(string) (string, error)
	CreateDefaultNetdataFn      func() error
	WriteNDConfFn               func() error
	GetUrbitConfAllFn           func() map[string]structs.UrbitDocker
	MinioRuntime                container.MinioRuntime
}

func LlamaRuntime(inputs RuntimeInputs, overrides container.LlamaRuntime) container.LlamaRuntime {
	return seams.Merge(inputs.llamaRuntimeTemplate(), overrides)
}

func NetdataRuntime(inputs RuntimeInputs, overrides container.NetdataRuntime) container.NetdataRuntime {
	return seams.Merge(container.NetdataRuntime(inputs.netdataRuntimeTemplate()), overrides)
}

func MinioRuntime(
	inputs RuntimeInputs,
	overrides container.MinioRuntime,
	copyToVolumeFn func(string, string, string, string, func() (string, error)) error,
) container.MinioRuntime {
	runtime := seams.Merge(inputs.MinioRuntime, overrides)
	runtime.CopyFileToVolumeFn = copyToVolumeFn
	return runtime
}

func (inputs RuntimeInputs) llamaRuntimeTemplate() container.LlamaRuntime {
	return container.LlamaRuntime{
		StartramSettingsSnapshotFn: inputs.StartramSettingsSnapshotFn,
		PenpaiSettingsSnapshotFn:   inputs.PenpaiSettingsSnapshotFn,
		ShipSettingsSnapshotFn:     inputs.ShipSettingsSnapshotFn,
		StopContainerByNameFn:      inputs.StopContainerByNameFn,
		StartContainerFn:           inputs.MinioRuntime.StartContainerFn,
		UpdateContainerStateFn:     inputs.MinioRuntime.UpdateContainerStateFn,
		GetLatestContainerImageFn:  inputs.MinioRuntime.GetLatestContainerImageFn,
		VolumeExistsFn:             inputs.MinioRuntime.VolumeExistsFn,
		CreateVolumeFn:             inputs.MinioRuntime.CreateVolumeFn,
		AddOrGetNetworkFn:          inputs.AddOrGetNetworkFn,
		WriteFileFn:                inputs.MinioRuntime.WriteFileFn,
		VolumeDirFn:                inputs.DockerDirFn,
		DockerDirFn:                inputs.DockerDirFn,
		UrbitsConfigFn:             inputs.GetUrbitConfAllFn,
	}
}

func (inputs RuntimeInputs) netdataRuntimeTemplate() container.NetdataRuntime {
	return container.NetdataRuntime{
		RuntimeFileOps: container.RuntimeFileOps{
			OpenFn:      inputs.MinioRuntime.OpenFn,
			ReadFileFn:  inputs.MinioRuntime.ReadFileFn,
			WriteFileFn: inputs.MinioRuntime.WriteFileFn,
			MkdirAllFn:  inputs.MinioRuntime.MkdirAllFn,
		},
		StartContainerFn:            inputs.MinioRuntime.StartContainerFn,
		UpdateContainerState:        inputs.MinioRuntime.UpdateContainerStateFn,
		CreateDefaultFn:             inputs.CreateDefaultNetdataFn,
		WriteNDConfFn:               inputs.WriteNDConfFn,
		GetLatestContainerInfoFn:    inputs.MinioRuntime.GetLatestContainerInfoFn,
		GetLatestContainerImageFn:   inputs.MinioRuntime.GetLatestContainerImageFn,
		CopyFileToVolumeFn:          inputs.MinioRuntime.CopyFileToVolumeFn,
		VolumeExistsFn:              inputs.MinioRuntime.VolumeExistsFn,
		CreateVolumeFn:              inputs.MinioRuntime.CreateVolumeFn,
		DockerDirFn:                 inputs.DockerDirFn,
		BasePathFn:                  inputs.BasePathFn,
		GetContainerRunningStatusFn: inputs.GetContainerRunningStatusFn,
		SleepFn:                     inputs.MinioRuntime.SleepFn,
		PollIntervalFn:              inputs.MinioRuntime.PollIntervalFn,
	}
}
