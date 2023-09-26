package docker

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

func llamaApiContainerConf() (container.Config, container.HostConfig, error) {
	desiredImage := "ghcr.io/abetlen/llama-cpp-python:latest@sha256:b6d21ff8c4d9baad65e1fa741a0f8c898d68735fff3f3cd777e3f0c6a1839dd4"
	containerConfig := container.Config{
		Image:    desiredImage,
		Hostname: "llama-gpt-api",
		Cmd:      []string{"/bin/sh", "/api/run.sh"},
		Env: []string{
			"MODEL=/models/llama-2-7b-chat.bin",
			"MODEL_DOWNLOAD_URL=https://huggingface.co/TheBloke/Nous-Hermes-Llama-2-7B-GGML/resolve/main/nous-hermes-llama-2-7b.ggmlv3.q4_0.bin",
			"N_GQA=1",
			"USE_MLOCK=1",
		},
		ExposedPorts: nat.PortSet{
			"8000/tcp": struct{}{},
		},
	}
	hostConfig := container.HostConfig{
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
		Binds: []string{
			"./models:/models",
			"./api:/api",
		},
		CapAdd: []string{
			"IPC_LOCK",
		},
	}
	return containerConfig, hostConfig, nil
}

func llamaUIContainerConf() (container.Config, container.HostConfig, error) {
	desiredImage := "nativeplanet/llama-gpt-ui:latest@sha256:bf4811fe07c11a3a78b760f58b01ee11a61e0e9d6ec8a9e8832d3e14af428200"
	containerConfig := container.Config{
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
	hostConfig := container.HostConfig{
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
