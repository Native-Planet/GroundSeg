package orchestration

import "fmt"

import "groundseg/docker/orchestration/container"

func llamaRuntimeFromDocker(rt dockerRuntime) container.LlamaRuntime {
	return container.LlamaRuntime{
		ConfFn:                    rt.configOps.ConfFn,
		StopContainerByNameFn:     rt.containerOps.StopContainerByNameFn,
		StartContainerFn:          rt.containerOps.StartContainerFn,
		UpdateContainerStateFn:    rt.containerOps.UpdateContainerStateFn,
		GetLatestContainerImageFn: rt.imageOps.GetLatestContainerImageFn,
		VolumeExistsFn:            rt.volumeOps.VolumeExistsFn,
		CreateVolumeFn:            rt.volumeOps.CreateVolumeFn,
		AddOrGetNetworkFn:         rt.containerOps.AddOrGetNetworkFn,
		WriteFileFn:               rt.fileOps.WriteFileFn,
		VolumeDirFn:               rt.contextOps.DockerDirFn,
		DockerDirFn:               rt.contextOps.DockerDirFn,
		UrbitsConfigFn:            rt.urbitOps.UrbitConfAllFn,
	}
}

func netdataRuntimeFromDocker(rt dockerRuntime) container.NetdataRuntime {
	runtime := container.NetdataRuntime{
		OpenFn:               rt.fileOps.OpenFn,
		ReadFileFn:           rt.fileOps.ReadFileFn,
		WriteFileFn:          rt.fileOps.WriteFileFn,
		MkdirAllFn:           rt.fileOps.MkdirAllFn,
		StartContainerFn:     rt.containerOps.StartContainerFn,
		UpdateContainerState: rt.containerOps.UpdateContainerStateFn,
		CreateDefaultFn:      rt.netdataOps.CreateDefaultNetdataConfFn,
		//
		GetLatestContainerInfoFn:    rt.imageOps.GetLatestContainerInfoFn,
		GetLatestContainerImageFn:   rt.imageOps.GetLatestContainerImageFn,
		CopyFileToVolumeFn:          rt.commandOps.CopyFileToVolumeFn,
		VolumeExistsFn:              rt.volumeOps.VolumeExistsFn,
		CreateVolumeFn:              rt.volumeOps.CreateVolumeFn,
		DockerDirFn:                 rt.contextOps.DockerDirFn,
		BasePathFn:                  rt.contextOps.BasePathFn,
		GetContainerRunningStatusFn: rt.containerOps.GetContainerRunningStatusFn,
		SleepFn:                     rt.timerOps.SleepFn,
		PollIntervalFn:              rt.timerOps.PollIntervalFn,
	}
	if rt.netdataOps.WriteNDConfFn != nil {
		runtime.WriteNDConfFn = func() error {
			return rt.netdataOps.WriteNDConfFn(netdataRuntimeFromDocker(rt))
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
	return container.MinioRuntime{
		OpenFn:                      rt.fileOps.OpenFn,
		ConfFn:                      rt.configOps.ConfFn,
		BasePathFn:                  rt.contextOps.BasePathFn,
		DockerDirFn:                 rt.contextOps.DockerDirFn,
		ReadFileFn:                  rt.fileOps.ReadFileFn,
		WriteFileFn:                 rt.fileOps.WriteFileFn,
		MkdirAllFn:                  rt.fileOps.MkdirAllFn,
		StartContainerFn:            rt.containerOps.StartContainerFn,
		UpdateContainerStateFn:      rt.containerOps.UpdateContainerStateFn,
		GetContainerRunningStatusFn: rt.containerOps.GetContainerRunningStatusFn,
		GetLatestContainerInfoFn:    rt.imageOps.GetLatestContainerInfoFn,
		GetLatestContainerImageFn:   rt.imageOps.GetLatestContainerImageFn,
		LoadUrbitConfigFn:           rt.urbitOps.LoadUrbitConfigFn,
		UrbitConfFn:                 rt.urbitOps.UrbitConfFn,
		SetMinIOPasswordFn:          rt.minioOps.SetMinIOPasswordFn,
		GetMinIOPasswordFn:          rt.minioOps.GetMinIOPasswordFn,
		RandReadFn:                  rt.commandOps.RandReadFn,
		ExecCommandFn:               rt.commandOps.ExecDockerCommandFn,
		ExecCommandExitFn:           rt.commandOps.ExecDockerCommandExitFn,
		CopyFileToVolumeFn:          copyToVolumeOrError,
		VolumeExistsFn:              rt.volumeOps.VolumeExistsFn,
		CreateVolumeFn:              rt.volumeOps.CreateVolumeFn,
		SleepFn:                     rt.timerOps.SleepFn,
		PollIntervalFn:              rt.timerOps.PollIntervalFn,
		CreateDefaultMCConfFn:       rt.minioOps.CreateDefaultMcConfFn,
	}
}
