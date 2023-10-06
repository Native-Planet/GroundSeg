package docker

import (
	"fmt"
	"goseg/config"
	"goseg/logger"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

func LoadNetdata() error {
	logger.Logger.Info("Loading NetData container")
	confPath := filepath.Join(config.BasePath, "settings", "netdata.json")
	_, err := os.Open(confPath)
	if err != nil {
		// create a default if it doesn't exist
		err = config.CreateDefaultNetdataConf()
		if err != nil {
			// panic if we can't create it
			errmsg := fmt.Sprintf("Unable to create NetData config! %v", err)
			logger.Logger.Error(errmsg)
			panic(errmsg)
		}
	}
	logger.Logger.Info("Running NetData")
	info, err := StartContainer("netdata", "netdata")
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Error starting NetData: %v", err))
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
			NanoCPUs: 1e8,
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
