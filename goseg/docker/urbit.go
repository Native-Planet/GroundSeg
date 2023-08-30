package docker

// start up urbits

import (
	"fmt"
	"goseg/config"
	"goseg/defaults"
	"goseg/structs"
	"io/ioutil"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
)

// load existing urbits from config json
func LoadUrbits() error {
	config.Logger.Info("Loading Urbit ships")
	// Loop through pier list
	conf := config.Conf()
	for _, pier := range conf.Piers {
		config.Logger.Info(fmt.Sprintf("Loading pier %s", pier))
		// load json into struct
		err := config.LoadUrbitConfig(pier)
		if err != nil {
			config.Logger.Error(fmt.Sprintf("Error loading %s config: %v", pier, err))
			continue
		}
		shipConf := config.UrbitConf(pier)
		// don't bootstrap if it's busted
		if shipConf.BootStatus != "noboot" {
			info, err := StartContainer(pier, "vere")
			if err != nil {
				config.Logger.Error(fmt.Sprintf("Error starting %s: %v", pier, err))
				continue
			}
			config.UpdateContainerState(pier, info)
		}
	}
	return nil
}

// urbit container config builder
func urbitContainerConf(containerName string) (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	var scriptContent string
	// construct the container metadata from version server info
	containerInfo, err := GetLatestContainerInfo("vere")
	if err != nil {
		return containerConfig, hostConfig, err
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	// reload urbit conf from disk
	err = config.LoadUrbitConfig(containerName)
	if err != nil {
		errmsg := fmt.Errorf("Error loading %s config: %v", containerName, err)
		return containerConfig, hostConfig, errmsg
	}
	// put in memory
	shipConf := config.UrbitConf(containerName)
	// todo: this BootStatus doesnt actually have anythin to do with pack and meld right now
	act := shipConf.BootStatus
	// get the correct startup script based on BootStatus val
	switch act {
	case "boot":
		scriptContent = defaults.StartScript
	case "pack":
		scriptContent = defaults.PackScript
	case "meld":
		scriptContent = defaults.MeldScript
	case "prep":
		scriptContent = defaults.PrepScript
	case "noboot":
		return containerConfig, hostConfig, fmt.Errorf("%s marked noboot!", containerName)
	default:
		return containerConfig, hostConfig, fmt.Errorf("Unknown action: %s", act)
	}
	// reset ship status to boot for next time
	if act != "boot" {
		updateUrbitConf := shipConf
		updateUrbitConf.BootStatus = "boot"
		var newConfig map[string]structs.UrbitDocker
		newConfig[containerName] = updateUrbitConf
		err = config.UpdateUrbitConfig(newConfig)
		if err != nil {
			config.Logger.Warn("Unable to reset %s boot script!", containerName)
		}
	}
	// write the script
	scriptPath := filepath.Join(config.DockerDir, containerName, "_data", containerName, "start_urbit.sh")
	err = ioutil.WriteFile(scriptPath, []byte(scriptContent), 0755) // make the script executable
	if err != nil {
		return containerConfig, hostConfig, fmt.Errorf("Failed to write script: %v", err)
	}
	// gather boot option values
	shipName := shipConf.PierName
	loomValue := string(shipConf.LoomSize)
	dirnameValue := shipConf.PierName
	var devMode string
	if shipConf.DevMode == true {
		devMode = "True"
	} else {
		devMode = "False"
	}
	// construct the network configuration based on conf val
	var httpPort string
	var amesPort string
	var network string
	var portMap nat.PortMap
	if shipConf.Network == "wireguard" {
		httpPort = string(shipConf.WgHTTPPort)
		amesPort = string(shipConf.WgAmesPort)
		network = "container:wireguard"
	} else {
		httpPort = string(shipConf.HTTPPort)
		amesPort = string(shipConf.AmesPort)
		network = "default"
		httpPortStr := nat.Port(fmt.Sprintf(httpPort + "/tcp"))
		amesPortStr := nat.Port(fmt.Sprintf(amesPort + "/udp"))
		portMap = nat.PortMap{
			httpPortStr: []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: httpPort},
			},
			amesPortStr: []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: amesPort},
			},
		}
	}
	// finally construct the container config structs
	containerConfig = container.Config{
		Image:      desiredImage,
		Entrypoint: []string{scriptPath, shipName, "--loom=" + loomValue, "--dirname=" + dirnameValue, "--dev-mode=" + devMode, "--http-port=" + httpPort, "--port=" + amesPort},
	}
	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: shipName,
			Target: "/urbit",
		},
	}
	hostConfig = container.HostConfig{
		NetworkMode:  container.NetworkMode(network),
		Mounts:       mounts,
		PortBindings: portMap,
	}
	return containerConfig, hostConfig, nil
}