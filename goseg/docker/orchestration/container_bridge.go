package orchestration

import (
	"fmt"

	"groundseg/docker/orchestration/adapters/containerbridge"
	"groundseg/docker/orchestration/container"
)

func collectDockerContainerRuntimeInputs(rt dockerRuntime) containerbridge.RuntimeInputs {
	return containerbridge.RuntimeInputs{
		ShipSettingsSnapshotFn:      rt.configOps.ShipSettingsSnapshotFn,
		StartramSettingsSnapshotFn:  rt.configOps.StartramSettingsSnapshotFn,
		PenpaiSettingsSnapshotFn:    rt.configOps.PenpaiSettingsSnapshotFn,
		BasePathFn:                  rt.contextOps.BasePathFn,
		DockerDirFn:                 rt.contextOps.DockerDirFn,
		StopContainerByNameFn:       rt.containerOps.StopContainerByNameFn,
		GetContainerRunningStatusFn: rt.containerOps.GetContainerRunningStatusFn,
		AddOrGetNetworkFn:           rt.containerOps.AddOrGetNetworkFn,
		CreateDefaultNetdataFn:      rt.netdataOps.CreateDefaultNetdataConfFn,
		WriteNDConfFn:               rt.netdataOps.WriteNDConfFn,
		GetUrbitConfAllFn:           rt.urbitOps.UrbitConfAllFn,
		MinioRuntime: container.MinioRuntime{
			StartramSettingsSnapshotFn:  rt.configOps.StartramSettingsSnapshotFn,
			ShipSettingsSnapshotFn:      rt.configOps.ShipSettingsSnapshotFn,
			BasePathFn:                  rt.contextOps.BasePathFn,
			DockerDirFn:                 rt.contextOps.DockerDirFn,
			OpenFn:                      rt.fileOps.OpenFn,
			ReadFileFn:                  rt.fileOps.ReadFileFn,
			WriteFileFn:                 rt.fileOps.WriteFileFn,
			MkdirAllFn:                  rt.fileOps.MkdirAllFn,
			CreateDefaultMCConfFn:       rt.minioOps.CreateDefaultMcConfFn,
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
			CopyFileToVolumeFn:          rt.commandOps.CopyFileToVolumeFn,
			VolumeExistsFn:              rt.volumeOps.VolumeExistsFn,
			CreateVolumeFn:              rt.volumeOps.CreateVolumeFn,
			SleepFn:                     rt.timerOps.SleepFn,
			PollIntervalFn:              rt.timerOps.PollIntervalFn,
		},
	}
}

func llamaRuntimeFromDocker(rt dockerRuntime) container.LlamaRuntime {
	return containerbridge.LlamaRuntime(collectDockerContainerRuntimeInputs(rt), container.LlamaRuntime{})
}

func netdataRuntimeFromDocker(rt dockerRuntime) container.NetdataRuntime {
	return containerbridge.NetdataRuntime(collectDockerContainerRuntimeInputs(rt), container.NetdataRuntime{})
}

func minioRuntimeFromDocker(rt dockerRuntime) container.MinioRuntime {
	copyToVolumeOrError := rt.commandOps.CopyFileToVolumeFn
	if copyToVolumeOrError == nil {
		copyToVolumeOrError = func(string, string, string, string, volumeWriterImageSelector) error {
			return fmt.Errorf("missing copy-to-volume runtime")
		}
	}
	return containerbridge.MinioRuntime(
		collectDockerContainerRuntimeInputs(rt),
		container.MinioRuntime{},
		copyToVolumeOrError,
	)
}
