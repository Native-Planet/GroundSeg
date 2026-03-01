package docker

// start up urbits

import (
	"fmt"
	"groundseg/defaults"
	"groundseg/structs"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
)

type urbitRuntime = dockerRuntime

func newUrbitRuntime() urbitRuntime {
	return newDockerRuntime()
}

// load existing urbits from config json
func LoadUrbits() error {
	return loadUrbits(newUrbitRuntime())
}

func loadUrbits(rt urbitRuntime) error {
	zap.L().Info("Loading Urbit ships")
	// Loop through pier list
	ships := rt.shipSettings()
	for _, pier := range ships.Piers {
		zap.L().Info(fmt.Sprintf("Loading pier %s", pier))
		// load json into struct
		err := rt.loadUrbitConfig(pier)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Error loading %s config: %v", pier, err))
			continue
		}
		shipConf := rt.urbitConf(pier)
		// don't bootstrap if it's busted
		if shipConf.BootStatus != "noboot" {
			info, err := rt.startContainer(pier, "vere")
			if err != nil {
				zap.L().Error(fmt.Sprintf("Error starting %s: %v", pier, err))
				continue
			}
			rt.updateContainerState(pier, info)
		} else {
			info, err := rt.createContainer(pier, "vere")
			if err != nil {
				zap.L().Error(fmt.Sprintf("Error starting %s: %v", pier, err))
				continue
			}
			rt.updateContainerState(pier, info)
		}
	}
	return nil
}

// urbit container config builder
func urbitContainerConf(containerName string) (container.Config, container.HostConfig, error) {
	return urbitContainerConfWithRuntime(newUrbitRuntime(), containerName)
}

func urbitContainerConfWithRuntime(rt urbitRuntime, containerName string) (container.Config, container.HostConfig, error) {
	runtimeSettings := rt.runtimeSettings()
	var containerConfig container.Config
	var hostConfig container.HostConfig
	var scriptContent string
	// construct the container metadata from version server info
	containerInfo, err := rt.getLatestContainerInfo("vere")
	if err != nil {
		return containerConfig, hostConfig, err
	}
	minioInfo, err := rt.getLatestContainerInfo("minio")
	if err != nil {
		return containerConfig, hostConfig, err
	}
	// compare existing config to current version info
	// update if new
	// sorry this is ugly
	shipConf := rt.urbitConf(containerName)
	newConf := shipConf
	if rt.architecture() == "amd64" {
		if containerInfo["hash"] != shipConf.UrbitAmd64Sha256 {
			newConf.UrbitAmd64Sha256 = containerInfo["hash"]
		}
		if minioInfo["hash"] != shipConf.MinioAmd64Sha256 {
			newConf.MinioAmd64Sha256 = minioInfo["hash"]
		}
	} else if rt.architecture() == "arm64" {
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
		if err := rt.updateUrbit(containerName, func(conf *structs.UrbitDocker) error {
			*conf = newConf
			return nil
		}); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't persist updated urbit conf! %v", err))
		}
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	// reload urbit conf from disk
	err = rt.loadUrbitConfig(containerName)
	if err != nil {
		errmsg := fmt.Errorf("Error loading %s config: %v", containerName, err)
		return containerConfig, hostConfig, errmsg
	}
	// todo: this BootStatus doesnt actually have anythin to do with pack and meld right now
	act := shipConf.BootStatus
	// get the correct startup script based on BootStatus val
	switch act {
	case "boot", "noboot":
		// we'll still give it the start script if its noboot.
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
	default:
		return containerConfig, hostConfig, fmt.Errorf("Unknown action: %s", act)
	}
	// reset ship status to boot for next time
	switch act {
	case "pack", "meld", "chop", "noboot":
		// we'll set this to noboot because we want to manually control the boot
		// status the next time handler (or other modules) decides to call this func
		updateUrbitConf := shipConf
		updateUrbitConf.BootStatus = "noboot"
		err = rt.updateUrbit(containerName, func(conf *structs.UrbitDocker) error {
			*conf = updateUrbitConf
			return nil
		})
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Unable to reset %s boot script!", containerName))
		}
	default:
		// set everything else back to boot
		updateUrbitConf := shipConf
		updateUrbitConf.BootStatus = "boot"
		err = rt.updateUrbit(containerName, func(conf *structs.UrbitDocker) error {
			*conf = updateUrbitConf
			return nil
		})
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Unable to reset %s boot script!", containerName))
		}
	}
	// write the script
	scriptPath := filepath.Join(rt.dockerDir(), containerName, "_data", "start_urbit.sh")
	if shipConf.CustomPierLocation != nil {
		if str, ok := shipConf.CustomPierLocation.(string); ok {
			scriptPath = filepath.Join(str, "start_urbit.sh")
		}
	}
	err = rt.writeFile(scriptPath, []byte(scriptContent), 0755) // make the script executable
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
	snapTime := "60"
	// global snap time default
	if runtimeSettings.SnapTime != 0 && runtimeSettings.SnapTime != 60 {
		snapTime = fmt.Sprintf("%v", runtimeSettings.SnapTime)
	}
	// per-ship snap time default
	if shipConf.SnapTime != 0 && shipConf.SnapTime != 60 {
		snapTime = fmt.Sprintf("%v", shipConf.SnapTime)
	}
	// construct the network configuration based on conf val
	var httpPort string
	var amesPort string
	var network string
	var portMap nat.PortMap
	if shipConf.Network == "wireguard" {
		zap.L().Debug(fmt.Sprintf("%v ship conf: %v", containerName, shipConf))
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
				"--snap-time=" + snapTime,
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
				"--snap-time=" + snapTime,
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
	zap.L().Debug(fmt.Sprintf("Boot command: %v", containerConfig.Cmd))
	return containerConfig, hostConfig, nil
}
