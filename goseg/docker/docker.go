package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"groundseg/config"
	"groundseg/dockerclient"
	"groundseg/structs"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	imagetypes "github.com/docker/docker/api/types/image"
	networktypes "github.com/docker/docker/api/types/network"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

var (
	VolumeDir          = config.DockerDir
	UTransBus          = make(chan structs.UrbitTransition, 100)   // urbit transition bus
	HermesTransBus     = make(chan structs.Event, 100)             // hermes profile transition bus
	SysTransBus        = make(chan structs.SystemTransition, 100)  // system transition bus
	NewShipTransBus    = make(chan structs.NewShipTransition, 100) // transition event bus
	ImportShipTransBus = make(chan structs.UploadTransition, 100)  // transition event bus
	ContainerStats     = make(map[string]structs.ContainerStats)   // used for broadcast
	ContainerStatList  []string                                    // slice of containers to poll for resource use
)

func init() {
	// kill old webui container if running
	_, err := FindContainer("groundseg-webui")
	if err == nil {
		if err = StopContainerByName("groundseg-webui"); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't stop old webui container: %v", err))
		}
	}
	if err = killContainerUsingPort(80); err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't stop container on port 80: %v", err))
	}
	cli, err := dockerclient.New()
	if err != nil {
		zap.L().Error(fmt.Sprintf("Error creating Docker client: %v", err))
		return
	}
	defer cli.Close()
	version, err := cli.ServerVersion(context.TODO())
	if err != nil {
		zap.L().Error(fmt.Sprintf("Error getting Docker version: %v", err))
		return
	}
	//go getContainerStats()
	zap.L().Info(fmt.Sprintf("Docker version: %s", version.Version))
}

func killContainerUsingPort(n uint16) error {
	// Initialize Docker client
	ctx := context.Background()
	cli, err := dockerclient.New()
	if err != nil {
		return err
	}
	defer cli.Close()

	// Prepare filters to get only running containers
	filters := filters.NewArgs()
	filters.Add("status", "running")

	// List running containers
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{Filters: filters})
	if err != nil {
		zap.L().Error(fmt.Sprintf("Unable to get container list. Failed to kill container using port %v", n))
		return err
	}

	// Check if any container is using host's port 80 and stop it
	for _, cont := range containers {
		for _, port := range cont.Ports {
			if port.PublicPort == n {
				zap.L().Debug(fmt.Sprintf("Stopping container %s to free port %v", cont.ID, n))
				options := container.StopOptions{}
				if err := cli.ContainerStop(ctx, cont.ID, options); err != nil {
					zap.L().Error(fmt.Sprintf("failed to stop container %s: %v", cont.ID, err))
				}
				return nil
			}
		}
	}
	return nil
}

// return the container status of a slice of ships
func GetShipStatus(patps []string) (map[string]string, error) {
	statuses := make(map[string]string)
	cli, err := dockerclient.New()
	if err != nil {
		errmsg := fmt.Sprintf("Error getting Docker info: %v", err)
		zap.L().Error(errmsg)
		return statuses, err
	} else {
		defer cli.Close()
		containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
		if err != nil {
			errmsg := fmt.Sprintf("Error getting containers: %v", err)
			zap.L().Error(errmsg)
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

func GetContainerRunningStatus(containerName string) (string, error) {
	var status string
	cli, err := dockerclient.New()
	if err != nil {
	}
	defer cli.Close()
	// List containers
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		return status, err
	}
	// Loop through containers to find the one with the given name
	for _, container := range containers {
		if slices.Contains(container.Names, "/"+containerName) {
			return container.Status, nil
		}
	}
	return status, fmt.Errorf("Unable to get container running status: %v", containerName)
}

// return the name of a container's network
func GetContainerNetwork(name string) (string, error) {
	cli, err := dockerclient.New()
	if err != nil {
		return "", err
	}
	defer cli.Close()
	containerJSON, err := cli.ContainerInspect(context.Background(), name)
	if err != nil {
		return "", err
	}
	if containerJSON.HostConfig.NetworkMode != "" {
		return string(containerJSON.HostConfig.NetworkMode), nil
	}
	return "", fmt.Errorf("container is not attached to any network: %v", name)
}

// creates a volume by name
func CreateVolume(name string) error {
	cli, err := dockerclient.New()
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker client: %v : %v", name, err)
		return errmsg
	}
	defer cli.Close()

	// Create volume
	vol, err := cli.VolumeCreate(context.Background(), volumetypes.CreateOptions{Name: name})
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker volume: %v : %v", name, err)
		return errmsg
	}
	// Output created volume information
	zap.L().Info(fmt.Sprintf("Created volume: %s", vol.Name))
	return nil
}

