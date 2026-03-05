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
	return getOrchestrationRuntime().initialize()
}

func (runtime *orchestrationRuntime) initialize() error {
	return runtime.lifecycleRuntime.Initialize()
}

func GetShipStatus(patps []string) (map[string]string, error) {
	return getOrchestrationRuntime().getShipStatus(patps)
}

func (runtime *orchestrationRuntime) getShipStatus(patps []string) (map[string]string, error) {
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
	return runtime.lifecycleRuntime.GetShipStatus(patps)
}

func GetContainerImageTag(containerName string) (string, error) {
	return getOrchestrationRuntime().getContainerImageTag(containerName)
}

func (runtime *orchestrationRuntime) getContainerImageTag(containerName string) (string, error) {
	if err := validateContainerName(containerName); err != nil {
		return "", err
	}
	return runtime.lifecycleRuntime.GetContainerImageTag(containerName)
}

func GetContainerRunningStatus(containerName string) (string, error) {
	return getOrchestrationRuntime().getContainerRunningStatus(containerName)
}

func (runtime *orchestrationRuntime) getContainerRunningStatus(containerName string) (string, error) {
	if err := validateContainerName(containerName); err != nil {
		return "", err
	}
	return runtime.lifecycleRuntime.GetContainerRunningStatus(containerName)
}

func DeleteContainer(name string) error {
	return getOrchestrationRuntime().deleteContainer(name)
}

func (runtime *orchestrationRuntime) deleteContainer(name string) error {
	if err := validateContainerName(name); err != nil {
		return err
	}
	return runtime.lifecycleRuntime.DeleteContainer(name)
}

func StartContainer(containerName string, containerType string) (structs.ContainerState, error) {
	return getOrchestrationRuntime().startContainer(containerName, containerType)
}

func (runtime *orchestrationRuntime) startContainer(containerName string, containerType string) (structs.ContainerState, error) {
	if err := validateContainerName(containerName); err != nil {
		return structs.ContainerState{}, err
	}
	if containerType == "" {
		return structs.ContainerState{}, fmt.Errorf("container type is required")
	}
	return runtime.lifecycleRuntime.StartContainer(containerName, containerType)
}

func CreateContainer(containerName string, containerType string) (structs.ContainerState, error) {
	return getOrchestrationRuntime().createContainer(containerName, containerType)
}

func (runtime *orchestrationRuntime) createContainer(containerName string, containerType string) (structs.ContainerState, error) {
	if err := validateContainerName(containerName); err != nil {
		return structs.ContainerState{}, err
	}
	if containerType == "" {
		return structs.ContainerState{}, fmt.Errorf("container type is required")
	}
	return runtime.lifecycleRuntime.CreateContainer(containerName, containerType)
}

func GetLatestContainerInfo(containerType string) (registry.ImageDescriptor, error) {
	return getOrchestrationRuntime().getLatestContainerInfo(containerType)
}

func (runtime *orchestrationRuntime) getLatestContainerInfo(containerType string) (registry.ImageDescriptor, error) {
	return registry.GetLatestContainerInfo(containerType)
}

func PullImageIfNotExist(desiredImage string, imageInfo registry.ImageDescriptor) (bool, error) {
	return getOrchestrationRuntime().pullImageIfNotExist(desiredImage, imageInfo)
}

func (runtime *orchestrationRuntime) pullImageIfNotExist(desiredImage string, imageInfo registry.ImageDescriptor) (bool, error) {
	if desiredImage == "" {
		return false, fmt.Errorf("desired image is required")
	}
	return registry.PullImageIfNotExist(desiredImage, imageInfo)
}

func StopContainerByName(containerName string) error {
	return getOrchestrationRuntime().stopContainerByName(containerName)
}

func (runtime *orchestrationRuntime) stopContainerByName(containerName string) error {
	if err := validateContainerName(containerName); err != nil {
		return err
	}
	return runtime.lifecycleRuntime.StopContainerByName(containerName)
}

func ExecDockerCommand(containerName string, cmd []string) (string, int, error) {
	return getOrchestrationRuntime().execDockerCommand(containerName, cmd)
}

func (runtime *orchestrationRuntime) execDockerCommand(containerName string, cmd []string) (string, int, error) {
	if err := validateContainerName(containerName); err != nil {
		return "", -1, err
	}
	if len(cmd) == 0 {
		return "", -1, fmt.Errorf("docker command is empty")
	}
	return runtime.lifecycleRuntime.ExecDockerCommand(containerName, cmd)
}

func RestartContainer(name string) error {
	return getOrchestrationRuntime().restartContainer(name)
}

func (runtime *orchestrationRuntime) restartContainer(name string) error {
	if err := validateContainerName(name); err != nil {
		return err
	}
	return runtime.lifecycleRuntime.RestartContainer(name)
}

func FindContainer(containerName string) (*container.Summary, error) {
	return getOrchestrationRuntime().findContainer(containerName)
}

func (runtime *orchestrationRuntime) findContainer(containerName string) (*container.Summary, error) {
	if err := validateContainerName(containerName); err != nil {
		return nil, err
	}
	return runtime.lifecycleRuntime.FindContainer(containerName)
}

func CreateVolume(volumeName string) error {
	return getOrchestrationRuntime().createVolume(volumeName)
}

func (runtime *orchestrationRuntime) createVolume(volumeName string) error {
	if err := validateContainerName(volumeName); err != nil {
		return err
	}
	return network.NewNetworkRuntime().CreateVolume(volumeName)
}

func DeleteVolume(volumeName string) error {
	return getOrchestrationRuntime().deleteVolume(volumeName)
}

func (runtime *orchestrationRuntime) deleteVolume(volumeName string) error {
	if err := validateContainerName(volumeName); err != nil {
		return err
	}
	return network.NewNetworkRuntime().DeleteVolume(volumeName)
}

func DockerPoller() {
	getOrchestrationRuntime().dockerPoller()
}

func (runtime *orchestrationRuntime) dockerPoller() {
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
