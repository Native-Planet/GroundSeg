package docker

import (
	"groundseg/docker/events"
	"groundseg/docker/orchestration"
	"groundseg/structs"
)

func PublishUrbitTransition(event structs.UrbitTransition) {
	events.PublishUrbitTransition(event)
}

func GetShipStatus(piers []string) (map[string]string, error) {
	return orchestration.GetShipStatus(piers)
}

func StartContainer(containerName string, containerType string) (structs.ContainerState, error) {
	return orchestration.StartContainer(containerName, containerType)
}

func StopContainerByName(containerName string) error {
	return orchestration.StopContainerByName(containerName)
}

func DeleteContainer(containerName string) error {
	return orchestration.DeleteContainer(containerName)
}

func CreateContainer(containerName string, containerType string) (structs.ContainerState, error) {
	return orchestration.CreateContainer(containerName, containerType)
}

func CreateMinIOServiceAccount(patp string) (structs.MinIOServiceAccount, error) {
	return orchestration.CreateMinIOServiceAccount(patp)
}
