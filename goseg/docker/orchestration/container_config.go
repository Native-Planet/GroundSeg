package orchestration

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
)

func ContainerConfigForType(containerName string, containerType string) (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	var err error
	switch containerType {
	case "vere":
		containerConfig, hostConfig, err = urbitContainerConf(containerName)
	case "netdata":
		containerConfig, hostConfig, err = netdataContainerConf()
	case "minio":
		containerConfig, hostConfig, err = minioContainerConf(containerName)
	case "miniomc":
		containerConfig, hostConfig, err = mcContainerConf()
	case "wireguard":
		containerConfig, hostConfig, err = wgContainerConf()
	case "llama-api":
		containerConfig, hostConfig, err = llamaApiContainerConf()
	default:
		return containerConfig, hostConfig, fmt.Errorf("Unrecognized container type %s", containerType)
	}
	if err != nil {
		return containerConfig, hostConfig, err
	}
	return containerConfig, hostConfig, nil
}
