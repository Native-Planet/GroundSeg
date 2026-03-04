package orchestration

import (
	"fmt"
	"os"
	"time"

	"groundseg/internal/seams"
	"groundseg/structs"
	
	"groundseg/config"
)

import "groundseg/docker/orchestration/container"

type netdataRuntimeOps = container.NetdataRuntime

type dockerContainerRuntimeInputs struct {
	shipSettingsSnapshotFn       func() config.ShipSettings
	startramSettingsSnapshotFn   func() config.StartramSettings
	penpaiSettingsSnapshotFn     func() config.PenpaiSettings
	basePathFn                  func() string
	dockerDirFn                 func() string
	openFn                      func(string) (*os.File, error)
	readFileFn                  func(string) ([]byte, error)
	writeFileFn                 func(string, []byte, os.FileMode) error
	mkdirAllFn                  func(string, os.FileMode) error
	startContainerFn            func(string, string) (structs.ContainerState, error)
	stopContainerByNameFn       func(string) error
	updateContainerStateFn      func(string, structs.ContainerState)
	getLatestContainerInfoFn    func(string) (map[string]string, error)
	getLatestContainerImageFn   func(string) (string, error)
	getContainerRunningStatusFn func(string) (string, error)
	addOrGetNetworkFn           func(string) (string, error)
	copyFileToVolumeFn          func(string, string, string, string, volumeWriterImageSelector) error
	volumeExistsFn              func(string) (bool, error)
	createVolumeFn              func(string) error
	sleepFn                     func(time.Duration)
	pollIntervalFn              func() time.Duration
	createDefaultNetdataFn      func() error
	writeNDConfFn               func() error
	getUrbitConfAllFn           func() map[string]structs.UrbitDocker
	loadUrbitConfigFn           func(string) error
	urbitConfFn                 func(string) structs.UrbitDocker
	randReadFn                  func([]byte) (int, error)
	execDockerCommandFn         func(string, []string) (string, error)
	execDockerCommandExitFn     func(string, []string) (string, int, error)
	setMinIOPasswordFn          func(string, string) error
	getMinIOPasswordFn          func(string) (string, error)
	createDefaultMcConfFn       func() error
}

func collectDockerContainerRuntimeInputs(rt dockerRuntime) dockerContainerRuntimeInputs {
		return dockerContainerRuntimeInputs{
			shipSettingsSnapshotFn:       rt.configOps.ShipSettingsSnapshotFn,
			startramSettingsSnapshotFn:   rt.configOps.StartramSettingsSnapshotFn,
			penpaiSettingsSnapshotFn:     rt.configOps.PenpaiSettingsSnapshotFn,
			basePathFn:                  rt.contextOps.BasePathFn,
			dockerDirFn:                 rt.contextOps.DockerDirFn,
		openFn:                      rt.fileOps.OpenFn,
		readFileFn:                  rt.fileOps.ReadFileFn,
		writeFileFn:                 rt.fileOps.WriteFileFn,
		mkdirAllFn:                  rt.fileOps.MkdirAllFn,
		startContainerFn:            rt.containerOps.StartContainerFn,
		stopContainerByNameFn:       rt.containerOps.StopContainerByNameFn,
		updateContainerStateFn:      rt.containerOps.UpdateContainerStateFn,
		getLatestContainerInfoFn:    rt.imageOps.GetLatestContainerInfoFn,
		getLatestContainerImageFn:   rt.imageOps.GetLatestContainerImageFn,
		getContainerRunningStatusFn: rt.containerOps.GetContainerRunningStatusFn,
		addOrGetNetworkFn:           rt.containerOps.AddOrGetNetworkFn,
		copyFileToVolumeFn:          rt.commandOps.CopyFileToVolumeFn,
		volumeExistsFn:              rt.volumeOps.VolumeExistsFn,
		createVolumeFn:              rt.volumeOps.CreateVolumeFn,
		sleepFn:                     rt.timerOps.SleepFn,
		pollIntervalFn:              rt.timerOps.PollIntervalFn,
		createDefaultNetdataFn:      rt.netdataOps.CreateDefaultNetdataConfFn,
		writeNDConfFn:               rt.netdataOps.WriteNDConfFn,
		getUrbitConfAllFn:           rt.urbitOps.UrbitConfAllFn,
		loadUrbitConfigFn:           rt.urbitOps.LoadUrbitConfigFn,
		urbitConfFn:                 rt.urbitOps.UrbitConfFn,
		randReadFn:                  rt.commandOps.RandReadFn,
		execDockerCommandFn:         rt.commandOps.ExecDockerCommandFn,
		execDockerCommandExitFn:     rt.commandOps.ExecDockerCommandExitFn,
		setMinIOPasswordFn:          rt.minioOps.SetMinIOPasswordFn,
		getMinIOPasswordFn:          rt.minioOps.GetMinIOPasswordFn,
		createDefaultMcConfFn:       rt.minioOps.CreateDefaultMcConfFn,
	}
}

func (inputs dockerContainerRuntimeInputs) applyLlamaRuntime(rt container.LlamaRuntime) container.LlamaRuntime {
	return seams.Merge(inputs.llamaRuntimeTemplate(), rt)
}

