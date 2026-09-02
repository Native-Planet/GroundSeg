package docker

import (
	"context"
	"fmt"
	"groundseg/config"
	"groundseg/dockerclient"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	volumetypes "github.com/docker/docker/api/types/volume"
	"go.uber.org/zap"
	// "golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var wireguardConfigMu sync.Mutex

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
	// Create wg0.conf or update it. If StarTram is temporarily unavailable,
	// SyncWireguardConfig preserves and validates the last known-good file.
	_, err = SyncWireguardConfig()
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
	conf, err := config.DecodeStartramWireguardConfig(confB64)
	if err != nil {
		return "", err
	}
	configData := config.Conf()
	res := strings.ReplaceAll(conf, "privkey", configData.Privkey)
	if err := config.ValidateWireguardConfig(res, false); err != nil {
		return "", fmt.Errorf("invalid rendered WireGuard configuration: %w", err)
	}
	return res, nil
}

func wireguardConfigPathFromMountpoint(mountpoint string) (string, error) {
	mountpoint = filepath.Clean(strings.TrimSpace(mountpoint))
	if mountpoint == "." || mountpoint == "/" {
		return "", fmt.Errorf("Docker returned an invalid WireGuard volume mountpoint %q", mountpoint)
	}
	return filepath.Join(mountpoint, "wg0.conf"), nil
}

func wireguardHostConfigPath() (string, error) {
	cli, err := dockerclient.New()
	if err != nil {
		return "", fmt.Errorf("create Docker client: %w", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	volume, err := cli.VolumeInspect(ctx, "wireguard")
	if err != nil {
		volume, err = cli.VolumeCreate(ctx, volumetypes.CreateOptions{Name: "wireguard"})
		if err != nil {
			return "", fmt.Errorf("inspect or create WireGuard volume: %w", err)
		}
	}
	return wireguardConfigPathFromMountpoint(volume.Mountpoint)
}

// SyncWireguardConfig writes a validated fresh StarTram configuration. When no
// fresh configuration is available, it preserves a valid last-known-good file
// instead of truncating it.
func SyncWireguardConfig() (bool, error) {
	wireguardConfigMu.Lock()
	defer wireguardConfigMu.Unlock()

	filePath, err := wireguardHostConfigPath()
	if err != nil {
		return false, err
	}
	existingConf, readErr := os.ReadFile(filePath)
	newConf, buildErr := buildWgConf()
	if buildErr != nil {
		if readErr == nil {
			if validationErr := config.ValidateWireguardConfig(string(existingConf), false); validationErr == nil {
				zap.L().Warn(fmt.Sprintf("Fresh WireGuard config unavailable; preserving last known-good config: %v", buildErr))
				return false, nil
			}
		}
		if readErr != nil {
			return false, fmt.Errorf("fresh WireGuard config unavailable (%v) and existing config could not be read: %w", buildErr, readErr)
		}
		return false, fmt.Errorf("fresh WireGuard config unavailable and existing config is invalid: %w", buildErr)
	}
	if readErr == nil && string(existingConf) == newConf {
		return false, nil
	}
	if readErr != nil && !os.IsNotExist(readErr) {
		return false, fmt.Errorf("read existing WireGuard config: %w", readErr)
	}

	if readErr == nil {
		zap.L().Info("Updating WG config")
	} else {
		zap.L().Info("Creating WG config")
	}
	if err := writeWgConfToFile(filePath, newConf); err != nil {
		return false, err
	}
	return true, nil
}

// WriteWgConf keeps the original API for callers that only need an error.
func WriteWgConf() error {
	_, err := SyncWireguardConfig()
	return err
}

// ApplyRetrievedWireguardConfig repairs the on-disk file after a successful
// retrieve and reloads a running tunnel only when its contents changed.
func ApplyRetrievedWireguardConfig() error {
	changed, err := SyncWireguardConfig()
	if err != nil || !changed {
		return err
	}
	status, err := GetContainerRunningStatus("wireguard")
	if err != nil || !strings.Contains(status, "Up") {
		return nil
	}
	zap.L().Info("Reloading Wireguard after applying an updated configuration")
	return RestartContainer("wireguard")
}

// RestartWireguard ensures the persisted configuration is valid before a
// restart so an empty file is never blindly reloaded.
func RestartWireguard() error {
	if _, err := SyncWireguardConfig(); err != nil {
		return err
	}
	return RestartContainer("wireguard")
}

// write directly to the Docker volume using an atomic, durable replacement
func writeWgConfToFile(filePath string, content string) error {
	if err := config.ValidateWireguardConfig(content, false); err != nil {
		return fmt.Errorf("refusing to write invalid WireGuard configuration: %w", err)
	}
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return config.WriteFileDurably(filePath, []byte(content), 0644)
}
