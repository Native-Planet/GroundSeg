package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"goseg/config"
	"goseg/structs"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

var (
	EventBus        = make(chan structs.Event, 100)
	UTransBus       = make(chan structs.UrbitTransition, 100)
	NewShipTransBus = make(chan structs.NewShipTransition, 100)
)

// return the container status of a slice of ships
func GetShipStatus(patps []string) (map[string]string, error) {
	statuses := make(map[string]string)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		errmsg := fmt.Sprintf("Error getting Docker info: %v", err)
		config.Logger.Error(errmsg)
		return statuses, err
	} else {
		containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
		if err != nil {
			errmsg := fmt.Sprintf("Error getting containers: %v", err)
			config.Logger.Error(errmsg)
			return statuses, err
		} else {
			for _, pier := range patps {
				found := false
				for _, container := range containers {
					for _, name := range container.Names {
						fasPier := "/" + pier
						if name == fasPier {
							statuses[pier] = container.Status
							found = true
							break
						}
					}
					if found {
						break
					}
				}
				if !found {
					statuses[pier] = "not found"
				}
			}
		}
		return statuses, nil
	}
}

// return the name of a container's network
func GetContainerNetwork(name string) (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}
	defer cli.Close()
	containerJSON, err := cli.ContainerInspect(context.Background(), name)
	if err != nil {
		return "", err
	}
	for networkName := range containerJSON.NetworkSettings.Networks {
		return networkName, nil
	}
	return "", fmt.Errorf("container is not attached to any network: %v", name)
}

// return the disk and memory usage for a container
func GetContainerStats(containerName string) (structs.ContainerStats, error) {
	var res structs.ContainerStats
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return res, err
	}
	defer cli.Close()
	statsResp, err := cli.ContainerStats(context.Background(), containerName, false)
	if err != nil {
		return res, err
	}
	defer statsResp.Body.Close()
	var stat types.StatsJSON
	if err := json.NewDecoder(statsResp.Body).Decode(&stat); err != nil {
		return res, err
	}
	memUsage := stat.MemoryStats.Usage
	inspectResp, err := cli.ContainerInspect(context.Background(), containerName)
	if err != nil {
		return res, err
	}
	diskUsage := int64(0)
	if inspectResp.SizeRw != nil {
		diskUsage = *inspectResp.SizeRw
	}
	return structs.ContainerStats{
		MemoryUsage: memUsage,
		DiskUsage:   diskUsage,
	}, nil
}

// creates a volume with name
func CreateVolume(name string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker client: %v : %v", name, err)
		return errmsg
	}

	// Create volume
	vol, err := cli.VolumeCreate(context.Background(), volume.CreateOptions{Name: name})
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker volume: %v : %v", name, err)
		return errmsg
	}
	// Output created volume information
	config.Logger.Info(fmt.Sprintf("Created volume: %s", vol.Name))
	return nil
}

