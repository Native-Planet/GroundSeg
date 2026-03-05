package orchestration

import (
	"groundseg/config"
	"groundseg/config/runtimecontext"
	"groundseg/docker/network"
	"groundseg/docker/registry"
	"os"
)

type runtimeSeamRegistry struct {
	contextOps   RuntimeContextOps
	fileOps      RuntimeFileOps
	imageOps     RuntimeImageOps
	snapshotOps  RuntimeSnapshotOps
	urbitOps     RuntimeUrbitOps
	wireguardOps RuntimeWireguardOps
	netdataOps   RuntimeNetdataOps
	minioOps     RuntimeMinioOps
	volumeOps    RuntimeVolumeOps
}

func runtimeSeams() runtimeSeamRegistry {
	return buildRuntimeSeamBundle()
}

func buildRuntimeSeamBundle() runtimeSeamRegistry {
	networkRuntime := network.NewNetworkRuntime()
	return runtimeSeamRegistry{
		contextOps: RuntimeContextOps{
			BasePathFn: func() string {
				return runtimecontext.Snapshot().BasePath
			},
			ArchitectureFn: func() string {
				return runtimecontext.Snapshot().Architecture
			},
			DebugModeFn: func() bool {
				return runtimecontext.Snapshot().DebugMode
			},
			DockerDirFn: func() string {
				return runtimecontext.Snapshot().DockerDir
			},
		},
		fileOps: RuntimeFileOps{
			OpenFn:      os.Open,
			ReadFileFn:  os.ReadFile,
			WriteFileFn: os.WriteFile,
			MkdirAllFn:  os.MkdirAll,
		},
		imageOps: RuntimeImageOps{
			GetLatestContainerInfoFn:  registry.GetLatestContainerInfo,
			GetLatestContainerImageFn: latestContainerImage,
		},
		snapshotOps:  defaultRuntimeSnapshot(),
		urbitOps:     defaultRuntimeUrbit(),
		wireguardOps: defaultRuntimeWireguardOps(),
		netdataOps: RuntimeNetdataOps{
			CreateDefaultNetdataConfFn: config.CreateDefaultNetdataConf,
		},
		minioOps: RuntimeMinioOps{
			CreateDefaultMcConfFn: config.CreateDefaultMcConf,
			SetMinIOPasswordFn:    config.SetMinIOPassword,
			GetMinIOPasswordFn:    config.GetMinIOPassword,
		},
		volumeOps: RuntimeVolumeOps{
			VolumeExistsFn: networkRuntime.VolumeExists,
			CreateVolumeFn: networkRuntime.CreateVolume,
		},
	}
}