func (inputs dockerContainerRuntimeInputs) llamaRuntimeTemplate() container.LlamaRuntime {
	return container.LlamaRuntime{
		StartramSettingsSnapshotFn: inputs.startramSettingsSnapshotFn,
		PenpaiSettingsSnapshotFn:   inputs.penpaiSettingsSnapshotFn,
		ShipSettingsSnapshotFn:     inputs.shipSettingsSnapshotFn,
		StopContainerByNameFn:     inputs.stopContainerByNameFn,
		StartContainerFn:          inputs.startContainerFn,
		UpdateContainerStateFn:    inputs.updateContainerStateFn,
		GetLatestContainerImageFn: inputs.getLatestContainerImageFn,
		VolumeExistsFn:            inputs.volumeExistsFn,
		CreateVolumeFn:            inputs.createVolumeFn,
		AddOrGetNetworkFn:         inputs.addOrGetNetworkFn,
		WriteFileFn:               inputs.writeFileFn,
		VolumeDirFn:               inputs.dockerDirFn,
		DockerDirFn:               inputs.dockerDirFn,
		UrbitsConfigFn:            inputs.getUrbitConfAllFn,
	}
}

func (inputs dockerContainerRuntimeInputs) netdataRuntimeOps() netdataRuntimeOps {
	return netdataRuntimeOps{
		RuntimeFileOps: container.RuntimeFileOps{
			OpenFn:      inputs.openFn,
			ReadFileFn:  inputs.readFileFn,
			WriteFileFn: inputs.writeFileFn,
			MkdirAllFn:  inputs.mkdirAllFn,
		},
		StartContainerFn:            inputs.startContainerFn,
		UpdateContainerState:        inputs.updateContainerStateFn,
		CreateDefaultFn:             inputs.createDefaultNetdataFn,
		WriteNDConfFn:               inputs.writeNDConfFn,
		GetLatestContainerInfoFn:    inputs.getLatestContainerInfoFn,
		GetLatestContainerImageFn:   inputs.getLatestContainerImageFn,
		CopyFileToVolumeFn:          inputs.copyFileToVolumeFn,
		VolumeExistsFn:              inputs.volumeExistsFn,
		CreateVolumeFn:              inputs.createVolumeFn,
		DockerDirFn:                 inputs.dockerDirFn,
		BasePathFn:                  inputs.basePathFn,
		GetContainerRunningStatusFn: inputs.getContainerRunningStatusFn,
		SleepFn:                     inputs.sleepFn,
		PollIntervalFn:              inputs.pollIntervalFn,
	}
}

func (inputs dockerContainerRuntimeInputs) applyNetdataRuntime(rt container.NetdataRuntime) container.NetdataRuntime {
	return seams.Merge(container.NetdataRuntime(inputs.netdataRuntimeOps()), rt)
}

func (inputs dockerContainerRuntimeInputs) minioRuntimeTemplate() container.MinioRuntime {
	return container.MinioRuntime{
		StartramSettingsSnapshotFn:  inputs.startramSettingsSnapshotFn,
		ShipSettingsSnapshotFn:      inputs.shipSettingsSnapshotFn,
		BasePathFn:                  inputs.basePathFn,
		DockerDirFn:                 inputs.dockerDirFn,
		OpenFn:                      inputs.openFn,
		ReadFileFn:                  inputs.readFileFn,
		WriteFileFn:                 inputs.writeFileFn,
		MkdirAllFn:                  inputs.mkdirAllFn,
		StartContainerFn:            inputs.startContainerFn,
		UpdateContainerStateFn:      inputs.updateContainerStateFn,
		GetContainerRunningStatusFn: inputs.getContainerRunningStatusFn,
		GetLatestContainerInfoFn:    inputs.getLatestContainerInfoFn,
		GetLatestContainerImageFn:   inputs.getLatestContainerImageFn,
		LoadUrbitConfigFn:           inputs.loadUrbitConfigFn,
		UrbitConfFn:                 inputs.urbitConfFn,
		SetMinIOPasswordFn:          inputs.setMinIOPasswordFn,
		GetMinIOPasswordFn:          inputs.getMinIOPasswordFn,
		RandReadFn:                  inputs.randReadFn,
		ExecCommandFn:               inputs.execDockerCommandFn,
		ExecCommandExitFn:           inputs.execDockerCommandExitFn,
		CopyFileToVolumeFn:          inputs.copyFileToVolumeFn,
		VolumeExistsFn:              inputs.volumeExistsFn,
		CreateVolumeFn:              inputs.createVolumeFn,
		SleepFn:                     inputs.sleepFn,
		PollIntervalFn:              inputs.pollIntervalFn,
		CreateDefaultMCConfFn:       inputs.createDefaultMcConfFn,
	}
}

func (inputs dockerContainerRuntimeInputs) applyMinioRuntime(rt container.MinioRuntime) container.MinioRuntime {
	return seams.Merge(inputs.minioRuntimeTemplate(), rt)
}

func llamaRuntimeFromDocker(rt dockerRuntime) container.LlamaRuntime {
	return collectDockerContainerRuntimeInputs(rt).applyLlamaRuntime(container.LlamaRuntime{})
}

func netdataRuntimeFromDocker(rt dockerRuntime) container.NetdataRuntime {
	return collectDockerContainerRuntimeInputs(rt).applyNetdataRuntime(container.NetdataRuntime{})
}

func minioRuntimeFromDocker(rt dockerRuntime) container.MinioRuntime {
	copyToVolumeOrError := rt.commandOps.CopyFileToVolumeFn
	if copyToVolumeOrError == nil {
		copyToVolumeOrError = func(string, string, string, string, volumeWriterImageSelector) error {
			return fmt.Errorf("missing copy-to-volume runtime")
		}
	}
	inputs := collectDockerContainerRuntimeInputs(rt)
	runtime := inputs.applyMinioRuntime(container.MinioRuntime{})
	runtime.CopyFileToVolumeFn = copyToVolumeOrError
	return runtime
}
