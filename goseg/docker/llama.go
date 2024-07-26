package docker

import (
	"fmt"
	"groundseg/config"
	"groundseg/defaults"
	"groundseg/structs"
	"io/ioutil"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
)

func LoadLlama() error {
	conf := config.Conf()
	if !conf.PenpaiAllow {
		zap.L().Info("Llama GPT disabled")
		return nil
	}
	zap.L().Info("Loading Llama GPT")
	if !conf.PenpaiRunning {
		if err := StopContainerByName("llama-gpt-api"); err != nil {
			zap.L().Warn(fmt.Sprintf("Failed to kill Llama API: %v", err))
		}
	}
	info, err := StartContainer("llama-gpt-api", "llama-api")
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Error starting Llama API: %v", err))
	}
	config.UpdateContainerState("llama-api", info)
	return nil
}

func llamaApiContainerConf() (container.Config, container.HostConfig, error) {
	conf := config.Conf()
	var containerConfig container.Config
	var hostConfig container.HostConfig
	apiContainerName := "llama-gpt-api"
	desiredImage := "nativeplanet/llama-gpt:dev@sha256:ac2dcfac72bc3d8ee51ee255edecc10072ef9c0f958120971c00be5f4944a6fa"
	// lessCores := conf.PenpaiCores
	exists, err := volumeExists(apiContainerName)
	if err != nil {
		return containerConfig, hostConfig, fmt.Errorf("Error checking volume: %v", err)
	}
	if !exists {
		if err = CreateVolume(apiContainerName); err != nil {
			return containerConfig, hostConfig, fmt.Errorf("Error creating volume: %v", err)
		}
	}
	exists, err = volumeExists(apiContainerName + "_api")
	if err != nil {
		return containerConfig, hostConfig, fmt.Errorf("Error checking volume: %v", err)
	}
	if !exists {
		if err = CreateVolume(apiContainerName + "_api"); err != nil {
			return containerConfig, hostConfig, fmt.Errorf("Error creating volume: %v", err)
		}
	}
	llamaNet, err := addOrGetNetwork("llama")
	if err != nil {
		return containerConfig, hostConfig, fmt.Errorf("Unable to create or get network: %v", err)
	}
	scriptPath := filepath.Join(config.DockerDir, apiContainerName+"_api", "_data", "run.sh")
	if err := ioutil.WriteFile(scriptPath, []byte(defaults.RunLlama), 0755); err != nil {
		return containerConfig, hostConfig, fmt.Errorf("Failed to write script: %v", err)
	}
	var found *structs.Penpai
	for _, item := range conf.PenpaiModels {
		if item.ModelName == conf.PenpaiActive {
			found = &item
			break
		}
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
	for _, pier := range conf.Piers {
		if config.UrbitsConfig[pier].BootStatus == "boot" {
			piers = append(piers, pier)
		}
	}
	var binds []string
	for _, pier := range piers {
		hostPath := VolumeDir + "/" + pier + "/_data/" + pier + "/.urb/dev"
		volPath := "/piers/" + pier
		pierBind := hostPath + ":" + volPath
		binds = append(binds, pierBind)
	}
	hostConfig = container.HostConfig{
		NetworkMode: container.NetworkMode(llamaNet),
		RestartPolicy: container.RestartPolicy{
			Name: "on-failure",
		},
		// Resources: container.Resources{
		// 	NanoCPUs: int64(lessCores) * 1e9,
		// },
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
				Source: apiContainerName, // host dir
				Target: "/models",        // in the container
			},
			{
				Type:   mount.TypeVolume,
				Source: apiContainerName + "_api",
				Target: "/api",
			},
		},
		Binds: binds,
		CapAdd: []string{
			"IPC_LOCK",
		},
	}
	return containerConfig, hostConfig, nil
}
