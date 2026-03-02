package network

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockernetwork "github.com/docker/docker/api/types/network"
	volumetypes "github.com/docker/docker/api/types/volume"
	"go.uber.org/zap"

	"github.com/docker/docker/client"
	"groundseg/dockerclient"
)

type NetworkRuntime struct {
	DockerClientNewFn func(...client.Opt) (*client.Client, error)
	OperationTimeout  time.Duration
}

func NewNetworkRuntime() NetworkRuntime {
	return NetworkRuntime{
		DockerClientNewFn: dockerclient.New,
		OperationTimeout:  30 * time.Second,
	}
}

func (runtime NetworkRuntime) withDefaults() NetworkRuntime {
	if runtime.DockerClientNewFn == nil {
		runtime.DockerClientNewFn = dockerclient.New
	}
	if runtime.OperationTimeout <= 0 {
		runtime.OperationTimeout = 30 * time.Second
	}
	return runtime
}

func (runtime NetworkRuntime) KillContainerUsingPort(port uint16) error {
	runtime = runtime.withDefaults()
	cli, err := runtime.DockerClientNewFn()
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), runtime.OperationTimeout)
	defer cancel()

	listFilters := filters.NewArgs()
	listFilters.Add("status", "running")

	containers, err := cli.ContainerList(ctx, container.ListOptions{Filters: listFilters})
	if err != nil {
		zap.L().Error(fmt.Sprintf("Unable to get container list. Failed to kill container using port %v", port))
		return fmt.Errorf("failed to list running containers while finding a handler for port %d: %w", port, err)
	}
	for _, cont := range containers {
		for _, containerPort := range cont.Ports {
			if containerPort.PublicPort == port {
				zap.L().Debug(fmt.Sprintf("Stopping container %s to free port %d", cont.ID, containerPort.PublicPort))
				if err := cli.ContainerStop(ctx, cont.ID, container.StopOptions{}); err != nil {
					zap.L().Error(fmt.Sprintf("failed to stop container %s: %v", cont.ID, err))
					return fmt.Errorf("failed to stop container %s while releasing port %d: %w", cont.ID, containerPort.PublicPort, err)
				}
				return nil
			}
		}
	}
	return nil
}

func (runtime NetworkRuntime) GetContainerNetwork(name string) (string, error) {
	runtime = runtime.withDefaults()
	cli, err := runtime.DockerClientNewFn()
	if err != nil {
		return "", fmt.Errorf("failed to create docker client for network lookup of %s: %w", name, err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), runtime.OperationTimeout)
	defer cancel()
	containerJSON, err := cli.ContainerInspect(ctx, name)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container %s: %w", name, err)
	}
	if containerJSON.HostConfig.NetworkMode != "" {
		return string(containerJSON.HostConfig.NetworkMode), nil
	}
	return "", fmt.Errorf("container is not attached to any network: %v", name)
}

func (runtime NetworkRuntime) CreateVolume(name string) error {
	runtime = runtime.withDefaults()
	cli, err := runtime.DockerClientNewFn()
	if err != nil {
		return fmt.Errorf("failed to create docker client for volume %q: %w", name, err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), runtime.OperationTimeout)
	defer cancel()
	vol, err := cli.VolumeCreate(ctx, volumetypes.CreateOptions{Name: name})
	if err != nil {
		return fmt.Errorf("failed to create docker volume %q: %w", name, err)
	}
	zap.L().Info(fmt.Sprintf("Created volume: %s", vol.Name))
	return nil
}

func (runtime NetworkRuntime) DeleteVolume(name string) error {
	runtime = runtime.withDefaults()
	cli, err := runtime.DockerClientNewFn()
	if err != nil {
		return fmt.Errorf("failed to create docker client for volume %q: %w", name, err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), runtime.OperationTimeout)
	defer cancel()
	if err := cli.VolumeRemove(ctx, name, true); err != nil {
		return fmt.Errorf("failed to remove docker volume %q: %w", name, err)
	}
	zap.L().Info(fmt.Sprintf("Deleted volume: %s", name))
	return nil
}

func (runtime NetworkRuntime) WriteFileToVolume(name string, file string, content string) error {
	runtime = runtime.withDefaults()
	cli, err := runtime.DockerClientNewFn()
	if err != nil {
		return fmt.Errorf("failed to create docker client for volume %q: %w", name, err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), runtime.OperationTimeout)
	defer cancel()
	vol, err := cli.VolumeInspect(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to inspect volume %q: %w", name, err)
	}

	fullPath := filepath.Join(vol.Mountpoint, file)
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %q for volume %q: %w", file, name, err)
	}
	zap.L().Info(fmt.Sprintf("Successfully wrote to file: %s", fullPath))
	return nil
}

func (runtime NetworkRuntime) VolumeExists(volumeName string) (bool, error) {
	runtime = runtime.withDefaults()
	cli, err := runtime.DockerClientNewFn()
	if err != nil {
		return false, fmt.Errorf("failed to create client for volume check: %w", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), runtime.OperationTimeout)
	defer cancel()
	volumeList, err := cli.VolumeList(ctx, volumetypes.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to list docker volumes: %w", err)
	}
	for _, volume := range volumeList.Volumes {
		if volume.Name == volumeName {
			return true, nil
		}
	}
	return false, nil
}

func (runtime NetworkRuntime) AddOrGetNetwork(networkName string) (string, error) {
	runtime = runtime.withDefaults()
	cli, err := runtime.DockerClientNewFn()
	if err != nil {
		return "", fmt.Errorf("failed to create client for network lookup: %w", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), runtime.OperationTimeout)
	defer cancel()
	networks, err := cli.NetworkList(ctx, dockernetwork.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list networks: %w", err)
	}
	for _, nw := range networks {
		if nw.Name == networkName {
			return nw.ID, nil
		}
	}
	networkResponse, err := cli.NetworkCreate(ctx, networkName, dockernetwork.CreateOptions{
		Driver: "bridge",
		Scope:  "local",
	})
	if err != nil {
		return "", fmt.Errorf("failed to create custom bridge network %q: %w", networkName, err)
	}
	return networkResponse.ID, nil
}
