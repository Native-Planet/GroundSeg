package docker

// start up urbits

import (
	"fmt"
	"groundseg/config"
	"groundseg/defaults"
	"groundseg/logger"
	"groundseg/structs"
	"io/ioutil"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
)

// load existing urbits from config json
func LoadUrbits() error {
	logger.Logger.Info("Loading Urbit ships")
	// Loop through pier list
	conf := config.Conf()
	for _, pier := range conf.Piers {
		logger.Logger.Info(fmt.Sprintf("Loading pier %s", pier))
		// load json into struct
		err := config.LoadUrbitConfig(pier)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Error loading %s config: %v", pier, err))
			continue
		}
		shipConf := config.UrbitConf(pier)
		// don't bootstrap if it's busted
		if shipConf.BootStatus != "noboot" {
			info, err := StartContainer(pier, "vere")
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("Error starting %s: %v", pier, err))
				continue
			}
			config.UpdateContainerState(pier, info)
		} else {
			info, err := CreateContainer(pier, "vere")
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("Error starting %s: %v", pier, err))
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
	minioInfo, err := GetLatestContainerInfo("minio")
	// compare existing config to current version info
	// update if new
	// sorry this is ugly
	shipConf := config.UrbitConf(containerName)
	newConf := shipConf
	if config.Architecture == "amd64" {
		if containerInfo["hash"] != shipConf.UrbitAmd64Sha256 {
			newConf.UrbitAmd64Sha256 = containerInfo["hash"]
		}
		if minioInfo["hash"] != shipConf.MinioAmd64Sha256 {
			newConf.MinioAmd64Sha256 = minioInfo["hash"]
		}
	} else if config.Architecture == "arm64" {
		if containerInfo["hash"] != shipConf.UrbitArm64Sha256 {
			newConf.UrbitArm64Sha256 = containerInfo["hash"]
		}
		if minioInfo["hash"] != shipConf.MinioArm64Sha256 {
			newConf.MinioArm64Sha256 = minioInfo["hash"]
		}
	}
	newConf.UrbitVersion = containerInfo["tag"]
	newConf.UrbitRepo = containerInfo["repo"]
	newConf.MinioVersion = minioInfo["tag"]
	newConf.MinioRepo = minioInfo["repo"]
	if shipConf != newConf {
		if err := config.UpdateUrbitConfig(map[string]structs.UrbitDocker{containerName: newConf}); err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't persist updated urbit conf! %v", err))
		}
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	// reload urbit conf from disk
	err = config.LoadUrbitConfig(containerName)
	if err != nil {
		errmsg := fmt.Errorf("Error loading %s config: %v", containerName, err)
		return containerConfig, hostConfig, errmsg
	}
	// todo: this BootStatus doesnt actually have anythin to do with pack and meld right now
	act := shipConf.BootStatus
	// get the correct startup script based on BootStatus val
	switch act {
	case "boot":
		scriptContent = defaults.StartScript
	case "ignore":
		scriptContent = defaults.StartScript
	case "pack":
		scriptContent = defaults.PackScript
	case "meld":
		scriptContent = defaults.MeldScript
	case "prep":
		scriptContent = defaults.PrepScript
	case "chop":
		scriptContent = defaults.ChopScript
	case "roll":
		scriptContent = defaults.RollScript
	case "noboot":
		return containerConfig, hostConfig, fmt.Errorf("%s marked noboot!", containerName)
	default:
		return containerConfig, hostConfig, fmt.Errorf("Unknown action: %s", act)
	}
	// reset ship status to boot for next time
	if act == "pack" || act == "meld" || act == "chop" {
		updateUrbitConf := shipConf
		updateUrbitConf.BootStatus = "noboot"
		newConfig := make(map[string]structs.UrbitDocker)
		newConfig[containerName] = updateUrbitConf
		err = config.UpdateUrbitConfig(newConfig)
		if err != nil {
			logger.Logger.Warn(fmt.Sprintf("Unable to reset %s boot script!", containerName))
		}
	} else if act != "boot" {
		updateUrbitConf := shipConf
		updateUrbitConf.BootStatus = "boot"
		newConfig := make(map[string]structs.UrbitDocker)
		newConfig[containerName] = updateUrbitConf
		err = config.UpdateUrbitConfig(newConfig)
		if err != nil {
			logger.Logger.Warn(fmt.Sprintf("Unable to reset %s boot script!", containerName))
		}
		// this is only for offline ships, otherwise we send a click
	}
	// write the script
	scriptPath := filepath.Join(config.DockerDir, containerName, "_data", "start_urbit.sh")
	if shipConf.CustomPierLocation != nil {
		if str, ok := shipConf.CustomPierLocation.(string); ok {
			scriptPath = filepath.Join(str, "start_urbit.sh")
		}
	}
	err = ioutil.WriteFile(scriptPath, []byte(scriptContent), 0755) // make the script executable
	if err != nil {
		return containerConfig, hostConfig, fmt.Errorf("Failed to write script: %v", err)
	}
	// gather boot option values
	shipName := shipConf.PierName
	loomValue := fmt.Sprintf("%v", shipConf.LoomSize)
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
		logger.Logger.Debug(fmt.Sprintf("%v ship conf: %v", containerName, shipConf))
		httpPort = fmt.Sprintf("%v", shipConf.WgHTTPPort)
		amesPort = fmt.Sprintf("%v", shipConf.WgAmesPort)
		network = "container:wireguard"
		containerConfig = container.Config{
			Image: desiredImage,
			Cmd: []string{
				"bash",
				"/urbit/start_urbit.sh",
				"--loom=" + loomValue,
				"--dirname=" + shipName,
				"--devmode=" + devMode,
				"--http-port=" + httpPort,
				"--port=" + amesPort,
			},
		}
	} else {
		httpPort = fmt.Sprintf("%v", shipConf.HTTPPort)
		amesPort = fmt.Sprintf("%v", shipConf.AmesPort)
		network = "default"
		//httpPortStr := nat.Port(fmt.Sprintf(httpPort + "/tcp"))
		//amesPortStr := nat.Port(fmt.Sprintf(amesPort + "/udp"))
		// Port mapping
		portMap = nat.PortMap{
			"80/tcp": []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: httpPort},
			},
			"34343/udp": []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: amesPort},
			},
		}
		// finally construct the container config structs
		containerConfig = container.Config{
			Image: desiredImage,
			ExposedPorts: nat.PortSet{
				"80/tcp":    struct{}{},
				"34343/udp": struct{}{},
			},
			Cmd: []string{
				"bash",
				"/urbit/start_urbit.sh",
				"--loom=" + loomValue,
				"--dirname=" + shipName,
				"--devmode=" + devMode,
			},
		}
	}
	mountType := mount.TypeVolume
	sourceStr := shipName
	if shipConf.CustomPierLocation != nil {
		mountType = mount.TypeBind
		if str, ok := shipConf.CustomPierLocation.(string); ok {
			sourceStr = str
		}
	}
	mounts := []mount.Mount{
		{
			Type:   mountType,
			Source: sourceStr,
			Target: "/urbit",
		},
	}

	hostConfig = container.HostConfig{
		NetworkMode:  container.NetworkMode(network),
		Mounts:       mounts,
		PortBindings: portMap,
	}
	logger.Logger.Debug(fmt.Sprintf("Boot command: %v", containerConfig.Cmd))
	return containerConfig, hostConfig, nil
}
