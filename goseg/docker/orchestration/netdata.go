package orchestration

import (
	"fmt"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
)

func LoadNetdata() error {
	return loadNetdata(newNetdataRuntime())
}

func loadNetdata(rt netdataRuntime) error {
	zap.L().Info("Loading NetData container")
	confPath := filepath.Join(rt.BasePath(), "settings", "netdata.json")
	_, err := rt.osOpen(confPath)
	if err != nil {
		// create a default if it doesn't exist
		err = rt.createDefaultNetdataConf()
		if err != nil {
			return fmt.Errorf("unable to create netdata config: %v", err)
		}
	}
	if rt.writeNDConf != nil {
		err = rt.writeNDConf(rt)
	} else {
		err = writeNDConfWithRuntime(rt)
	}
	if err != nil {
		return err
	}
	zap.L().Info("Running NetData")
	info, err := rt.startContainer("netdata", "netdata")
	if err != nil {
		zap.L().Error(fmt.Sprintf("Error starting NetData: %v", err))
		return err
	}
	rt.updateContainerState("netdata", info)
	return nil
}

// netdata container config builder
func netdataContainerConf() (container.Config, container.HostConfig, error) {
	return netdataContainerConfWithRuntime(newNetdataRuntime())
}

func netdataContainerConfWithRuntime(rt netdataRuntime) (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	// construct the container metadata from version server info
	containerInfo, err := rt.getLatestContainerInfo("netdata")
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
	return writeNDConfWithRuntime(newNetdataRuntime())
}

func writeNDConfWithRuntime(rt netdataRuntime) error {
	newConf := "[plugins]\n     apps = no\n"
	filePath := filepath.Join(rt.DockerDir(), "netdataconfig", "_data", "netdata.conf")
	existingConf, err := rt.readFile(filePath)
	if err != nil {
		// assume it doesn't exist, so write the current config
		zap.L().Info("Creating ND config")
		return writeNDConfToFileWithRuntime(rt, filePath, newConf)
	}
	if string(existingConf) != newConf {
		// If they differ, overwrite
		zap.L().Info("Writing ND config")
		return writeNDConfToFileWithRuntime(rt, filePath, newConf)
	}
	return nil
}

// either write directly or create volumes
func writeNDConfToFile(filePath string, content string) error {
	return writeNDConfToFileWithRuntime(newNetdataRuntime(), filePath, content)
}

func writeNDConfToFileWithRuntime(rt netdataRuntime, filePath string, content string) error {
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
		if rt.copyNDFileToVolume != nil {
			err = rt.copyNDFileToVolume(rt, filePath, "/etc/netdata/", "netdata")
		} else {
			err = copyNDFileToVolumeWithRuntime(rt, filePath, "/etc/netdata/", "netdata")
		}
		// otherwise create the volume
		if err != nil {
			return fmt.Errorf("Failed to copy ND config file to volume: %v", err)
		}
	}
	return nil
}

// write ND conf to volume
func copyNDFileToVolume(filePath string, targetPath string, volumeName string) error {
	return copyNDFileToVolumeWithRuntime(newNetdataRuntime(), filePath, targetPath, volumeName)
}

func copyNDFileToVolumeWithRuntime(rt netdataRuntime, filePath string, targetPath string, volumeName string) error {
	return rt.copyFileToVolumeWithTempContainer(
		filePath,
		targetPath,
		volumeName,
		"nd_writer",
		func() (string, error) {
			return rt.latestContainerImage("netdata")
		},
	)
}
