package orchestration

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"groundseg/transition"
)

type containerConfigBuilder func(dockerRuntime, string) (container.Config, container.HostConfig, error)

var containerConfigBuilders = map[transition.ContainerType]containerConfigBuilder{
	transition.ContainerTypeVere:      urbitContainerConfigBuilder,
	transition.ContainerTypeNetdata:   netdataContainerConfigBuilder,
	transition.ContainerTypeMinio:     minioContainerConfigBuilder,
	transition.ContainerTypeMinioMC:   mcContainerConfigBuilder,
	transition.ContainerTypeWireguard: wgContainerConfigBuilder,
	transition.ContainerTypeLlamaAPI:  llamaApiContainerConfigBuilder,
}

func urbitContainerConfigBuilder(rt dockerRuntime, containerName string) (container.Config, container.HostConfig, error) {
	return urbitContainerConfWithRuntime(urbitRuntimeFromDocker(rt), containerName)
}

func netdataContainerConfigBuilder(rt dockerRuntime, _ string) (container.Config, container.HostConfig, error) {
	return netdataContainerConfWithRuntime(netdataRuntimeFromDocker(rt))
}

func minioContainerConfigBuilder(rt dockerRuntime, containerName string) (container.Config, container.HostConfig, error) {
	return minioContainerConfWithRuntime(minioRuntimeFromDocker(rt), containerName)
}

func mcContainerConfigBuilder(rt dockerRuntime, _ string) (container.Config, container.HostConfig, error) {
	return mcContainerConfWithRuntime(minioRuntimeFromDocker(rt))
}

func wgContainerConfigBuilder(rt dockerRuntime, _ string) (container.Config, container.HostConfig, error) {
	return wireguardRuntimeFromDocker(rt).wgContainerConf()
}

func llamaApiContainerConfigBuilder(rt dockerRuntime, _ string) (container.Config, container.HostConfig, error) {
	return llamaApiContainerConfWithRuntime(llamaRuntimeFromDocker(rt))
}

func ContainerConfigForType(containerName string, containerType transition.ContainerType) (container.Config, container.HostConfig, error) {
	return ContainerConfigForTypeWithRuntime(newDockerRuntime(), containerName, containerType)
}

// ContainerConfigForTypeString parses external string input and delegates to the typed API.
func ContainerConfigForTypeString(containerName string, containerType string) (container.Config, container.HostConfig, error) {
	return ContainerConfigForTypeWithRuntime(newDockerRuntime(), containerName, transition.ContainerType(containerType))
}

func ContainerConfigForTypeWithRuntime(rt dockerRuntime, containerName string, containerType transition.ContainerType) (container.Config, container.HostConfig, error) {
	builder, ok := containerConfigBuilders[containerType]
	if !ok {
		return container.Config{}, container.HostConfig{}, fmt.Errorf("Unrecognized container type %s", containerType)
	}
	return builder(rt, containerName)
}
