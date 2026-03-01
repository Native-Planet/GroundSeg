package orchestration

import (
	"fmt"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"groundseg/config"
	"groundseg/docker/lifecycle"
	"groundseg/docker/network"
	"groundseg/docker/registry"
	"groundseg/dockerclient"
	"groundseg/structs"
)

var (
	VolumeDir       = config.DockerDir
	ContainerStats  = make(map[string]structs.ContainerStats)
	dockerClientNew = dockerclient.New

	containerConfigResolver = func(containerName string, containerType string) (container.Config, container.HostConfig, error) {
		return container.Config{}, container.HostConfig{}, fmt.Errorf("container config resolver not configured")
	}
	subpackageOnce sync.Once
)

func SetDockerClientNew(newFactory func(opts ...client.Opt) (*client.Client, error)) {
	dockerClientNew = newFactory
}

func SetContainerConfigResolver(
	resolver func(containerName string, containerType string) (container.Config, container.HostConfig, error),
) {
	containerConfigResolver = resolver
}

func ConfigureSubpackages() {
	clientFactory := func(opts ...client.Opt) (*client.Client, error) {
		return dockerClientNew(opts...)
	}

	lifecycle.SetClientFactory(clientFactory)
	lifecycle.SetContainerConfigResolver(containerConfigResolver)
	lifecycle.SetImageInfoLookup(registry.GetLatestContainerInfo)
	lifecycle.SetImagePuller(registry.PullImageIfNotExist)

	network.SetClientFactory(clientFactory)
	registry.SetClientFactory(clientFactory)
}

func Initialize() error {
	subpackageOnce.Do(ConfigureSubpackages)
	return lifecycle.Initialize()
}

func GetShipStatus(patps []string) (map[string]string, error) {
	return lifecycle.GetShipStatus(patps)
}

func GetContainerImageTag(containerName string) (string, error) {
	return lifecycle.GetContainerImageTag(containerName)
}

func GetContainerRunningStatus(containerName string) (string, error) {
	return lifecycle.GetContainerRunningStatus(containerName)
}

func GetContainerNetwork(name string) (string, error) {
	return network.GetContainerNetwork(name)
}

func CreateVolume(name string) error {
	return network.CreateVolume(name)
}

func DeleteVolume(name string) error {
	return network.DeleteVolume(name)
}

func DeleteContainer(name string) error {
	return lifecycle.DeleteContainer(name)
}

func WriteFileToVolume(name string, file string, content string) error {
	return network.WriteFileToVolume(name, file, content)
}

func StartContainer(containerName string, containerType string) (structs.ContainerState, error) {
	return lifecycle.StartContainer(containerName, containerType)
}

func CreateContainer(containerName string, containerType string) (structs.ContainerState, error) {
	return lifecycle.CreateContainer(containerName, containerType)
}

func GetLatestContainerInfo(containerType string) (map[string]string, error) {
	return registry.GetLatestContainerInfo(containerType)
}

func PullImageIfNotExist(desiredImage string, imageInfo map[string]string) (bool, error) {
	return registry.PullImageIfNotExist(desiredImage, imageInfo)
}

func StopContainerByName(containerName string) error {
	return lifecycle.StopContainerByName(containerName)
}

func ExecDockerCommand(containerName string, cmd []string) (string, int, error) {
	return lifecycle.ExecDockerCommand(containerName, cmd)
}

func RestartContainer(name string) error {
	return lifecycle.RestartContainer(name)
}

func FindContainer(containerName string) (*container.Summary, error) {
	c, err := lifecycle.FindContainer(containerName)
	if c == nil || err != nil {
		return nil, err
	}
	return c, nil
}

func DockerPoller() {
	lifecycle.DockerPoller()
}

func Contains(slice []string, str string) bool {
	return lifecycle.Contains(slice, str)
}

func VolumeExists(volumeName string) (bool, error) {
	return network.VolumeExists(volumeName)
}

func AddOrGetNetwork(networkName string) (string, error) {
	return network.AddOrGetNetwork(networkName)
}
