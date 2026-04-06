package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"groundseg/config"
	"groundseg/dockerclient"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"go.uber.org/zap"
	// "golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func LoadWireguard() error {
	zap.L().Info("Loading Startram Wireguard container")
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
	zap.L().Info("Running Wireguard")
	info, err := StartContainer("wireguard", "wireguard")
	if err != nil {
		zap.L().Error(fmt.Sprintf("Error starting wireguard: %v", err))
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
		Image:     desiredImage,
		Hostname:  "wireguard",
		Tty:       true,
		OpenStdin: true,
	}
	// Define volume mount
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: "wireguard",
			Target: "/config",
		},
	}
	wgConfig, err := config.GetWgConf()
	if err != nil {
		return containerConfig, hostConfig, err
	}
	hostConfig = container.HostConfig{
		Mounts: mounts,
		CapAdd: wgConfig.CapAdd,
		Sysctls: map[string]string{
			"net.ipv4.conf.all.src_valid_mark": strconv.Itoa(wgConfig.Sysctls.NetIpv4ConfAllSrcValidMark),
		},
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

func wireguardHostConfigPath() string {
	dockerDir := filepath.Clean(config.DockerDir)
	if dockerDir == "." || dockerDir == "/" {
		return ""
	}
	return filepath.Join(dockerDir, "wireguard", "_data", "wg0.conf")
}

// write latest conf
func WriteWgConf() error {
	newConf, err := buildWgConf()
	if err != nil {
		return err
	}
	filePath := wireguardHostConfigPath()
	if filePath != "" {
		existingConf, err := ioutil.ReadFile(filePath)
		switch {
		case err == nil && string(existingConf) == newConf:
			return nil
		case err == nil:
			zap.L().Info("Updating WG config")
			if err := writeWgConfToFile(filePath, newConf); err == nil {
				return nil
			} else {
				zap.L().Warn(fmt.Sprintf("Direct WG config write failed, falling back to Docker volume copy: %v", err))
			}
		case os.IsNotExist(err):
			zap.L().Info("Creating WG config")
			if err := writeWgConfToFile(filePath, newConf); err == nil {
				return nil
			} else {
				zap.L().Warn(fmt.Sprintf("Direct WG config create failed, falling back to Docker volume copy: %v", err))
			}
		default:
			zap.L().Warn(fmt.Sprintf("Couldn't read WG config from host path %s, falling back to Docker volume copy: %v", filePath, err))
		}
	} else {
		zap.L().Warn("Docker volume path could not be resolved on host, writing WG config via Docker volume copy")
	}
	return copyWGFileToVolume("wg0.conf", newConf, "/config", "wireguard")
}

// write directly to the docker volume mount when the host path is known
func writeWgConfToFile(filePath string, content string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, []byte(content), 0644)
}

func wgConfTarStream(fileName string, content string) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	body := []byte(content)
	header := &tar.Header{
		Name: fileName,
		Mode: 0644,
		Size: int64(len(body)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return nil, err
	}
	if _, err := tw.Write(body); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return &buf, nil
}

// write wg conf to volume
func copyWGFileToVolume(fileName string, content string, targetPath string, volumeName string) error {
	ctx := context.Background()
	cli, err := dockerclient.New()
	if err != nil {
		return err
	}
	defer cli.Close()
	containerInfo, err := GetLatestContainerInfo("wireguard")
	if err != nil {
		return err
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	tarStream, err := wgConfTarStream(fileName, content)
	if err != nil {
		return err
	}
	// temp container to mount
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: desiredImage,
		Cmd:   []string{"sh", "-c", "sleep 30"},
	}, &container.HostConfig{
		Binds: []string{volumeName + ":" + targetPath},
	}, nil, nil, "wg_writer")
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true}); removeErr != nil {
			zap.L().Error(fmt.Sprintf("Failed to remove temporary container: %v", removeErr))
		}
	}()
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}
	// Copy the file to the volume via the temporary container
	err = cli.CopyToContainer(ctx, resp.ID, targetPath, tarStream, container.CopyToContainerOptions{})
	if err != nil {
		return err
	}
	return nil
}
