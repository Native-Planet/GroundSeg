package container

import (
	"fmt"
	"groundseg/config"
	"groundseg/defaults"
	"groundseg/docker/orchestration/internal/artifactwriter"
	"groundseg/docker/registry"
	"os"
	"path/filepath"

	"groundseg/structs"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
)

type LlamaRuntime struct {
	PenpaiSettingsSnapshotFn   func() config.PenpaiSettings
	StartramSettingsSnapshotFn func() config.StartramSettings
	ShipSettingsSnapshotFn     func() config.ShipSettings
	StopContainerByNameFn      func(string) error
	StartContainerFn           func(string, string) (structs.ContainerState, error)
	UpdateContainerStateFn     func(string, structs.ContainerState)
	GetLatestContainerImageFn  func(string) (string, error)
	VolumeExistsFn             func(string) (bool, error)
	CreateVolumeFn             func(string) error
	AddOrGetNetworkFn          func(string) (string, error)
	WriteFileFn                func(string, []byte, os.FileMode) error
	VolumeDirFn                func() string
	DockerDirFn                func() string
	UrbitsConfigFn             func() map[string]structs.UrbitDocker
}

func LoadLlamaWithRuntime(rt LlamaRuntime) error {
	if rt.PenpaiSettingsSnapshotFn == nil {
		return fmt.Errorf("llama runtime requires penpai settings snapshot callback")
	}
	conf := rt.PenpaiSettingsSnapshotFn()
	if !conf.Allowed {
		zap.L().Info("Llama GPT disabled")
		return nil
	}
	zap.L().Info("Loading Llama GPT")
	if !conf.Running && rt.StopContainerByNameFn != nil {
		if err := rt.StopContainerByNameFn("llama-gpt-api"); err != nil {
			zap.L().Warn(fmt.Sprintf("Failed to kill Llama API: %v", err))
		}
	}
	if rt.StartContainerFn == nil {
		return fmt.Errorf("missing start container runtime")
	}
	info, err := rt.StartContainerFn("llama-gpt-api", "llama-api")
	if err != nil {
		return fmt.Errorf("start llama API container: %w", err)
	}
	if rt.UpdateContainerStateFn != nil {
		rt.UpdateContainerStateFn("llama-api", info)
	}
	return nil
}

func LlamaContainerConfWithRuntime(rt LlamaRuntime) (container.Config, container.HostConfig, error) {
	if rt.PenpaiSettingsSnapshotFn == nil || rt.ShipSettingsSnapshotFn == nil {
		return container.Config{}, container.HostConfig{}, fmt.Errorf("llama runtime requires settings snapshot callbacks")
	}
	penpaiSettings := rt.PenpaiSettingsSnapshotFn()
	shipSettings := rt.ShipSettingsSnapshotFn()
	if rt.VolumeDirFn == nil {
		return container.Config{}, container.HostConfig{}, fmt.Errorf("missing volume dir runtime")
	}
	if rt.DockerDirFn == nil {
		return container.Config{}, container.HostConfig{}, fmt.Errorf("missing docker dir runtime")
	}
	if rt.UrbitsConfigFn == nil {
		return container.Config{}, container.HostConfig{}, fmt.Errorf("missing urbits config runtime")
	}
	if rt.WriteFileFn == nil {
		return container.Config{}, container.HostConfig{}, fmt.Errorf("missing write file runtime")
	}
	selectImage := rt.GetLatestContainerImageFn
	if selectImage == nil {
		selectImage = registry.LatestContainerImage
	}
	var containerConfig container.Config
	var hostConfig container.HostConfig
	apiContainerName := "llama-gpt-api"
	desiredImage, err := selectImage("llama-api")
	if err != nil {
		return containerConfig, hostConfig, fmt.Errorf("lookup llama-api image: %w", err)
	}
	if err := ensureLlamaVolumesWithRuntime(rt, apiContainerName, apiContainerName+"_api"); err != nil {
		return containerConfig, hostConfig, err
	}
	if rt.AddOrGetNetworkFn == nil {
		return containerConfig, hostConfig, fmt.Errorf("missing network runtime")
	}
	llamaNet, err := rt.AddOrGetNetworkFn("llama")
	if err != nil {
		return containerConfig, hostConfig, fmt.Errorf("create or get llama network: %w", err)
	}
	scriptPath := filepath.Join(rt.DockerDirFn(), apiContainerName+"_api", "_data", "run.sh")
	if err := rt.WriteFileFn(scriptPath, []byte(defaults.RunLlama), 0755); err != nil {
		return containerConfig, hostConfig, fmt.Errorf("write llama startup script: %w", err)
	}
	var found *structs.Penpai
	for _, item := range penpaiSettings.Models {
		if item.ModelName == penpaiSettings.ActiveModel {
			found = &item
			break
		}
	}
	if found == nil {
		return containerConfig, hostConfig, fmt.Errorf("active penpai model %q not found", penpaiSettings.ActiveModel)
	}
	containerConfig = container.Config{
		Image:    desiredImage,
		Hostname: apiContainerName,
		Cmd:      []string{"/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"},
		Env: []string{
			fmt.Sprintf("MODEL=/models/%v", found.ModelName),
			fmt.Sprintf("MODEL_NAME=%v", found.ModelName),
			fmt.Sprintf("MODEL_DOWNLOAD_URL=%v", found.ModelUrl),
			"N_GQA=1",
			"USE_MLOCK=1",
		},
		ExposedPorts: nat.PortSet{
			"8000/tcp": struct{}{},
		},
	}
	var piers []string
	for _, pier := range shipSettings.Piers {
		if rt.UrbitsConfigFn()[pier].BootStatus == "boot" {
			piers = append(piers, pier)
		}
	}
	var binds []string
	for _, pier := range piers {
		hostPath := rt.VolumeDirFn() + "/" + pier + "/_data/" + pier + "/.urb/dev"
		volPath := "/piers/" + pier
		pierBind := hostPath + ":" + volPath
		binds = append(binds, pierBind)
	}
	hostConfig = container.HostConfig{
		NetworkMode: container.NetworkMode(llamaNet),
		RestartPolicy: container.RestartPolicy{
			Name: "on-failure",
		},
		PortBindings: nat.PortMap{
			"8000/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "3001",
				},
			},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: apiContainerName,
				Target: "/models",
			},
			{
				Type:   mount.TypeVolume,
				Source: apiContainerName + "_api",
				Target: "/api",
			},
		},
		Binds:  binds,
		CapAdd: []string{"IPC_LOCK"},
	}
	return containerConfig, hostConfig, nil
}

func ensureLlamaVolumesWithRuntime(rt LlamaRuntime, volumeNames ...string) error {
	if rt.VolumeExistsFn == nil || rt.CreateVolumeFn == nil {
		return nil
	}
	return artifactwriter.NewVolumeInitializationPlan(artifactwriter.VolumeOps{
		VolumeExistsFn: rt.VolumeExistsFn,
		CreateVolumeFn: rt.CreateVolumeFn,
	}, volumeNames...).EnsureVolumes()
}
