package orchestration

import (
	"fmt"
	"groundseg/config"
	"groundseg/docker/lifecycle"
	"groundseg/docker/network"
	"groundseg/docker/registry"
	"groundseg/dockerclient"
	"groundseg/structs"
	"sync"

	"github.com/docker/docker/api/types/container"
)

var (
	VolumeDir      = config.DockerDir()
	ContainerStats = make(map[string]structs.ContainerStats)
)

var (
	defaultOrchestrationRuntime     *orchestrationRuntime
	defaultOrchestrationRuntimeOnce sync.Once
)

type orchestrationRuntime struct {
	lifecycleRuntime *lifecycle.Runtime

	containerConfigResolver func(string, string) (container.Config, container.HostConfig, error)
}

func newOrchestrationRuntime() *orchestrationRuntime {
	clientFactory := dockerclient.New
	rt := &orchestrationRuntime{
		containerConfigResolver: ContainerConfigForTypeString,
		lifecycleRuntime: lifecycle.NewRuntime(
			lifecycle.WithDockerClientFactory(clientFactory),
			lifecycle.WithContainerConfigResolver(func(containerName, containerType string) (container.Config, container.HostConfig, error) {
				return ContainerConfigForTypeString(containerName, containerType)
			}),
			lifecycle.WithImageInfoLookup(registry.GetLatestContainerInfo),
			lifecycle.WithImagePuller(registry.PullImageIfNotExist),
		),
	}
	return rt
}

func getOrchestrationRuntime() *orchestrationRuntime {
	defaultOrchestrationRuntimeOnce.Do(func() {
		defaultOrchestrationRuntime = newOrchestrationRuntime()
	})
	return defaultOrchestrationRuntime
}

func Initialize() error {
	return getOrchestrationRuntime().Initialize()
}

func (runtime *orchestrationRuntime) Initialize() error {
	return runtime.lifecycleRuntime.Initialize()
}

func GetShipStatus(patps []string) (map[string]string, error) {
	if len(patps) == 0 {
		return map[string]string{}, nil
	}
	filteredPatps := make([]string, 0, len(patps))
	for _, patp := range patps {
		if patp != "" {
			filteredPatps = append(filteredPatps, patp)
		}
	}
	if len(filteredPatps) == 0 {
		return map[string]string{}, nil
	}
	return getOrchestrationRuntime().GetShipStatus(filteredPatps)
}

func (runtime *orchestrationRuntime) GetShipStatus(patps []string) (map[string]string, error) {
	return runtime.lifecycleRuntime.GetShipStatus(patps)
}

func GetContainerImageTag(containerName string) (string, error) {
	if err := validateContainerName(containerName); err != nil {
		return "", err
	}
	return getOrchestrationRuntime().GetContainerImageTag(containerName)
}

func (runtime *orchestrationRuntime) GetContainerImageTag(containerName string) (string, error) {
	return runtime.lifecycleRuntime.GetContainerImageTag(containerName)
}

func GetContainerRunningStatus(containerName string) (string, error) {
	if err := validateContainerName(containerName); err != nil {
		return "", err
	}
	return getOrchestrationRuntime().GetContainerRunningStatus(containerName)
}

func (runtime *orchestrationRuntime) GetContainerRunningStatus(containerName string) (string, error) {
	return runtime.lifecycleRuntime.GetContainerRunningStatus(containerName)
}

func DeleteContainer(name string) error {
	if err := validateContainerName(name); err != nil {
		return err
	}
	return getOrchestrationRuntime().DeleteContainer(name)
}

func (runtime *orchestrationRuntime) DeleteContainer(name string) error {
	return runtime.lifecycleRuntime.DeleteContainer(name)
}

func StartContainer(containerName string, containerType string) (structs.ContainerState, error) {
	if err := validateContainerName(containerName); err != nil {
		return structs.ContainerState{}, err
	}
	if containerType == "" {
		return structs.ContainerState{}, fmt.Errorf("container type is required")
	}
	return getOrchestrationRuntime().StartContainer(containerName, containerType)
}

func (runtime *orchestrationRuntime) StartContainer(containerName string, containerType string) (structs.ContainerState, error) {
	return runtime.lifecycleRuntime.StartContainer(containerName, containerType)
}

