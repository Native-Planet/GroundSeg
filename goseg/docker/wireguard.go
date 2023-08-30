package docker

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"goseg/config"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	// "golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func LoadWireguard() error {
	config.Logger.Info("Loading Startram Wireguard container")
	confPath := filepath.Join(config.BasePath, "settings", "wireguard.json")
	_, err := os.Open(confPath)
	if err != nil {
		// create a default container conf if it doesn't exist
		err = config.CreateDefaultWGConf()
		if err != nil {
			// error if we can't create it
			return err
		}
	}
	// create wg0.conf or update it
	err = WriteWgConf()
	if err != nil {
		return err
	}
	config.Logger.Info("Running Wireguard")
	info, err := StartContainer("wireguard", "wireguard")
	if err != nil {
		config.Logger.Error(fmt.Sprintf("Error starting wireguard: %v", err))
		return err
	}
	config.UpdateContainerState("wireguard", info)
	return nil
}

// wireguard container config builder
func wgContainerConf() (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	// construct the container metadata from version server info
	containerInfo, err := GetLatestContainerInfo("wireguard")
	if err != nil {
		return containerConfig, hostConfig, err
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	// construct the container config struct
	containerConfig = container.Config{
		Image:      desiredImage,
		Entrypoint: []string{"/bin/bash"},
		Tty:        true,
		OpenStdin:  true,
	}
	// always on wg nw
	hostConfig = container.HostConfig{
		NetworkMode: "container:wireguard",
	}
	return containerConfig, hostConfig, nil
}

// wg0.conf builder
func buildWgConf() (string, error) {
	confB64 := config.StartramConfig.Conf
	confBytes, err := base64.StdEncoding.DecodeString(confB64)
	if err != nil {
		return "", fmt.Errorf("Failed to decode remote WG base64: %v", err)
	}
	conf := string(confBytes)
	configData := config.Conf()
	res := strings.Replace(conf, "privkey", configData.Privkey, -1)
	return res, nil
}

// write latest conf
func WriteWgConf() error {
	newConf, err := buildWgConf()
	if err != nil {
		return err
	}
	filePath := filepath.Join(config.DockerDir, "settings", "wireguard", "_data", "wg0.conf")
	existingConf, err := ioutil.ReadFile(filePath)
	if err != nil {
		// assume it doesn't exist, so write the current config
		config.Logger.Info("Creating WG config")
		return writeWgConfToFile(filePath, newConf)
	}
	if string(existingConf) != newConf {
		// If they differ, overwrite
		config.Logger.Info("Updating WG config")
		return writeWgConfToFile(filePath, newConf)
	}
	return nil
}

// either write directly or create volumes
func writeWgConfToFile(filePath string, content string) error {
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
		err = copyFileToVolume(filePath, "/etc/wireguard/", "wireguard")
		// otherwise create the volume
		if err != nil {
			return fmt.Errorf("Failed to copy WG config file to volume: %v", err)
		}
	}
	return nil
}

// write wg conf to volume
func copyFileToVolume(filePath string, targetPath string, volumeName string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}
	containerInfo, err := GetLatestContainerInfo("wireguard")
	if err != nil {
		return err
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	// temp container to mount
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: desiredImage,
	}, &container.HostConfig{
		Binds: []string{volumeName + ":" + targetPath},
	}, nil, nil, "wg_writer")
	if err != nil {
		return err
	}
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}
	file, err := os.Open(filepath.Join(filePath))
	if err != nil {
		return fmt.Errorf("failed to open wg0 file: %v", err)
	}
	defer file.Close()
	// Copy the file to the volume via the temporary container
	err = cli.CopyToContainer(ctx, resp.ID, targetPath, file, types.CopyToContainerOptions{})
	if err != nil {
		return err
	}
	// remove temporary container
	if err := StopContainerByName("wg_writer"); err != nil {
		return err
	}
	if err := cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
		return err
	}
	defer func() {
		if removeErr := cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true}); removeErr != nil {
			config.Logger.Error("Failed to remove temporary container: ", removeErr)
		}
	}()
	return nil
}
