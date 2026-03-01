package orchestration

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"go.uber.org/zap"
	// "golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func LoadWireguard() error {
	return loadWireguard(newWireguardRuntime())
}

func loadWireguard(rt wireguardRuntime) error {
	zap.L().Info("Loading Startram Wireguard container")
	confPath := filepath.Join(rt.BasePath(), "settings", "wireguard.json")
	_, err := rt.osOpen(confPath)
	if err != nil {
		// create a default container conf if it doesn't exist
		err = rt.createDefaultWGConf()
		if err != nil {
			// error if we can't create it
			return err
		}
	}
	// create wg0.conf or update it
	if rt.writeWgConf != nil {
		err = rt.writeWgConf(rt)
	} else {
		err = writeWgConfWithRuntime(rt)
	}
	if err != nil {
		return err
	}
	zap.L().Info("Running Wireguard")
	info, err := rt.startContainer("wireguard", "wireguard")
	if err != nil {
		zap.L().Error(fmt.Sprintf("Error starting wireguard: %v", err))
		return err
	}
	rt.updateContainerState("wireguard", info)
	return nil
}

// wireguard container config builder
func wgContainerConf() (container.Config, container.HostConfig, error) {
	return wgContainerConfWithRuntime(newWireguardRuntime())
}

func wgContainerConfWithRuntime(rt wireguardRuntime) (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	// construct the container metadata from version server info
	containerInfo, err := rt.getLatestContainerInfo("wireguard")
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
	wgConfig, err := rt.getWgConf()
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
	return buildWgConfWithRuntime(newWireguardRuntime())
}

func buildWgConfWithRuntime(rt wireguardRuntime) (string, error) {
	confB64 := rt.getStartramConfig().Conf
	confBytes, err := base64.StdEncoding.DecodeString(confB64)
	if err != nil {
		return "", fmt.Errorf("Failed to decode remote WG base64: %v", err)
	}
	conf := string(confBytes)
	configData := rt.conf()
	res := strings.Replace(conf, "privkey", configData.Privkey, -1)
	return res, nil
}

// write latest conf
func WriteWgConf() error {
	return writeWgConfWithRuntime(newWireguardRuntime())
}

func writeWgConfWithRuntime(rt wireguardRuntime) error {
	newConf, err := buildWgConfWithRuntime(rt)
	if err != nil {
		return err
	}
	filePath := filepath.Join(rt.DockerDir(), "wireguard", "_data", "wg0.conf")
	existingConf, err := rt.readFile(filePath)
	if err != nil {
		// assume it doesn't exist, so write the current config
		zap.L().Info("Creating WG config")
		return writeWgConfToFileWithRuntime(rt, filePath, newConf)
	}
	if string(existingConf) != newConf {
		// If they differ, overwrite
		zap.L().Info("Updating WG config")
		return writeWgConfToFileWithRuntime(rt, filePath, newConf)
	}
	return nil
}

// either write directly or create volumes
func writeWgConfToFile(filePath string, content string) error {
	return writeWgConfToFileWithRuntime(newWireguardRuntime(), filePath, content)
}

func writeWgConfToFileWithRuntime(rt wireguardRuntime, filePath string, content string) error {
	// try writing
	err := rt.writeFile(filePath, []byte(content), 0644)
	if err == nil {
		return nil
	}
	// ensure the directory structure exists
	dir := filepath.Dir(filePath)
	if err = rt.mkdirAll(dir, 0755); err != nil {
		return err
	}
	// try writing again
	err = rt.writeFile(filePath, []byte(content), 0644)
	if err != nil {
		if rt.copyWGFileToVolume != nil {
			err = rt.copyWGFileToVolume(rt, filePath, "/etc/wireguard/", "wireguard")
		} else {
			err = copyWGFileToVolumeWithRuntime(rt, filePath, "/etc/wireguard/", "wireguard")
		}
		// otherwise create the volume
		if err != nil {
			return fmt.Errorf("Failed to copy WG config file to volume: %v", err)
		}
	}
	return nil
}

// write wg conf to volume
func copyWGFileToVolume(filePath string, targetPath string, volumeName string) error {
	return copyWGFileToVolumeWithRuntime(newWireguardRuntime(), filePath, targetPath, volumeName)
}

func copyWGFileToVolumeWithRuntime(rt wireguardRuntime, filePath string, targetPath string, volumeName string) error {
	return rt.copyFileToVolumeWithTempContainer(
		filePath,
		targetPath,
		volumeName,
		"wg_writer",
		func() (string, error) {
			return rt.latestContainerImage("wireguard")
		},
	)
}
