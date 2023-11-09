package docker

import (
	"fmt"
	"goseg/config"
	"goseg/defaults"
	"goseg/logger"
	"goseg/structs"
	"io/ioutil"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
)

func LoadLlama() error {
	logger.Logger.Info("Loading Llama GPT")
	conf := config.Conf()
	if !conf.PenpaiRunning {
		if err := StopContainerByName("llama-gpt-api"); err != nil {
			logger.Logger.Warn(fmt.Sprintf("Failed to kill Llama API: %v", err))
		}
		if err := StopContainerByName("llama-gpt-ui"); err != nil {
			logger.Logger.Warn(fmt.Sprintf("Failed to kill Llama UI: %v", err))
		}
	}
	info, err := StartContainer("llama-gpt-api", "llama-api")
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Error starting Llama API: %v", err))
	}
	config.UpdateContainerState("llama-api", info)
	info, err = StartContainer("llama-gpt-ui", "llama-ui")
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Error starting Llama UI: %v", err))
	}
	config.UpdateContainerState("llama-ui", info)
	return nil
}

func llamaApiContainerConf() (container.Config, container.HostConfig, error) {
	conf := config.Conf()
	var containerConfig container.Config
	var hostConfig container.HostConfig
	apiContainerName := "llama-gpt-api"
	desiredImage := "nativeplanet/llama-gpt:latest@sha256:6a2123f4f67713b3dccfb4b69ca964f9a65710f261ff69348dd88f57bd3e6a79"
	lessCores := conf.PenpaiCores
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
		if item.ModelTitle == conf.PenpaiActive {
			found = &item
			break
		}
	}
	containerConfig = container.Config{
		Image:    desiredImage,
		Hostname: apiContainerName,
		Cmd:      []string{"/bin/sh", "/api/run.sh"},
		Env: []string{
			fmt.Sprintf("MODEL=/models/%v", found.ModelName),
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
		Resources: container.Resources{
			NanoCPUs: int64(lessCores) * 1e9,
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

func llamaUIContainerConf() (container.Config, container.HostConfig, error) {
	desiredImage := "nativeplanet/llama-gpt-ui:latest@sha256:bf4811fe07c11a3a78b760f58b01ee11a61e0e9d6ec8a9e8832d3e14af428200"
	var containerConfig container.Config
	var hostConfig container.HostConfig
	llamaNet, err := addOrGetNetwork("llama")
	if err != nil {
		return containerConfig, hostConfig, fmt.Errorf("Unable to create or get network: %v", err)
	}
	containerConfig = container.Config{
		Image:    desiredImage,
		Hostname: "llama-gpt-ui",
		Env: []string{
			"OPENAI_API_KEY=sk-XXXXXXXXXXXXXXXXXXXX",
			"OPENAI_API_HOST=http://llama-gpt-api:8000",
			"DEFAULT_MODEL=/models/llama-2-7b-chat.bin",
			`NEXT_PUBLIC_DEFAULT_SYSTEM_PROMPT=You are a helpful and friendly AI assistant. Respond very concisely.`,
			"WAIT_HOSTS=llama-gpt-api:8000",
			"WAIT_TIMEOUT=3600",
		},
		ExposedPorts: nat.PortSet{
			"3000/tcp": struct{}{},
		},
	}
	hostConfig = container.HostConfig{
		NetworkMode: container.NetworkMode(llamaNet),
		RestartPolicy: container.RestartPolicy{
			Name: "on-failure",
		},
		PortBindings: nat.PortMap{
			"3000/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "3002",
				},
			},
		},
	}
	return containerConfig, hostConfig, nil
}