// start a container by name + type
// contructs a container.Config, then runs through whether to boot/restart/etc
// saves the current container state in memory after completion
func StartContainer(containerName string, containerType string) (structs.ContainerState, error) {
	// bundle of info about container
	var containerState structs.ContainerState
	// config params for container
	var containerConfig container.Config
	// host config for container
	var hostConfig container.HostConfig
	// switch on containerType to process containerConfig
	switch containerType {
	case "vere":
		// containerConfig, HostConfig, err := urbitContainerConf(containerName)
		_, _, err := urbitContainerConf(containerName)
		if err != nil {
			return containerState, err
		}
	case "netdata":
		_, _, err := netdataContainerConf()
		if err != nil {
			return containerState, err
		}
	case "minio":
		_, _, err := minioContainerConf(containerName)
		if err != nil {
			return containerState, err
		}
	case "miniomc":
		_, _, err := mcContainerConf()
		if err != nil {
			return containerState, err
		}
	case "wireguard":
		_, _, err := wgContainerConf()
		if err != nil {
			return containerState, err
		}
	default:
		errmsg := fmt.Errorf("Unrecognized container type %s", containerType)
		return containerState, errmsg
	}
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return containerState, err
	}
	// get the desired tag and hash from config
	containerInfo, err := GetLatestContainerInfo(containerType)
	if err != nil {
		return containerState, err
	}
	// check if container exists
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return containerState, err
	}
	var existingContainer *types.Container = nil
	for _, container := range containers {
		for _, name := range container.Names {
			if name == "/"+containerName {
				existingContainer = &container
				break
			}
		}
		if existingContainer != nil {
			break
		}
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	desiredStatus := "running"
	// check if the desired image is available locally
	images, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return containerState, err
	}
	imageExistsLocally := false
	for _, img := range images {
		if img.ID == containerInfo["hash"] {
			imageExistsLocally = true
			break
		}
		if imageExistsLocally {
			break
		}
	}
	if !imageExistsLocally {
		// pull the image if it doesn't exist locally
		_, err = cli.ImagePull(ctx, desiredImage, types.ImagePullOptions{})
		if err != nil {
			return containerState, err
		}
	}
	switch {
	case existingContainer == nil:
		// if the container does not exist, create and start it
		_, err := cli.ContainerCreate(ctx, &containerConfig, nil, nil, nil, containerName)
		if err != nil {
			return containerState, err
		}
		err = cli.ContainerStart(ctx, containerName, types.ContainerStartOptions{})
		if err != nil {
			return containerState, err
		}
		msg := fmt.Sprintf("%s started with image %s", containerName, desiredImage)
		config.Logger.Info(msg)
	case existingContainer.State == "exited":
		// if the container exists but is stopped, start it
		err := cli.ContainerStart(ctx, containerName, types.ContainerStartOptions{})
		if err != nil {
			return containerState, err
		}
		msg := fmt.Sprintf("Started stopped container %s", containerName)
		config.Logger.Info(msg)
	default:
		// if container is running, check the image digest
		currentImage := existingContainer.Image
		digestParts := strings.Split(currentImage, "@sha256:")
		currentDigest := ""
		if len(digestParts) > 1 {
			currentDigest = digestParts[1]
		}
		if currentDigest != containerInfo["hash"] {
			// if the hashes don't match, recreate the container with the new one
			err := cli.ContainerRemove(ctx, containerName, types.ContainerRemoveOptions{Force: true})
			if err != nil {
				return containerState, err
			}
			_, err = cli.ContainerCreate(ctx, &container.Config{
				Image: desiredImage,
			}, nil, nil, nil, containerName)
			if err != nil {
				return containerState, err
			}
			err = cli.ContainerStart(ctx, containerName, types.ContainerStartOptions{})
			if err != nil {
				return containerState, err
			}
			msg := fmt.Sprintf("Restarted %s with image %s", containerName, desiredImage)
			config.Logger.Info(msg)
		}
	}
	containerDetails, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return containerState, fmt.Errorf("failed to inspect container %s: %v", containerName, err)
	}
	// save the current state of the container in memory for reference
	containerState = structs.ContainerState{
		ID:            containerDetails.ID,           // container id hash
		Name:          containerName,                 // name (eg @p)
		Image:         desiredImage,                  // full repo:tag@hash string
		Type:          containerType,                 // eg `vere` (corresponds with version server label)
		DesiredStatus: desiredStatus,                 // what the user sets
		ActualStatus:  containerDetails.State.Status, // what the daemon reports
		CreatedAt:     containerDetails.Created,      // this is a string
		Config:        containerConfig,               // container.Config struct constructed above
		Host:          hostConfig,                    // host.Config struct constructed above
	}
	return containerState, err
}

// convert the version info back into json then a map lol
// so we can easily get the correct repo/release channel/tag/hash
func GetLatestContainerInfo(containerType string) (map[string]string, error) {
	var res map[string]string
	arch := config.Architecture
	hashLabel := arch + "_sha256"
	versionInfo := config.VersionInfo
	jsonData, err := json.Marshal(versionInfo)
	if err != nil {
		return res, err
	}
	// Convert JSON to map
	var m map[string]interface{}
	err = json.Unmarshal(jsonData, &m)
	if err != nil {
		return res, err
	}
	containerData, ok := m[containerType].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s data is not a map", containerType)
	}
	tag, ok := containerData["tag"].(string)
	if !ok {
		return nil, fmt.Errorf("'tag' is not a string")
	}
	hashValue, ok := containerData[hashLabel].(string)
	if !ok {
		return nil, fmt.Errorf("'%s' is not a string", hashLabel)
	}
	repo, ok := containerData["repo"].(string)
	if !ok {
		return nil, fmt.Errorf("'repo' is not a string")
	}
	res = make(map[string]string)
	res["tag"] = tag
	res["hash"] = hashValue
	res["repo"] = repo
	return res, nil
}

// stop a container with the name
func StopContainerByName(containerName string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}
	// fetch all containers incl stopped
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return err
	}
	for _, cont := range containers {
		for _, name := range cont.Names {
			if name == "/"+containerName {
				// Stop the container
				options := container.StopOptions{}
				if err := cli.ContainerStop(ctx, cont.ID, options); err != nil {
					return fmt.Errorf("failed to stop container %s: %v", containerName, err)
				}
				config.Logger.Info(fmt.Sprintf("Successfully stopped container %s\n", containerName))
				return nil
			}
		}
	}
	return fmt.Errorf("container with name %s not found", containerName)
}

// subscribe to docker events and feed them into eventbus
func DockerListener() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		config.Logger.Error(fmt.Sprintf("Error initializing Docker client: %v", err))
		return
	}
	messages, errs := cli.Events(ctx, types.EventsOptions{})
	for {
		select {
		case event := <-messages:
			// Convert the Docker event to our custom event and send it to the EventBus
			EventBus <- structs.Event{Type: event.Action, Data: event}
		case err := <-errs:
			config.Logger.Error(fmt.Sprintf("Docker event error: %v", err))
		}
	}
}

// periodically poll docker in case we miss something
func DockerPoller() {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			config.Logger.Info("polling docker")
			// todo (maybe not necessary?)
			// fetch the status of all containers and compare with app's state
			// if there's a change, send an event to the EventBus
			return
		}
	}
}
