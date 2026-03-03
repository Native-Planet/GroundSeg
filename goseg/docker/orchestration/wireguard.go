package orchestration

import (
	"encoding/base64"
	"errors"
	"fmt"
	"groundseg/docker/orchestration/container"
	"groundseg/docker/orchestration/internal/artifactwriter"
	"groundseg/internal/seams"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	dockerc "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"go.uber.org/zap"
)

func (rt WireguardRuntime) LoadWireguard() error {
	zap.L().Info("Loading Startram Wireguard container")
	if rt.BasePathFn == nil {
		return seams.MissingRuntimeDependency("wireguard base path getter", "")
	}
	confPath := filepath.Join(rt.BasePathFn(), "settings", "wireguard.json")
	return container.RunContainerWithRuntime(container.ContainerRuntimePlan{
		ContainerName:         "wireguard",
		ContainerImage:        "wireguard",
		ConfigPath:            confPath,
		OpenConfigFn:          rt.OpenFn,
		CreateDefaultConfigFn: rt.CreateDefaultWGConfFn,
		WriteConfigFn:         rt.WriteWgConf,
		StartContainerFn:      rt.StartContainerFn,
		UpdateContainerState:  rt.UpdateContainerStateFn,
	})
}

func (rt WireguardRuntime) wgContainerConf() (dockerc.Config, dockerc.HostConfig, error) {
	if rt.GetLatestContainerInfoFn == nil {
		return dockerc.Config{}, dockerc.HostConfig{}, seams.MissingRuntimeDependency("wireguard image selector", "")
	}
	containerInfo, err := rt.GetLatestContainerInfoFn("wireguard")
	if err != nil {
		return dockerc.Config{}, dockerc.HostConfig{}, fmt.Errorf("unable to load latest wireguard image metadata: %w", err)
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	containerConfig := dockerc.Config{
		Image:     desiredImage,
		Hostname:  "wireguard",
		Tty:       true,
		OpenStdin: true,
	}
	if rt.GetWgConfFn == nil {
		return containerConfig, dockerc.HostConfig{}, seams.MissingRuntimeDependency("wireguard config callback", "")
	}
	wgConfig, err := rt.GetWgConfFn()
	if err != nil {
		return containerConfig, dockerc.HostConfig{}, fmt.Errorf("unable to get wireguard config: %w", err)
	}
	hostConfig := dockerc.HostConfig{
		Mounts: []mount.Mount{
			{Type: mount.TypeVolume, Source: "wireguard", Target: "/config"},
		},
		CapAdd: wgConfig.CapAdd,
		Sysctls: map[string]string{
			"net.ipv4.conf.all.src_valid_mark": strconv.Itoa(wgConfig.Sysctls.NetIpv4ConfAllSrcValidMark),
		},
	}
	return containerConfig, hostConfig, nil
}

func (rt WireguardRuntime) buildWgConf() (string, error) {
	if rt.GetWgConfBlobFn == nil || rt.GetWgPrivkeyFn == nil {
		return "", seams.MissingRuntimeDependency("wireguard key material runtime", "")
	}
	confB64, err := rt.GetWgConfBlobFn()
	if err != nil {
		return "", fmt.Errorf("unable to read startram wireguard config: %w", err)
	}
	confBytes, err := base64.StdEncoding.DecodeString(confB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode remote WG base64: %w", err)
	}
	return strings.Replace(string(confBytes), "privkey", rt.GetWgPrivkeyFn(), -1), nil
}

func (rt WireguardRuntime) WriteWgConf() error {
	if rt.WriteWgConfFn != nil {
		return rt.WriteWgConfFn()
	}
	newConf, err := rt.buildWgConf()
	if err != nil {
		return fmt.Errorf("failed to build wireguard configuration: %w", err)
	}
	if rt.DockerDirFn == nil {
		return seams.MissingRuntimeDependency("wireguard docker directory callback", "")
	}
	if rt.ReadFileFn == nil {
		return seams.MissingRuntimeDependency("wireguard config file reader", "")
	}
	filePath := filepath.Join(rt.DockerDirFn(), "wireguard", "_data", "wg0.conf")
	existingConf, err := rt.ReadFileFn(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			zap.L().Info("Creating WG config")
			if writeErr := artifactwriter.Write(wireguardConfigWriteArtifactOptions(rt, filePath, newConf)); writeErr != nil {
				return fmt.Errorf("unable to create wireguard config: %w", writeErr)
			}
			return nil
		}
		return fmt.Errorf("failed to read existing wireguard config: %w", err)
	}
	if string(existingConf) == newConf {
		return nil
	}
	zap.L().Info("Updating WG config")
	if err := artifactwriter.Write(wireguardConfigWriteArtifactOptions(rt, filePath, newConf)); err != nil {
		return fmt.Errorf("failed to update wireguard config: %w", err)
	}
	return nil
}

func (rt WireguardRuntime) writeWgConfToFile(filePath string, content string) error {
	return artifactwriter.Write(wireguardConfigWriteArtifactOptions(rt, filePath, content))
}

func wireguardConfigWriteArtifactOptions(rt WireguardRuntime, filePath string, content string) artifactwriter.WriteConfig {
	return artifactwriter.WriteConfig{
		FilePath:            filePath,
		Content:             content,
		FileMode:            0644,
		DirectoryMode:       0755,
		WriteFileFn:         rt.WriteFileFn,
		MkdirAllFn:          rt.MkdirAllFn,
		CopyToVolumeFn:      rt.CopyFileToVolumeFn,
		TargetPath:          "/etc/wireguard/",
		VolumeName:          "wireguard",
		WriterContainerName: "wg_writer",
		SelectImageFn: func() (string, error) {
			if rt.GetLatestContainerImageFn == nil {
				return "", seams.MissingRuntimeDependency("wireguard image selector", "")
			}
			return rt.GetLatestContainerImageFn("wireguard")
		},
		CopyErrorPrefix: "Failed to copy WG config file to volume",
		EnsureVolumesFn: artifactwriter.NewVolumeInitializationPlan(artifactwriter.VolumeOps{
			VolumeExistsFn: rt.VolumeExistsFn,
			CreateVolumeFn: rt.CreateVolumeFn,
		}, "wireguard").EnsureVolumes,
	}
}

func (rt WireguardRuntime) copyWGFileToVolume(filePath string, targetPath string, volumeName string) error {
	if rt.CopyFileToVolumeFn == nil {
		return seams.MissingRuntimeDependency("missing copy-to-volume runtime", "wireguard")
	}
	return rt.CopyFileToVolumeFn(
		filePath,
		targetPath,
		volumeName,
		"wg_writer",
		func() (string, error) {
			if rt.GetLatestContainerImageFn == nil {
				return "", seams.MissingRuntimeDependency("missing wireguard image selector", "")
			}
			return rt.GetLatestContainerImageFn("wireguard")
		},
	)
}
