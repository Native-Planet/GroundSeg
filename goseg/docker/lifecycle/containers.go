package lifecycle

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/errdefs"
	"go.uber.org/zap"
	"groundseg/structs"
)

func (runtime *Runtime) StartContainer(containerName string, containerType string) (structs.ContainerState, error) {
	zap.L().Debug(fmt.Sprintf("StartContainer issued for %v", containerName))
	var containerState structs.ContainerState
	if err := runtime.cleanupMinIOContainerForStart(containerName, containerType); err != nil {
		return containerState, fmt.Errorf("cleanup container %s before start: %w", containerName, err)
	}
	plan, err := runtime.containerPlanFor(containerName, containerType)
	if err != nil {
		return containerState, fmt.Errorf("build container plan for %s/%s: %w", containerName, containerType, err)
	}
	ctx := context.Background()
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return containerState, fmt.Errorf("create docker client for %s: %w", containerName, err)
	}
	defer cli.Close()
	if err := ensureRunningContainer(runtime, ctx, cli, plan); err != nil {
		return containerState, fmt.Errorf("start container %s: %w", containerName, err)
	}
	containerDetails, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return containerState, fmt.Errorf("failed to inspect container %s: %w", containerName, err)
	}
	containerState = containerStateFromInspect(plan, "running", containerDetails)
	return containerState, err
}

func (runtime *Runtime) CreateContainer(containerName string, containerType string) (structs.ContainerState, error) {
	var containerState structs.ContainerState
	if err := runtime.cleanupMinIOContainerForStart(containerName, containerType); err != nil {
		return containerState, fmt.Errorf("cleanup container %s before create: %w", containerName, err)
	}
	plan, err := runtime.containerPlanFor(containerName, containerType)
	if err != nil {
		return containerState, fmt.Errorf("build container plan for %s/%s: %w", containerName, containerType, err)
	}
	ctx := context.Background()
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return containerState, fmt.Errorf("create docker client for %s: %w", containerName, err)
	}
	defer cli.Close()
	if err := ensureCreatedContainer(ctx, cli, plan); err != nil {
		return containerState, fmt.Errorf("create container %s: %w", containerName, err)
	}
	containerDetails, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return containerState, fmt.Errorf("failed to inspect container %s: %w", containerName, err)
	}
	containerState = containerStateFromInspect(plan, "stopped", containerDetails)
	return containerState, err
}

func (runtime *Runtime) StopContainerByName(containerName string) error {
	ctx := context.Background()
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return fmt.Errorf("unable to create docker client: %w", err)
	}
	defer cli.Close()
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}
	for _, cont := range containers {
		for _, name := range cont.Names {
			if name == "/"+containerName {
				options := container.StopOptions{}
				if err := cli.ContainerStop(ctx, cont.ID, options); err != nil {
					return fmt.Errorf("failed to stop container %s: %w", containerName, err)
				}
				zap.L().Info(fmt.Sprintf("Successfully stopped container %s\n", containerName))
				return nil
			}
		}
	}
	return errdefs.NotFound(fmt.Errorf("container %s not found", containerName))
}

func (runtime *Runtime) DeleteContainer(name string) error {
	cli, err := runtime.dockerClientNew()
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker client %s: %w", name, err)
		return errmsg
	}
	defer cli.Close()
	if err := cli.ContainerRemove(context.Background(), name, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("Failed to delete docker container %s: %w", name, err)
	}
	zap.L().Info(fmt.Sprintf("Deleted Container: %s", name))
	return nil
}

func (runtime *Runtime) cleanupMinIOContainerForStart(containerName string, containerType string) error {
	if containerType != "minio" {
		return nil
	}
	existingContainer, err := runtime.FindContainer(containerName)
	if err != nil {
		if isContainerLookupNotFound(err) {
			return nil
		}
		return fmt.Errorf("find minio container %s for cleanup: %w", containerName, err)
	}
	if existingContainer == nil {
		return nil
	}
	return runtime.DeleteContainer(containerName)
}

func isContainerLookupNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errdefs.IsNotFound(err)
}

func (runtime *Runtime) RestartContainer(name string) error {
	ctx := context.Background()
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return fmt.Errorf("Couldn't create client: %w", err)
	}
	defer cli.Close()

	containerID, err := GetContainerIDByName(ctx, cli, name)
	if err != nil {
		return fmt.Errorf("Couldn't get ID for %s: %w", name, err)
	}
	timeout := 30
	stopOptions := container.StopOptions{Timeout: &timeout}
	if err := cli.ContainerRestart(ctx, containerID, stopOptions); err != nil {
		return fmt.Errorf("Couldn't restart container %s: %w", name, err)
	}
	return nil
}

func (runtime *Runtime) FindContainer(containerName string) (*container.Summary, error) {
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.TrimPrefix(name, "/") == containerName {
				return &container, nil
			}
		}
	}
	return nil, errdefs.NotFound(fmt.Errorf("container %s not found", containerName))
}