// deletes a volume by its name
func DeleteVolume(name string) error {
	cli, err := dockerclient.New()
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker client: %v : %v", name, err)
		return errmsg
	}
	defer cli.Close()
	// Remove the volume
	err = cli.VolumeRemove(context.Background(), name, true)
	if err != nil {
		errmsg := fmt.Errorf("Failed to remove docker volume: %v : %v", name, err)
		return errmsg
	}
	zap.L().Info(fmt.Sprintf("Deleted volume: %s", name))
	return nil
}

// deletes a container by its name
func DeleteContainer(name string) error {
	cli, err := dockerclient.New()
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker client: %v : %v", name, err)
		return errmsg
	}
	defer cli.Close()
	// Force-remove the container
	err = cli.ContainerRemove(context.Background(), name, container.RemoveOptions{Force: true})
	if err != nil {
		errmsg := fmt.Errorf("Failed to delete docker container: %v : %v", name, err)
		return errmsg
	}
	// Output created volume information
	zap.L().Info(fmt.Sprintf("Deleted Container: %s", name))
	return nil
}

// Write a file to a specific location in a volume
func WriteFileToVolume(name string, file string, content string) error {
	cli, err := dockerclient.New()
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker client: %v : %v", name, err)
		return errmsg
	}
	defer cli.Close()
	// Inspect volume
	vol, err := cli.VolumeInspect(context.Background(), name)
	if err != nil {
		errmsg := fmt.Errorf("Failed to inspect volume: %v : %v", name, err)
		return errmsg
	}
	// Get volume directory path
	fullPath := filepath.Join(vol.Mountpoint, file)
	// Write to file
	err = os.WriteFile(fullPath, []byte(content), 0644)
	if err != nil {
		errmsg := fmt.Errorf("Failed to write to volume: %v : %v", name, err)
		return errmsg
	}
	zap.L().Info(fmt.Sprintf("Successfully wrote to file: %s", fullPath))
	return nil
}

