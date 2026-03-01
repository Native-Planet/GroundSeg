package network

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockernetwork "github.com/docker/docker/api/types/network"
	volumetypes "github.com/docker/docker/api/types/volume"
	"go.uber.org/zap"

	"github.com/docker/docker/client"
	"groundseg/dockerclient"
)

var dockerClientNew = dockerclient.New

func SetClientFactory(factory func(...client.Opt) (*client.Client, error)) {
	if factory == nil {
		dockerClientNew = dockerclient.New
		return
	}
	dockerClientNew = factory
}

// KillContainerUsingPort stops the first running container that is bound to the provided port.
func KillContainerUsingPort(n uint16) error {
	ctx := context.Background()
	cli, err := dockerClientNew()
	if err != nil {
		return err
	}
	defer cli.Close()

	listFilters := filters.NewArgs()
	listFilters.Add("status", "running")

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{Filters: listFilters})
	if err != nil {
		zap.L().Error(fmt.Sprintf("Unable to get container list. Failed to kill container using port %v", n))
		return err
	}

	for _, cont := range containers {
		for _, port := range cont.Ports {
			if port.PublicPort == n {
				zap.L().Debug(fmt.Sprintf("Stopping container %s to free port %v", cont.ID, n))
				options := container.StopOptions{}
				if err := cli.ContainerStop(ctx, cont.ID, options); err != nil {
					zap.L().Error(fmt.Sprintf("failed to stop container %s: %v", cont.ID, err))
					return err
				}
				return nil
			}
		}
	}
	return nil
}

// GetContainerNetwork returns the raw network mode name attached to a container.
func GetContainerNetwork(name string) (string, error) {
	cli, err := dockerClientNew()
	if err != nil {
		return "", err
	}
	defer cli.Close()

	containerJSON, err := cli.ContainerInspect(context.Background(), name)
	if err != nil {
		return "", err
	}
	if containerJSON.HostConfig.NetworkMode != "" {
		return string(containerJSON.HostConfig.NetworkMode), nil
	}
	return "", fmt.Errorf("container is not attached to any network: %v", name)
}

// CreateVolume creates a named Docker volume.
func CreateVolume(name string) error {
	cli, err := dockerClientNew()
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker client: %v : %v", name, err)
		return errmsg
	}
	defer cli.Close()

	vol, err := cli.VolumeCreate(context.Background(), volumetypes.CreateOptions{Name: name})
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker volume: %v : %v", name, err)
		return errmsg
	}
	zap.L().Info(fmt.Sprintf("Created volume: %s", vol.Name))
	return nil
}

// DeleteVolume removes a named Docker volume.
func DeleteVolume(name string) error {
	cli, err := dockerClientNew()
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker client: %v : %v", name, err)
		return errmsg
	}
	defer cli.Close()

	err = cli.VolumeRemove(context.Background(), name, true)
	if err != nil {
		errmsg := fmt.Errorf("Failed to remove docker volume: %v : %v", name, err)
		return errmsg
	}
	zap.L().Info(fmt.Sprintf("Deleted volume: %s", name))
	return nil
}

// WriteFileToVolume writes content to a file inside a docker volume.
func WriteFileToVolume(name string, file string, content string) error {
	cli, err := dockerClientNew()
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker client: %v : %v", name, err)
		return errmsg
	}
	defer cli.Close()

	vol, err := cli.VolumeInspect(context.Background(), name)
	if err != nil {
		errmsg := fmt.Errorf("Failed to inspect volume: %v : %v", name, err)
		return errmsg
	}

	fullPath := filepath.Join(vol.Mountpoint, file)
	if err = os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		errmsg := fmt.Errorf("Failed to write to volume: %v : %v", name, err)
		return errmsg
	}
	zap.L().Info(fmt.Sprintf("Successfully wrote to file: %s", fullPath))
	return nil
}

// VolumeExists reports whether the named docker volume exists.
func VolumeExists(volumeName string) (bool, error) {
	cli, err := dockerClientNew()
	if err != nil {
		return false, fmt.Errorf("Failed to create client: %v", err)
	}
	defer cli.Close()

	volumeList, err := cli.VolumeList(context.Background(), volumetypes.ListOptions{})
	if err != nil {
		return false, err
	}
	for _, volume := range volumeList.Volumes {
		if volume.Name == volumeName {
			return true, nil
		}
	}
	return false, nil
}

// AddOrGetNetwork returns an existing network id or creates a new local bridge network.
func AddOrGetNetwork(networkName string) (string, error) {
	cli, err := dockerClientNew()
	if err != nil {
		return "", fmt.Errorf("Failed to create client: %v", err)
	}
	defer cli.Close()

	networks, err := cli.NetworkList(context.Background(), dockernetwork.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("Failed to list networks: %v", err)
	}
	for _, nw := range networks {
		if nw.Name == networkName {
			return nw.ID, nil
		}
	}
	networkResponse, err := cli.NetworkCreate(context.Background(), networkName, dockernetwork.CreateOptions{
		Driver: "bridge",
		Scope:  "local",
	})
	if err != nil {
		return "", fmt.Errorf("Failed to create custom bridge network: %v", err)
	}
	return networkResponse.ID, nil
}
