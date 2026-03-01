package docker

import (
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"groundseg/docker/orchestration"
	"groundseg/dockerclient"
	"groundseg/structs"
)

var (
	VolumeDir       = orchestration.VolumeDir
	ContainerStats  = orchestration.ContainerStats
	dockerClientNew = func(opts ...client.Opt) (*client.Client, error) {
		return dockerclient.New(opts...)
	}
)

func init() {
	// Configure the orchestration layer to route container config through the root package,
	// which owns the concrete container-type resolver helpers.
	orchestration.SetContainerConfigResolver(orchestration.ContainerConfigForType)
}

func Initialize() error {
	// Keep legacy test seam behavior by using the package-level variable.
	orchestration.SetDockerClientNew(dockerClientNew)
	return orchestration.Initialize()
}

func GetShipStatus(patps []string) (map[string]string, error) {
	return orchestration.GetShipStatus(patps)
}

func GetContainerImageTag(containerName string) (string, error) {
	return orchestration.GetContainerImageTag(containerName)
}

func GetContainerRunningStatus(containerName string) (string, error) {
	return orchestration.GetContainerRunningStatus(containerName)
}

func GetContainerNetwork(name string) (string, error) {
	return orchestration.GetContainerNetwork(name)
}

func CreateVolume(name string) error {
	return orchestration.CreateVolume(name)
}

func DeleteVolume(name string) error {
	return orchestration.DeleteVolume(name)
}

func DeleteContainer(name string) error {
	return orchestration.DeleteContainer(name)
}

func WriteFileToVolume(name string, file string, content string) error {
	return orchestration.WriteFileToVolume(name, file, content)
}

func StartContainer(containerName string, containerType string) (structs.ContainerState, error) {
	return orchestration.StartContainer(containerName, containerType)
}

func CreateContainer(containerName string, containerType string) (structs.ContainerState, error) {
	return orchestration.CreateContainer(containerName, containerType)
}

func GetLatestContainerInfo(containerType string) (map[string]string, error) {
	return orchestration.GetLatestContainerInfo(containerType)
}

func PullImageIfNotExist(desiredImage string, imageInfo map[string]string) (bool, error) {
	return orchestration.PullImageIfNotExist(desiredImage, imageInfo)
}

func StopContainerByName(containerName string) error {
	return orchestration.StopContainerByName(containerName)
}

func ExecDockerCommand(containerName string, cmd []string) (string, int, error) {
	return orchestration.ExecDockerCommand(containerName, cmd)
}

func RestartContainer(name string) error {
	return orchestration.RestartContainer(name)
}

func FindContainer(containerName string) (*container.Summary, error) {
	return orchestration.FindContainer(containerName)
}

func LoadLlama() error {
	return orchestration.LoadLlama()
}

func LoadMC() error {
	return orchestration.LoadMC()
}

func LoadMinIOs() error {
	return orchestration.LoadMinIOs()
}

func CreateMinIOServiceAccount(patp string) (structs.MinIOServiceAccount, error) {
	return orchestration.CreateMinIOServiceAccount(patp)
}

func LoadNetdata() error {
	return orchestration.LoadNetdata()
}

func WriteNDConf() error {
	return orchestration.WriteNDConf()
}

func LoadUrbits() error {
	return orchestration.LoadUrbits()
}

func LoadWireguard() error {
	return orchestration.LoadWireguard()
}

func WriteWgConf() error {
	return orchestration.WriteWgConf()
}

func WaitForShipExit(patp string, timeout time.Duration) error {
	return orchestration.WaitForShipExit(patp, timeout)
}

func GetContainerStats(name string) structs.ContainerStats {
	return orchestration.GetContainerStats(name)
}

func ForceUpdateContainerStats(name string) structs.ContainerStats {
	return orchestration.ForceUpdateContainerStats(name)
}

func contains(slice []string, str string) bool {
	return orchestration.Contains(slice, str)
}

func DockerPoller() {
	orchestration.DockerPoller()
}

func VolumeExists(volumeName string) (bool, error) {
	return orchestration.VolumeExists(volumeName)
}

func AddOrGetNetwork(networkName string) (string, error) {
	return orchestration.AddOrGetNetwork(networkName)
}