// start a container by name + type
// contructs a container.Config, then runs through whether to boot/restart/etc
// saves the current container state in memory after completion
func StartContainer(containerName string, containerType string) (structs.ContainerState, error) {
	zap.L().Debug(fmt.Sprintf("StartContainer issued for %v", containerName))
	// bundle of info about container
	var containerState structs.ContainerState
	// config params for container
	var containerConfig container.Config
	// host config for container
	var hostConfig container.HostConfig
	// init error
	var err error
	containerState.Type = containerType
	// switch on containerType to process containerConfig
	switch containerType {
	case "vere":
		containerConfig, hostConfig, err = urbitContainerConf(containerName)
		if err != nil {
			return containerState, err
		}
	case "netdata":
		containerConfig, hostConfig, err = netdataContainerConf()
		if err != nil {
			return containerState, err
		}
	case "minio":
		DeleteContainer(containerName)
		containerConfig, hostConfig, err = minioContainerConf(containerName)
		if err != nil {
			return containerState, err
		}
	case "hermes":
		containerConfig, hostConfig, err = hermesContainerConf(containerName)
		if err != nil {
			return containerState, err
		}
	case "wireguard":
		containerConfig, hostConfig, err = wgContainerConf()
		if err != nil {
			return containerState, err
		}
	default:
		errmsg := fmt.Errorf("Unrecognized container type %s", containerType)
		return containerState, errmsg
	}
	var imageInfo map[string]string
	desiredImage := containerConfig.Image
	desiredImageID := ""
	if containerType == "hermes" {
		if desiredImage == "" {
			return containerState, fmt.Errorf("empty image ref for %s", containerName)
		}
		installed, err := ImageRefExists(desiredImage)
		if err != nil {
			return containerState, err
		}
		if !installed {
			return containerState, fmt.Errorf("Hermes image %s is not installed", desiredImage)
		}
		imageInfo = map[string]string{"hash": ""}
	} else if containerType == "minio" {
		if desiredImage == "" {
			return containerState, fmt.Errorf("empty image ref for %s", containerName)
		}
		if err := PullImageByRef(desiredImage); err != nil {
			return containerState, err
		}
		imageInfo = map[string]string{"hash": ""}
	} else {
		// get the desired tag and hash from config
		imageInfo, err = GetLatestContainerInfo(containerType)
		if err != nil {
			return containerState, err
		}
		versionServerImage := fmt.Sprintf("%s:%s@sha256:%s", imageInfo["repo"], imageInfo["tag"], imageInfo["hash"])
		if strings.TrimSpace(desiredImage) == "" {
			desiredImage = versionServerImage
		}
		imageInfo = imageInfoFromImageRef(desiredImage, imageInfo)
		// check if the desired image is available locally
		if desiredImage == versionServerImage && imageInfo["hash"] != "" {
			_, err = PullImageIfNotExist(desiredImage, imageInfo)
			if err != nil {
				return containerState, err
			}
		} else {
			if err := PullImageByRef(desiredImage); err != nil {
				return containerState, err
			}
		}
		if desiredImageID, err = getLocalImageID(desiredImage, imageInfo); err != nil {
			zap.L().Warn(fmt.Sprintf("Unable to inspect desired image %s: %v", desiredImage, err))
		}
	}
	// check if container exists
	existingContainer, _ := FindContainer(containerName)

	ctx := context.Background()
	cli, err := dockerclient.New()
	if err != nil {
		return containerState, err
	}
	defer cli.Close()
	switch {
	case existingContainer == nil:
		// if the container does not exist, create and start it
		_, err := cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, containerName)
		if err != nil {
			return containerState, err
		}
		err = cli.ContainerStart(ctx, containerName, container.StartOptions{})
		if err != nil {
			return containerState, err
		}
		msg := fmt.Sprintf("%s started with image %s", containerName, desiredImage)
		zap.L().Info(msg)
	case containerConfigChanged(existingContainer, containerConfig):
		err := cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true})
		if err != nil {
			return containerState, err
		}
		_, err = cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, containerName)
		if err != nil {
			return containerState, err
		}
		err = cli.ContainerStart(ctx, containerName, container.StartOptions{})
		if err != nil {
			return containerState, err
		}
		msg := fmt.Sprintf("Recreated %s with updated container config", containerName)
		zap.L().Info(msg)
	case existingContainer.State == "exited":
		err := cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true})
		if err != nil {
			return containerState, err
		}
		_, err = cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, containerName)
		if err != nil {
			return containerState, err
		}
		err = cli.ContainerStart(ctx, containerName, container.StartOptions{})
		if err != nil {
			return containerState, err
		}
		msg := fmt.Sprintf("Started stopped container %s", containerName)
		zap.L().Info(msg)
	case existingContainer.State == "created":
		err := cli.ContainerStart(ctx, containerName, container.StartOptions{})
		if err != nil {
			return containerState, err
		}
		msg := fmt.Sprintf("Started created container %s", containerName)
		zap.L().Info(msg)
	default:
		// if container is running, check the image digest
		currentImage := existingContainer.Image
		digestParts := strings.Split(currentImage, "@sha256:")
		currentDigest := ""
		if len(digestParts) > 1 {
			currentDigest = digestParts[1]
		}
		imageMatches := desiredImageID != "" && imageIDsEqual(existingContainer.ImageID, desiredImageID)
		if !imageMatches && currentDigest != imageInfo["hash"] {
			// if the hashes don't match, recreate the container with the new one
			// for vere containers, gracefully stop with a 60s timeout before removing
			if containerType == "vere" {
				gracefulTimeout := 60
				stopOpts := container.StopOptions{Timeout: &gracefulTimeout}
				zap.L().Info(fmt.Sprintf("Gracefully stopping %s (60s timeout) before update", containerName))
				if err := cli.ContainerStop(ctx, containerName, stopOpts); err != nil {
					zap.L().Warn(fmt.Sprintf("Graceful stop failed for %s: %v, forcing removal", containerName, err))
				}
			}
			err := cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true})
			if err != nil {
				zap.L().Warn(fmt.Sprintf("Couldn't remove container %v (may not exist yet)", containerName))
			}
			_, err = cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, containerName)
			if err != nil {
				return containerState, err
			}
			err = cli.ContainerStart(ctx, containerName, container.StartOptions{})
			if err != nil {
				return containerState, err
			}
			msg := fmt.Sprintf("Restarted %s with image %s", containerName, desiredImage)
			zap.L().Info(msg)
		}
	}
	containerDetails, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return containerState, fmt.Errorf("failed to inspect container %s: %v", containerName, err)
	}
	if IsObjectStoreContainerName(containerName) {
		if err := setMinIOAdminAccount(containerName); err != nil {
			return containerState, fmt.Errorf("failed to set admin account %s: %v", containerName, err)
		}
	}
	desiredStatus := "running"
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

