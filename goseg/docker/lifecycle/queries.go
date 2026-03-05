package lifecycle

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
)

func (runtime *Runtime) GetShipStatus(patps []string) (statuses map[string]string, err error) {
	statuses = make(map[string]string)
	cli, err := runtime.dockerClientNew()
	if err != nil {
		errmsg := fmt.Errorf("unable to create docker client: %w", err)
		return statuses, errmsg
	}
	defer closeRuntimeDockerClient(cli, "ship status", &err)
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		errmsg := fmt.Errorf("failed to list containers: %w", err)
		return statuses, errmsg
	}

	statusIndex := NewContainerStatusIndex(containers)
	return ResolveStatuses(statusIndex, patps), nil
}

// GetContainerImageTag returns the image tag for a container name if the container exists.
func (runtime *Runtime) GetContainerImageTag(containerName string) (string, error) {
	ctx := context.Background()
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return "", fmt.Errorf("unable to create docker client: %w", err)
	}
	defer closeRuntimeDockerClient(cli, "container image tag lookup", &err)

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}
	for _, cont := range containers {
		for _, name := range cont.Names {
			if strings.TrimPrefix(name, "/") == containerName {
				if tag := imageTagFromReference(cont.Image); tag != "" {
					return tag, nil
				}
				return "latest", nil
			}
		}
	}
	return "", fmt.Errorf("no exact match found for container %s", containerName)
}

func imageTagFromReference(image string) string {
	imageWithoutDigest := strings.SplitN(image, "@", 2)[0]
	lastColon := strings.LastIndex(imageWithoutDigest, ":")
	if lastColon == -1 {
		return ""
	}
	lastSlash := strings.LastIndex(imageWithoutDigest, "/")
	if lastSlash > lastColon {
		return ""
	}
	return imageWithoutDigest[lastColon+1:]
}

// GetContainerRunningStatus returns status for a container by exact name.
func (runtime *Runtime) GetContainerRunningStatus(containerName string) (status string, err error) {
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return status, fmt.Errorf("unable to create docker client: %w", err)
	}
	defer closeRuntimeDockerClient(cli, "container running status", &err)
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		return status, fmt.Errorf("failed to list containers: %w", err)
	}
	for _, cont := range containers {
		for _, name := range cont.Names {
			if name == "/"+containerName {
				return cont.Status, nil
			}
		}
	}
	return status, fmt.Errorf("unable to get container running status for %s", containerName)
}
