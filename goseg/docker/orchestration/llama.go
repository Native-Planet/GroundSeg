package orchestration

import (
	"groundseg/docker/orchestration/container"

	dockerc "github.com/docker/docker/api/types/container"
)

func LoadLlama() error {
	return loadLlamaWithRuntime(llamaRuntimeFromDocker(newDockerRuntime()))
}

func loadLlamaWithRuntime(rt container.LlamaRuntime) error {
	return container.LoadLlamaWithRuntime(rt)
}

func llamaApiContainerConf() (dockerc.Config, dockerc.HostConfig, error) {
	return llamaApiContainerConfWithRuntime(llamaRuntimeFromDocker(newDockerRuntime()))
}

func llamaApiContainerConfWithRuntime(rt container.LlamaRuntime) (dockerc.Config, dockerc.HostConfig, error) {
	return container.LlamaContainerConfWithRuntime(rt)
}