func containerConfigChanged(existingContainer *container.Summary, desiredConfig container.Config) bool {
	if existingContainer == nil || len(desiredConfig.Labels) == 0 {
		return false
	}
	for key, desiredValue := range desiredConfig.Labels {
		if existingContainer.Labels[key] != desiredValue {
			return true
		}
	}
	return false
}

// create a stopped container
func CreateContainer(containerName string, containerType string) (structs.ContainerState, error) {
	var containerState structs.ContainerState
	var containerConfig container.Config
	var hostConfig container.HostConfig
	var err error
	switch containerType {
	case "vere":
		containerConfig, hostConfig, err = urbitContainerConf(containerName)
		if err != nil {
			return containerState, err
		}
	case "netdata":
		containerConfig, hostConfig, err = netdataContainerConf()
		if err != nil {
			return containerState, err
		}
	case "minio":
		DeleteContainer(containerName)
		containerConfig, hostConfig, err = minioContainerConf(containerName)
		if err != nil {
			return containerState, err
		}
	case "hermes":
		containerConfig, hostConfig, err = hermesContainerConf(containerName)
		if err != nil {
			return containerState, err
		}
	case "wireguard":
		containerConfig, hostConfig, err = wgContainerConf()
		if err != nil {
			return containerState, err
		}
	default:
		errmsg := fmt.Errorf("Unrecognized container type %s", containerType)
		return containerState, errmsg
	}
	var desiredImage string
	if containerType == "hermes" {
		desiredImage = containerConfig.Image
		if desiredImage == "" {
			return containerState, fmt.Errorf("empty image ref for %s", containerName)
		}
		installed, err := ImageRefExists(desiredImage)
		if err != nil {
			return containerState, err
		}
		if !installed {
			return containerState, fmt.Errorf("Hermes image %s is not installed", desiredImage)
		}
	} else if containerType == "minio" {
		desiredImage = containerConfig.Image
		if desiredImage == "" {
			return containerState, fmt.Errorf("empty image ref for %s", containerName)
		}
		if err := PullImageByRef(desiredImage); err != nil {
			return containerState, err
		}
	} else {
		imageInfo, err := GetLatestContainerInfo(containerType)
		if err != nil {
			return containerState, err
		}
		versionServerImage := fmt.Sprintf("%s:%s@sha256:%s", imageInfo["repo"], imageInfo["tag"], imageInfo["hash"])
		if strings.TrimSpace(desiredImage) == "" {
			desiredImage = versionServerImage
		}
		imageInfo = imageInfoFromImageRef(desiredImage, imageInfo)
		// check if the desired image is available locally
		if desiredImage == versionServerImage && imageInfo["hash"] != "" {
			_, err = PullImageIfNotExist(desiredImage, imageInfo)
			if err != nil {
				return containerState, err
			}
		} else {
			if err := PullImageByRef(desiredImage); err != nil {
				return containerState, err
			}
		}
	}
	ctx := context.Background()
	cli, err := dockerclient.New()
	if err != nil {
		return containerState, err
	}
	defer cli.Close()
	_, err = cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, containerName)
	if err != nil {
		return containerState, err
	}
	containerDetails, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return containerState, fmt.Errorf("failed to inspect container %s: %v", containerName, err)
	}
	containerState = structs.ContainerState{
		ID:            containerDetails.ID,           // container id hash
		Name:          containerName,                 // name (eg @p)
		Image:         desiredImage,                  // full repo:tag@hash string
		Type:          containerType,                 // eg `vere` (corresponds with version server label)
		DesiredStatus: "stopped",                     // what the user sets
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
	arch := config.Architecture
	hashLabel := arch + "_sha256"
	detail, ok := containerVersionDetails(config.VersionInfo, containerType)
	if !ok || strings.TrimSpace(detail.Tag) == "" || strings.TrimSpace(detail.Repo) == "" {
		localVersion := config.LocalVersion()
		releaseChannel := config.Conf().UpdateBranch
		channel, selectedChannel, exactChannel := config.SelectVersionChannel(localVersion, releaseChannel)
		if !exactChannel {
			zap.L().Warn(fmt.Sprintf("Version channel %q not found locally; using %q", releaseChannel, selectedChannel))
		}
		detail, ok = containerVersionDetails(channel, containerType)
	}
	if !ok {
		return nil, fmt.Errorf("%s data is not configured", containerType)
	}
	tag := strings.TrimSpace(detail.Tag)
	if tag == "" {
		return nil, fmt.Errorf("%s tag is empty", containerType)
	}
	repo := strings.TrimSpace(detail.Repo)
	if repo == "" {
		return nil, fmt.Errorf("%s repo is empty", containerType)
	}
	hashValue := strings.TrimSpace(detail.Amd64Sha256)
	if arch != "amd64" {
		hashValue = strings.TrimSpace(detail.Arm64Sha256)
	}
	if hashValue == "" {
		return nil, fmt.Errorf("%s %s is empty", containerType, hashLabel)
	}
	return map[string]string{
		"tag":  tag,
		"hash": hashValue,
		"repo": repo,
		"type": containerType,
	}, nil
}

