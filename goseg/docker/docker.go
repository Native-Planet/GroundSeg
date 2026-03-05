package docker

import (
	"context"
	"fmt"
	"strings"

	"groundseg/docker/events"
	"groundseg/docker/orchestration"
	"groundseg/structs"
)

func validateContainerName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("container name is required")
	}
	return nil
}

func PublishUrbitTransition(ctx context.Context, event structs.UrbitTransition) error {
	if strings.TrimSpace(event.Patp) == "" {
		return fmt.Errorf("publish urbit transition requires non-empty patp")
	}
	if strings.TrimSpace(event.Type) == "" {
		return fmt.Errorf("publish urbit transition for %s requires non-empty type", event.Patp)
	}
	if err := events.DefaultEventRuntime().PublishUrbitTransition(ctx, event); err != nil {
		return fmt.Errorf("publish urbit transition for %s (%s): %w", event.Patp, event.Type, err)
	}
	return nil
}

func GetShipStatus(piers []string) (map[string]string, error) {
	if len(piers) == 0 {
		return map[string]string{}, nil
	}
	statuses, err := orchestration.GetShipStatus(piers)
	if err != nil {
		return nil, fmt.Errorf("get ship status for %d ships: %w", len(piers), err)
	}
	return statuses, nil
}

func StartContainer(containerName string, containerType string) (structs.ContainerState, error) {
	if err := validateContainerName(containerName); err != nil {
		return structs.ContainerState{}, err
	}
	if strings.TrimSpace(containerType) == "" {
		return structs.ContainerState{}, fmt.Errorf("container type is required for %s", containerName)
	}
	state, err := orchestration.StartContainer(containerName, containerType)
	if err != nil {
		return structs.ContainerState{}, fmt.Errorf("start container %s (%s): %w", containerName, containerType, err)
	}
	return state, nil
}

func StopContainerByName(containerName string) error {
	if err := validateContainerName(containerName); err != nil {
		return err
	}
	if err := orchestration.StopContainerByName(containerName); err != nil {
		return fmt.Errorf("stop container %s: %w", containerName, err)
	}
	return nil
}

func DeleteContainer(containerName string) error {
	if err := validateContainerName(containerName); err != nil {
		return err
	}
	if err := orchestration.DeleteContainer(containerName); err != nil {
		return fmt.Errorf("delete container %s: %w", containerName, err)
	}
	return nil
}

func CreateContainer(containerName string, containerType string) (structs.ContainerState, error) {
	if err := validateContainerName(containerName); err != nil {
		return structs.ContainerState{}, err
	}
	if strings.TrimSpace(containerType) == "" {
		return structs.ContainerState{}, fmt.Errorf("container type is required for %s", containerName)
	}
	state, err := orchestration.CreateContainer(containerName, containerType)
	if err != nil {
		return structs.ContainerState{}, fmt.Errorf("create container %s (%s): %w", containerName, containerType, err)
	}
	return state, nil
}

func CreateMinIOServiceAccount(patp string) (structs.MinIOServiceAccount, error) {
	if strings.TrimSpace(patp) == "" {
		return structs.MinIOServiceAccount{}, fmt.Errorf("service account creation requires non-empty patp")
	}
	account, err := orchestration.CreateMinIOServiceAccount(patp)
	if err != nil {
		return structs.MinIOServiceAccount{}, fmt.Errorf("create minio service account for %s: %w", patp, err)
	}
	return account, nil
}