func CreateContainer(containerName string, containerType string) (structs.ContainerState, error) {
	if err := validateContainerName(containerName); err != nil {
		return structs.ContainerState{}, err
	}
	if containerType == "" {
		return structs.ContainerState{}, fmt.Errorf("container type is required")
	}
	return getOrchestrationRuntime().CreateContainer(containerName, containerType)
}

func (runtime *orchestrationRuntime) CreateContainer(containerName string, containerType string) (structs.ContainerState, error) {
	return runtime.lifecycleRuntime.CreateContainer(containerName, containerType)
}

func GetLatestContainerInfo(containerType string) (map[string]string, error) {
	return getOrchestrationRuntime().GetLatestContainerInfo(containerType)
}

func (runtime *orchestrationRuntime) GetLatestContainerInfo(containerType string) (map[string]string, error) {
	return registry.GetLatestContainerInfo(containerType)
}

func PullImageIfNotExist(desiredImage string, imageInfo map[string]string) (bool, error) {
	if desiredImage == "" {
		return false, fmt.Errorf("desired image is required")
	}
	return getOrchestrationRuntime().PullImageIfNotExist(desiredImage, imageInfo)
}

func (runtime *orchestrationRuntime) PullImageIfNotExist(desiredImage string, imageInfo map[string]string) (bool, error) {
	return registry.PullImageIfNotExist(desiredImage, imageInfo)
}

func StopContainerByName(containerName string) error {
	if err := validateContainerName(containerName); err != nil {
		return err
	}
	return getOrchestrationRuntime().StopContainerByName(containerName)
}

func (runtime *orchestrationRuntime) StopContainerByName(containerName string) error {
	return runtime.lifecycleRuntime.StopContainerByName(containerName)
}

func ExecDockerCommand(containerName string, cmd []string) (string, int, error) {
	if err := validateContainerName(containerName); err != nil {
		return "", -1, err
	}
	if len(cmd) == 0 {
		return "", -1, fmt.Errorf("docker command is empty")
	}
	return getOrchestrationRuntime().ExecDockerCommand(containerName, cmd)
}

func (runtime *orchestrationRuntime) ExecDockerCommand(containerName string, cmd []string) (string, int, error) {
	return runtime.lifecycleRuntime.ExecDockerCommand(containerName, cmd)
}

func RestartContainer(name string) error {
	if err := validateContainerName(name); err != nil {
		return err
	}
	return getOrchestrationRuntime().RestartContainer(name)
}

func (runtime *orchestrationRuntime) RestartContainer(name string) error {
	return runtime.lifecycleRuntime.RestartContainer(name)
}

func FindContainer(containerName string) (*container.Summary, error) {
	if err := validateContainerName(containerName); err != nil {
		return nil, err
	}
	return getOrchestrationRuntime().FindContainer(containerName)
}

func (runtime *orchestrationRuntime) FindContainer(containerName string) (*container.Summary, error) {
	return runtime.lifecycleRuntime.FindContainer(containerName)
}

func CreateVolume(volumeName string) error {
	if err := validateContainerName(volumeName); err != nil {
		return err
	}
	return network.NewNetworkRuntime().CreateVolume(volumeName)
}

func (runtime *orchestrationRuntime) CreateVolume(volumeName string) error {
	return network.NewNetworkRuntime().CreateVolume(volumeName)
}

func DeleteVolume(volumeName string) error {
	if err := validateContainerName(volumeName); err != nil {
		return err
	}
	return network.NewNetworkRuntime().DeleteVolume(volumeName)
}

func (runtime *orchestrationRuntime) DeleteVolume(volumeName string) error {
	return network.NewNetworkRuntime().DeleteVolume(volumeName)
}

func DockerPoller() {
	getOrchestrationRuntime().DockerPoller()
}

func (runtime *orchestrationRuntime) DockerPoller() {
	runtime.lifecycleRuntime.DockerPoller()
}

func Contains(slice []string, str string) bool {
	return lifecycle.Contains(slice, str)
}

func validateContainerName(containerName string) error {
	if containerName == "" {
		return fmt.Errorf("container name is required")
	}
	return nil
}