func containerVersionDetails(channel structs.Channel, containerType string) (structs.VersionDetails, bool) {
	switch strings.ToLower(strings.TrimSpace(containerType)) {
	case "groundseg":
		return channel.Groundseg, true
	case "manual":
		return channel.Manual, true
	case "rustfs":
		return channel.Rustfs, true
	case "minio":
		return channel.Minio, true
	case "miniomc", "mc":
		return channel.Miniomc, true
	case "netdata":
		return channel.Netdata, true
	case "vere":
		return channel.Vere, true
	case "webui":
		return channel.Webui, true
	case "wireguard":
		return channel.Wireguard, true
	default:
		return structs.VersionDetails{}, false
	}
}

// stop a container with the name
func StopContainerByName(containerName string) error {
	ctx := context.Background()
	cli, err := dockerclient.New()
	if err != nil {
		return err
	}
	defer cli.Close()
	// fetch all containers incl stopped
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
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
				zap.L().Info(fmt.Sprintf("Successfully stopped container %s\n", containerName))
				return nil
			}
		}
	}
	return fmt.Errorf("container with name %s not found", containerName)
}

func RestartContainerByName(containerName string) error {
	ctx := context.Background()
	cli, err := dockerclient.New()
	if err != nil {
		return err
	}
	defer cli.Close()
	timeout := 10
	if err := cli.ContainerRestart(ctx, containerName, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to restart container %s: %v", containerName, err)
	}
	zap.L().Info(fmt.Sprintf("Successfully restarted container %s", containerName))
	return nil
}

