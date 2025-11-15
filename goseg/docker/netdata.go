package docker

import (
	"context"
	"fmt"
	"groundseg/config"
	"groundseg/dockerclient"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
)

func LoadNetdata() error {
	zap.L().Info("Loading NetData container")
	confPath := filepath.Join(config.BasePath, "settings", "netdata.json")
	_, err := os.Open(confPath)
	if err != nil {
		// create a default if it doesn't exist
		err = config.CreateDefaultNetdataConf()
		if err != nil {
			// panic if we can't create it
			errmsg := fmt.Sprintf("Unable to create NetData config! %v", err)
			zap.L().Error(errmsg)
			panic(errmsg)
		}
	}
	err = WriteNDConf()
	if err != nil {
		return err
	}
	zap.L().Info("Running NetData")
	info, err := StartContainer("netdata", "netdata")
	if err != nil {
		zap.L().Error(fmt.Sprintf("Error starting NetData: %v", err))
		return err
	}
	config.UpdateContainerState("netdata", info)
	return nil
}

// netdata container config builder
func netdataContainerConf() (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	// construct the container metadata from version server info
	containerInfo, err := GetLatestContainerInfo("netdata")
	if err != nil {
		return containerConfig, hostConfig, err
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	// construct the container config struct
	containerConfig = container.Config{
		Image:        desiredImage,
		ExposedPorts: nat.PortSet{"19999/tcp": struct{}{}},
		Volumes: map[string]struct{}{
			"/etc/netdata":         {},
			"/var/lib/netdata":     {},
			"/var/cache/netdata":   {},
			"/host/etc/passwd":     {},
			"/host/etc/group":      {},
			"/host/proc":           {},
			"/host/sys":            {},
			"/host/etc/os-release": {},
		},
	}
	hostConfig = container.HostConfig{
		CapAdd: []string{"SYS_PTRACE"},
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		Resources: container.Resources{
			NanoCPUs: 1e8,               // 1cpu
			Memory:   200 * 1024 * 1024, // 200mb
		},
		SecurityOpt: []string{"apparmor=unconfined"},
		PortBindings: nat.PortMap{
			"19999/tcp": []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: "19999"},
			},
		},
		Binds: []string{
			"netdataconfig:/etc/netdata",
			"netdatalib:/var/lib/netdata",
			"netdatacache:/var/cache/netdata",
			"/etc/passwd:/host/etc/passwd:ro",
			"/etc/group:/host/etc/group:ro",
			"/proc:/host/proc:ro",
			"/sys:/host/sys:ro",
			"/etc/os-release:/host/etc/os-release:ro",
		},
	}
	return containerConfig, hostConfig, nil
}

// write edited conf
func WriteNDConf() error {
	newConf := "[plugins]\n     apps = no\n"
	filePath := filepath.Join(config.DockerDir, "netdataconfig", "_data", "netdata.conf")
	existingConf, err := ioutil.ReadFile(filePath)
	if err != nil {
		// assume it doesn't exist, so write the current config
		zap.L().Info("Creating ND config")
		return writeNDConfToFile(filePath, newConf)
	}
	if string(existingConf) != newConf {
		// If they differ, overwrite
		zap.L().Info("Writing ND config")
		return writeNDConfToFile(filePath, newConf)
	}
	return nil
}

// either write directly or create volumes
func writeNDConfToFile(filePath string, content string) error {
	// try writing
	err := ioutil.WriteFile(filePath, []byte(content), 0644)
	if err == nil {
		return nil
	}
	// ensure the directory structure exists
	dir := filepath.Dir(filePath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// try writing again
	err = ioutil.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		err = copyNDFileToVolume(filePath, "/etc/netdata/", "netdata")
		// otherwise create the volume
		if err != nil {
			return fmt.Errorf("Failed to copy ND config file to volume: %v", err)
		}
	}
	return nil
}

// write ND conf to volume
func copyNDFileToVolume(filePath string, targetPath string, volumeName string) error {
	ctx := context.Background()
	cli, err := dockerclient.New()
	if err != nil {
		return err
	}
	defer cli.Close()
	containerInfo, err := GetLatestContainerInfo("netdata")
	if err != nil {
		return err
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	// temp container to mount
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: desiredImage,
	}, &container.HostConfig{
		Binds: []string{volumeName + ":" + targetPath},
	}, nil, nil, "nd_writer")
	if err != nil {
		return err
	}
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}
	file, err := os.Open(filepath.Join(filePath))
	if err != nil {
		return fmt.Errorf("failed to open netdata.config file: %v", err)
	}
	defer file.Close()
	// Copy the file to the volume via the temporary container
	err = cli.CopyToContainer(ctx, resp.ID, targetPath, file, types.CopyToContainerOptions{})
	if err != nil {
		return err
	}
	// remove temporary container
	if err := StopContainerByName("nd_writer"); err != nil {
		return err
	}
	if err := cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
		return err
	}
	defer func() {
		if removeErr := cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true}); removeErr != nil {
			zap.L().Error(fmt.Sprintf("Failed to remove temporary container: ", removeErr))
		}
	}()
	return nil
}
