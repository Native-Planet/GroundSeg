package orchestration

import "fmt"

import "groundseg/docker/orchestration/container"

type dockerContainerRuntimeInputs struct {
	confFn                      func() structs.SysConfig
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
	addOrGetNetworkFn          func(string) (string, error)
	copyFileToVolumeFn         func(string, string, string, string, volumeWriterImageSelector) error
	volumeExistsFn             func(string) (bool, error)
	createVolumeFn             func(string) error
	sleepFn                    func(time.Duration)
	pollIntervalFn             func() time.Duration
	createDefaultNetdataFn      func() error
	writeNDConfFn              func() error
	getUrbitConfAllFn          func() map[string]structs.UrbitDocker
	loadUrbitConfigFn          func(string) error
	urbitConfFn                func(string) structs.UrbitDocker
	randReadFn                 func([]byte) (int, error)
	execDockerCommandFn        func(string, []string) (string, error)
	execDockerCommandExitFn    func(string, []string) (string, int, error)
	setMinIOPasswordFn         func(string, string) error
	getMinIOPasswordFn         func(string) (string, error)
	createDefaultMcConfFn       func() error
	getWgConfBlobFn            func() (string, error)
	createDefaultWGConfFn       func() error
}

func collectDockerContainerRuntimeInputs(rt dockerRuntime) dockerContainerRuntimeInputs {
	return dockerContainerRuntimeInputs{
		confFn:                      rt.configOps.ConfFn,
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
		addOrGetNetworkFn:          rt.containerOps.AddOrGetNetworkFn,
		copyFileToVolumeFn:         rt.commandOps.CopyFileToVolumeFn,
		volumeExistsFn:             rt.volumeOps.VolumeExistsFn,
		createVolumeFn:             rt.volumeOps.CreateVolumeFn,
		sleepFn:                    rt.timerOps.SleepFn,
		pollIntervalFn:             rt.timerOps.PollIntervalFn,
		createDefaultNetdataFn:      rt.netdataOps.CreateDefaultNetdataConfFn,
		writeNDConfFn:              rt.netdataOps.WriteNDConfFn,
		getUrbitConfAllFn:          rt.urbitOps.UrbitConfAllFn,
		loadUrbitConfigFn:          rt.urbitOps.LoadUrbitConfigFn,
		urbitConfFn:                rt.urbitOps.UrbitConfFn,
		randReadFn:                 rt.commandOps.RandReadFn,
		execDockerCommandFn:        rt.commandOps.ExecDockerCommandFn,
		execDockerCommandExitFn:    rt.commandOps.ExecDockerCommandExitFn,
		setMinIOPasswordFn:         rt.minioOps.SetMinIOPasswordFn,
		getMinIOPasswordFn:         rt.minioOps.GetMinIOPasswordFn,
		createDefaultMcConfFn:       rt.minioOps.CreateDefaultMcConfFn,
	}
}

func (inputs dockerContainerRuntimeInputs) applyLlamaRuntime(rt container.LlamaRuntime) container.LlamaRuntime {
	return container.LlamaRuntime{
		ConfFn:                    inputs.confFn,
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

func (inputs dockerContainerRuntimeInputs) applyNetdataRuntime(rt container.NetdataRuntime) container.NetdataRuntime {
	rt.OpenFn = inputs.openFn
	rt.ReadFileFn = inputs.readFileFn
	rt.WriteFileFn = inputs.writeFileFn
	rt.MkdirAllFn = inputs.mkdirAllFn
	rt.StartContainerFn = inputs.startContainerFn
	rt.UpdateContainerState = inputs.updateContainerStateFn
	rt.CreateDefaultFn = inputs.createDefaultNetdataFn
	rt.GetLatestContainerInfoFn = inputs.getLatestContainerInfoFn
	rt.GetLatestContainerImageFn = inputs.getLatestContainerImageFn
	rt.CopyFileToVolumeFn = inputs.copyFileToVolumeFn
	rt.VolumeExistsFn = inputs.volumeExistsFn
	rt.CreateVolumeFn = inputs.createVolumeFn
	rt.DockerDirFn = inputs.dockerDirFn
	rt.BasePathFn = inputs.basePathFn
	rt.GetContainerRunningStatusFn = inputs.getContainerRunningStatusFn
	rt.SleepFn = inputs.sleepFn
	rt.PollIntervalFn = inputs.pollIntervalFn
	return rt
}

func (inputs dockerContainerRuntimeInputs) applyMinioRuntime(rt container.MinioRuntime) container.MinioRuntime {
	rt.ConfFn = inputs.confFn
	rt.BasePathFn = inputs.basePathFn
	rt.DockerDirFn = inputs.dockerDirFn
	rt.ReadFileFn = inputs.readFileFn
	rt.WriteFileFn = inputs.writeFileFn
	rt.MkdirAllFn = inputs.mkdirAllFn
	rt.StartContainerFn = inputs.startContainerFn
	rt.UpdateContainerStateFn = inputs.updateContainerStateFn
	rt.GetContainerRunningStatusFn = inputs.getContainerRunningStatusFn
	rt.GetLatestContainerInfoFn = inputs.getLatestContainerInfoFn
	rt.GetLatestContainerImageFn = inputs.getLatestContainerImageFn
	rt.LoadUrbitConfigFn = inputs.loadUrbitConfigFn
	rt.UrbitConfFn = inputs.urbitConfFn
	rt.SetMinIOPasswordFn = inputs.setMinIOPasswordFn
	rt.GetMinIOPasswordFn = inputs.getMinIOPasswordFn
	rt.RandReadFn = inputs.randReadFn
	rt.ExecCommandFn = inputs.execDockerCommandFn
	rt.ExecCommandExitFn = inputs.execDockerCommandExitFn
	rt.CopyFileToVolumeFn = inputs.copyFileToVolumeFn
	rt.VolumeExistsFn = inputs.volumeExistsFn
	rt.CreateVolumeFn = inputs.createVolumeFn
	rt.SleepFn = inputs.sleepFn
	rt.PollIntervalFn = inputs.pollIntervalFn
	rt.CreateDefaultMCConfFn = inputs.createDefaultMcConfFn
	return rt
}

func llamaRuntimeFromDocker(rt dockerRuntime) container.LlamaRuntime {
	return collectDockerContainerRuntimeInputs(rt).applyLlamaRuntime(container.LlamaRuntime{})
}

func netdataRuntimeFromDocker(rt dockerRuntime) container.NetdataRuntime {
	runtime := collectDockerContainerRuntimeInputs(rt).applyNetdataRuntime(container.NetdataRuntime{})
	if rt.netdataOps.WriteNDConfFn != nil {
		runtime.WriteNDConfFn = func() error {
			return rt.netdataOps.WriteNDConfFn()
		}
	}
	return runtime
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
