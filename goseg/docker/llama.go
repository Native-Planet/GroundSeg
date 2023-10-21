package docker

import (
	"fmt"
	"goseg/config"
	"goseg/defaults"
	"goseg/logger"
	"io/ioutil"
	"path/filepath"
	"runtime"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
)

func LoadLlama() error {
	logger.Logger.Info("Loading Llama GPT")
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
	var containerConfig container.Config
	var hostConfig container.HostConfig
	apiContainerName := "llama-gpt-api"
	desiredImage := "nativeplanet/llama-gpt:latest@sha256:4367338ef47a7f83072f76abbe2a3e49f1f3a8df9aa7c7ceb5632c9cbf66582d"
	lessCores := runtime.NumCPU() - 1
	exists, err := volumeExists(apiContainerName)
	if err != nil {
		return containerConfig, hostConfig, fmt.Errorf("Error checking volume: %v", err)
	}
	if !exists {
		if err = CreateVolume(apiContainerName); err != nil {
			return containerConfig, hostConfig, fmt.Errorf("Error creating volume: %v", err)
		}
	}
	exists, err = volumeExists(apiContainerName + "_app")
	if err != nil {
		return containerConfig, hostConfig, fmt.Errorf("Error checking volume: %v", err)
	}
	if !exists {
		if err = CreateVolume(apiContainerName + "_app"); err != nil {
			return containerConfig, hostConfig, fmt.Errorf("Error creating volume: %v", err)
		}
	}
	llamaNet, err := addOrGetNetwork("llama")
	if err != nil {
		return containerConfig, hostConfig, fmt.Errorf("Unable to create or get network: %v", err)
	}
	scriptPath := filepath.Join(config.DockerDir, apiContainerName+"_app", "_data", "run.sh")
	if err := ioutil.WriteFile(scriptPath, []byte(defaults.RunLlama), 0755); err != nil {
		return containerConfig, hostConfig, fmt.Errorf("Failed to write script: %v", err)
	}
	containerConfig = container.Config{
		Image:    desiredImage,
		Hostname: apiContainerName,
		Cmd:      []string{"/bin/sh", "/app/run.sh"},
		Env: []string{
			"MODEL=/models/code-llama-7b-chat.gguf",
			"MODEL_DOWNLOAD_URL=https://huggingface.co/TheBloke/CodeLlama-7B-Instruct-GGUF/resolve/main/codellama-7b-instruct.Q4_K_M.gguf",
			"N_GQA=1",
			"DEFAULT_SYSTEM_PROMPT=\"You are a helpful coding assistant. Use markdown when responding with code.\"",
			"WAIT_TIMEOUT=3600",
			"USE_MLOCK=1",
		},
		ExposedPorts: nat.PortSet{
			"8000/tcp": struct{}{},
		},
	}
	conf := config.Conf()
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
				Source: apiContainerName + "_app",
				Target: "/app",
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
			"DEFAULT_MODEL=/models/code-llama-7b-chat.gguf",
			`NEXT_PUBLIC_DEFAULT_SYSTEM_PROMPT=You are a helpful coding assistant. Use markdown when responding with code.`,
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