// pull the image if it doesn't exist locally
func PullImageIfNotExist(desiredImage string, imageInfo map[string]string) (bool, error) {
	ctx := context.Background()
	cli, err := dockerclient.New()
	if err != nil {
		return false, err
	}
	defer cli.Close()

	refs := imageRefCandidates(desiredImage, imageInfo)
	if _, ok := inspectImageRefs(ctx, cli, refs); ok {
		return true, nil
	}

	resp, err := cli.ImagePull(ctx, fmt.Sprintf("%s@sha256:%s", imageInfo["repo"], imageInfo["hash"]), imagetypes.PullOptions{})
	if err != nil {
		return false, err
	}
	defer resp.Close()
	io.Copy(ioutil.Discard, resp) // wait until it's done
	return true, nil
}

// pull image by reference (tag or digest) if missing locally
func PullImageByRef(imageRef string) error {
	return PullImageByRefWithProgress(imageRef, nil)
}

type imagePullMessage struct {
	Status         string `json:"status"`
	ID             string `json:"id"`
	Error          string `json:"error"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
}

type imageLayerProgress struct {
	current int64
	total   int64
}

func ImageRefExists(imageRef string) (bool, error) {
	if strings.TrimSpace(imageRef) == "" {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cli, err := dockerclient.New()
	if err != nil {
		return false, err
	}
	defer cli.Close()
	_, ok := inspectImageRefs(ctx, cli, dockerHubRefAliases(imageRef))
	return ok, nil
}

// PullImageByRefWithProgress pulls an image by tag or digest and reports coarse progress.
func PullImageByRefWithProgress(imageRef string, progress func(string)) error {
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return fmt.Errorf("empty image ref")
	}
	ctx := context.Background()
	cli, err := dockerclient.New()
	if err != nil {
		return err
	}
	defer cli.Close()

	if _, ok := inspectImageRefs(ctx, cli, dockerHubRefAliases(imageRef)); ok {
		emitImagePullProgress(progress, "installed")
		return nil
	}

	zap.L().Info(fmt.Sprintf("Pulling Docker image %s", imageRef))
	emitImagePullProgress(progress, "pulling")
	resp, err := cli.ImagePull(ctx, imageRef, imagetypes.PullOptions{})
	if err != nil {
		return err
	}
	defer resp.Close()
	layers := map[string]imageLayerProgress{}
	lastPercent := -1
	decoder := json.NewDecoder(resp)
	for {
		var msg imagePullMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading image pull progress for %s: %v", imageRef, err)
		}
		if msg.Error != "" {
			return fmt.Errorf("failed to pull image %s: %s", imageRef, msg.Error)
		}
		if msg.ID != "" && msg.ProgressDetail.Total > 0 {
			layers[msg.ID] = imageLayerProgress{
				current: msg.ProgressDetail.Current,
				total:   msg.ProgressDetail.Total,
			}
			percent := max(imagePullPercent(layers), lastPercent)
			if percent != lastPercent {
				lastPercent = percent
				emitImagePullProgress(progress, fmt.Sprintf("pulling %d%%", percent))
			}
			continue
		}
		status := strings.ToLower(strings.TrimSpace(msg.Status))
		if msg.ID == "" && status != "" {
			emitImagePullProgress(progress, status)
		}
	}
	emitImagePullProgress(progress, "installed")
	zap.L().Info(fmt.Sprintf("Docker image %s installed", imageRef))
	return nil
}

func imagePullPercent(layers map[string]imageLayerProgress) int {
	var current int64
	var total int64
	for _, layer := range layers {
		current += layer.current
		total += layer.total
	}
	if total <= 0 {
		return 0
	}
	percent := int(float64(current) / float64(total) * 100)
	if percent > 100 {
		return 100
	}
	return percent
}

func emitImagePullProgress(progress func(string), status string) {
	if progress != nil && strings.TrimSpace(status) != "" {
		progress(status)
	}
}

func imageInfoFromImageRef(imageRef string, fallback map[string]string) map[string]string {
	info := map[string]string{
		"repo": fallback["repo"],
		"tag":  fallback["tag"],
		"hash": fallback["hash"],
	}
	ref := strings.TrimSpace(imageRef)
	if ref == "" {
		return info
	}
	if beforeDigest, digest, ok := strings.Cut(ref, "@sha256:"); ok {
		ref = beforeDigest
		info["hash"] = digest
	} else {
		info["hash"] = ""
	}
	lastSlash := strings.LastIndex(ref, "/")
	lastColon := strings.LastIndex(ref, ":")
	if lastColon > lastSlash {
		info["repo"] = ref[:lastColon]
		info["tag"] = ref[lastColon+1:]
	} else if ref != "" {
		info["repo"] = ref
	}
	return info
}

func getLocalImageID(desiredImage string, imageInfo map[string]string) (string, error) {
	ctx := context.Background()
	cli, err := dockerclient.New()
	if err != nil {
		return "", err
	}
	defer cli.Close()
	id, ok := inspectImageRefs(ctx, cli, imageRefCandidates(desiredImage, imageInfo))
	if !ok {
		return "", fmt.Errorf("image not found locally")
	}
	return id, nil
}

func inspectImageRefs(ctx context.Context, cli *client.Client, refs []string) (string, bool) {
	for _, ref := range refs {
		if ref == "" {
			continue
		}
		img, _, err := cli.ImageInspectWithRaw(ctx, ref)
		if err == nil {
			return img.ID, true
		}
	}
	return "", false
}

func imageRefCandidates(desiredImage string, imageInfo map[string]string) []string {
	refs := dockerHubRefAliases(desiredImage)
	repo := imageInfo["repo"]
	tag := imageInfo["tag"]
	hash := imageInfo["hash"]
	for _, repoAlias := range dockerHubRepoAliases(repo) {
		if hash != "" {
			refs = appendUnique(refs, fmt.Sprintf("%s@sha256:%s", repoAlias, hash))
			if tag != "" {
				refs = appendUnique(refs, fmt.Sprintf("%s:%s@sha256:%s", repoAlias, tag, hash))
			}
		}
		if tag != "" && hash == "" {
			refs = appendUnique(refs, fmt.Sprintf("%s:%s", repoAlias, tag))
		}
	}
	return refs
}

func dockerHubRefAliases(ref string) []string {
	if ref == "" {
		return nil
	}
	refs := []string{ref}
	if after, ok := strings.CutPrefix(ref, "registry.hub.docker.com/"); ok {
		trimmed := after
		refs = appendUnique(refs, trimmed)
		refs = appendUnique(refs, "docker.io/"+trimmed)
	}
	if after, ok := strings.CutPrefix(ref, "docker.io/"); ok {
		trimmed := after
		refs = appendUnique(refs, trimmed)
		refs = appendUnique(refs, "registry.hub.docker.com/"+trimmed)
	}
	return refs
}

func dockerHubRepoAliases(repo string) []string {
	return dockerHubRefAliases(repo)
}

func appendUnique(items []string, item string) []string {
	if item == "" || slices.Contains(items, item) {
		return items
	}
	return append(items, item)
}

func imageIDsEqual(a string, b string) bool {
	return strings.TrimPrefix(a, "sha256:") == strings.TrimPrefix(b, "sha256:")
}

// looks for a container with the given name and returns it, or nil if not found
func FindContainer(containerName string) (*container.Summary, error) {
	cli, err := dockerclient.New()
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	// Fetch list of running containers
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}
	// Search for the container with the given name
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.TrimPrefix(name, "/") == containerName {
				return &container, nil
			}
		}
	}
	return nil, fmt.Errorf("Container %v not found", containerName)
}

// periodically poll docker in case we miss something
func DockerPoller() {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			zap.L().Info("polling docker")
			// todo (maybe not necessary?)
			// fetch the status of all containers and compare with app's state
			// if there's a change, send an event to the EventBus
			return
		}
	}
}

// execute command
func ExecDockerCommand(containerName string, cmd []string) (string, error) {
	cli, err := dockerclient.New()
	if err != nil {
		return "", err
	}
	defer cli.Close()
	// Create an Exec configuration
	execConfig := container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	}
	// Context
	ctx := context.Background()

	// Get container ID by name
	containerID, err := GetContainerIDByName(ctx, cli, containerName)
	if err != nil {
		return "", err
	}
	// Create an exec instance, replace 'container_id_here' with your container ID
	resp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", err
	}

	// Start the exec command
	hijackedResp, err := cli.ContainerExecAttach(ctx, resp.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", err
	}
	defer hijackedResp.Close()

	// Read the output
	//stdout, err := ioutil.ReadAll(hijackedResp.Reader)
	output, err := io.ReadAll(hijackedResp.Reader)
	if err != nil {
		return "", err
	}
	inspect, err := cli.ContainerExecInspect(ctx, resp.ID)
	if err != nil {
		return "", err
	}
	if inspect.ExitCode != 0 {
		return string(output), fmt.Errorf("docker exec exited with code %d", inspect.ExitCode)
	}
	return string(output), nil
}

// Function to get container ID by name
func GetContainerIDByName(ctx context.Context, cli *client.Client, name string) (string, error) {
	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, container := range containers {
		if slices.Contains(container.Names, "/"+name) {
			return container.ID, nil
		}
	}
	return "", fmt.Errorf("Container not found")
}

// restart a running container
func RestartContainer(name string) error {
	ctx := context.Background()
	cli, err := dockerclient.New()
	if err != nil {
		return fmt.Errorf("Couldn't create client: %v", err)
	}
	defer cli.Close()
	containerID, err := GetContainerIDByName(ctx, cli, name)
	if err != nil {
		return fmt.Errorf("Couldn't get ID: %v", err)
	}
	timeout := 30
	stopOptions := container.StopOptions{
		Timeout: &timeout,
	}
	if err := cli.ContainerRestart(ctx, containerID, stopOptions); err != nil {
		return fmt.Errorf("Couldn't restart container: %v", err)
	}
	return nil
}

func contains(slice []string, str string) bool {
	return slices.Contains(slice, str)
}

func volumeExists(volumeName string) (bool, error) {
	cli, err := dockerclient.New()
	if err != nil {
		return false, fmt.Errorf("Failed to create client: %v", err)
	}
	defer cli.Close()
	volumeList, err := cli.VolumeList(context.Background(), volumetypes.ListOptions{})
	if err != nil {
		return false, err
	}
	for _, volume := range volumeList.Volumes {
		if volume.Name == volumeName {
			return true, nil
		}
	}
	return false, nil
}

func addOrGetNetwork(networkName string) (string, error) {
	cli, err := dockerclient.New()
	if err != nil {
		return "", fmt.Errorf("Failed to create client: %v", err)
	}
	defer cli.Close()
	networks, err := cli.NetworkList(context.Background(), networktypes.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("Failed to list networks: %v", err)
	}
	for _, network := range networks {
		if network.Name == networkName {
			return network.ID, nil
		}
	}
	networkResponse, err := cli.NetworkCreate(context.Background(), networkName, networktypes.CreateOptions{
		Driver: "bridge",
		Scope:  "local",
	})
	if err != nil {
		return "", fmt.Errorf("Failed to create custom bridge network: %v", err)
	}
	return networkResponse.ID, nil
}
