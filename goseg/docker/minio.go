package docker

import (
	"fmt"
	"goseg/config"
	"goseg/logger"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

func LoadMC() error {
	conf := config.Conf()
	if conf.WgRegistered {
		logger.Logger.Info("Loading MC container")
		confPath := filepath.Join(config.BasePath, "settings", "mc.json")
		_, err := os.Open(confPath)
		if err != nil {
			// create a default if it doesn't exist
			err = config.CreateDefaultMcConf()
			if err != nil {
				// error if we can't create it
				errmsg := fmt.Sprintf("Unable to create MC config! %v", err)
				logger.Logger.Error(errmsg)
			}
		}
		logger.Logger.Info("Running MC")
		info, err := StartContainer("mc", "miniomc")
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Error starting MC: %v", err))
			return err
		}
		config.UpdateContainerState("mc", info)
	}
	return nil
}

// iterate through each ship and create a minio
// version stuff is offloaded to version server struct
func LoadMinIOs() error {
	conf := config.Conf()
	if conf.WgRegistered {
		logger.Logger.Info("Loading MinIO containers")
		for _, pier := range conf.Piers {
			label := "minio_" + pier
			info, err := StartContainer(label, "minio")
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("Error starting %s Minio: %v", pier, err))
				return err
			}
			config.UpdateContainerState(label, info)
		}
	}
	return nil
}

// minio container config builder
func minioContainerConf(containerName string) (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	shipName := strings.Split(containerName, "_")[1]
	err := config.LoadUrbitConfig(shipName)
	if err != nil {
		errmsg := fmt.Errorf("Error loading %s config: %v", shipName, err)
		return containerConfig, hostConfig, errmsg
	}
	shipConf := config.UrbitConf(shipName)
	// construct the container metadata from version server info
	containerInfo, err := GetLatestContainerInfo("minio")
	if err != nil {
		return containerConfig, hostConfig, err
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	command := fmt.Sprintf("server /data --console-address \":%v\" --address \":%v\"", shipConf.WgConsolePort, shipConf.WgS3Port)
	tempPassword := "11111111"
	environment := []string{
		fmt.Sprintf("MINIO_ROOT_USER=%s", shipName),
		fmt.Sprintf("MINIO_ROOT_PASSWORD=%s", tempPassword), //shipConf.MinioPassword),
		fmt.Sprintf("MINIO_DOMAIN=s3.%s", shipConf.WgURL),
		fmt.Sprintf("MINIO_SERVER_URL=https://s3.%s", shipConf.WgURL),
	}
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: containerName,
			Target: "/data",
		},
	}
	containerConfig = container.Config{
		Image: desiredImage,
		Cmd:   []string{command},
		Env:   environment,
	}
	// always on wg nw
	hostConfig = container.HostConfig{
		NetworkMode: "container:wireguard",
		Mounts:      mounts,
	}
	return containerConfig, hostConfig, nil
}

// miniomc container config builder
func mcContainerConf() (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	// construct the container metadata from version server info
	containerInfo, err := GetLatestContainerInfo("miniomc")
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
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			status, err := GetContainerRunningStatus("wireguard")
			if err != nil {
				return containerConfig, hostConfig, err
			}
			if strings.Contains(status, "Up") {
				break
			}
		}
	}
	return containerConfig, hostConfig, nil
}
