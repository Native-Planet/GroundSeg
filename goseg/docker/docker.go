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
	"path/filepath"
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
	VolumeDir      = config.DockerDir
	ContainerStats = make(map[string]structs.ContainerStats) // used for broadcast
)

func Initialize() error {
	// kill old webui container if running
	_, err := FindContainer("groundseg-webui")
	if err == nil {
		if err = StopContainerByName("groundseg-webui"); err != nil {
			zap.L().Warn(fmt.Sprintf("Couldn't stop old webui container: %v", err))
		}
	}
	if err = killContainerUsingPort(80); err != nil {
		zap.L().Warn(fmt.Sprintf("Couldn't stop container on port 80: %v", err))
	}
	cli, err := dockerclient.New()
	if err != nil {
		return fmt.Errorf("error creating docker client: %v", err)
	}
	defer cli.Close()
	version, err := cli.ServerVersion(context.TODO())
	if err != nil {
		return fmt.Errorf("error getting docker version: %v", err)
	}
	//go getContainerStats()
	zap.L().Info(fmt.Sprintf("Docker version: %s", version.Version))
	return nil
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
		for _, name := range container.Names {
			if name == "/"+containerName {
				return container.Status, nil
			}
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
	err = ioutil.WriteFile(fullPath, []byte(content), 0644)
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
type containerPlan struct {
	Name         string
	Type         string
	Config       container.Config
	HostConfig   container.HostConfig
	ImageInfo    map[string]string
	DesiredImage string
}

func containerConfigForType(containerName string, containerType string) (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	var err error
	switch containerType {
	case "vere":
		containerConfig, hostConfig, err = urbitContainerConf(containerName)
	case "netdata":
		containerConfig, hostConfig, err = netdataContainerConf()
	case "minio":
		DeleteContainer(containerName)
		containerConfig, hostConfig, err = minioContainerConf(containerName)
	case "miniomc":
		containerConfig, hostConfig, err = mcContainerConf()
	case "wireguard":
		containerConfig, hostConfig, err = wgContainerConf()
	case "llama-api":
		containerConfig, hostConfig, err = llamaApiContainerConf()
	default:
		return containerConfig, hostConfig, fmt.Errorf("Unrecognized container type %s", containerType)
	}
	if err != nil {
		return containerConfig, hostConfig, err
	}
	return containerConfig, hostConfig, nil
}

func buildContainerPlan(containerName string, containerType string) (containerPlan, error) {
	plan := containerPlan{
		Name: containerName,
		Type: containerType,
	}
	containerConfig, hostConfig, err := containerConfigForType(containerName, containerType)
	if err != nil {
		return plan, err
	}
	plan.Config = containerConfig
	plan.HostConfig = hostConfig

	imageInfo, err := GetLatestContainerInfo(containerType)
	if err != nil {
		return plan, err
	}
	plan.ImageInfo = imageInfo
	plan.DesiredImage = fmt.Sprintf("%s:%s@sha256:%s", imageInfo["repo"], imageInfo["tag"], imageInfo["hash"])
	if _, err := PullImageIfNotExist(plan.DesiredImage, imageInfo); err != nil {
		return plan, err
	}
	return plan, nil
}

func createAndStartContainer(ctx context.Context, cli *client.Client, plan containerPlan) error {
	if _, err := cli.ContainerCreate(ctx, &plan.Config, &plan.HostConfig, nil, nil, plan.Name); err != nil {
		return err
	}
	if err := cli.ContainerStart(ctx, plan.Name, container.StartOptions{}); err != nil {
		return err
	}
	return nil
}

func recreateContainerIfImageChanged(ctx context.Context, cli *client.Client, plan containerPlan, currentImage string) error {
	digestParts := strings.Split(currentImage, "@sha256:")
	currentDigest := ""
	if len(digestParts) > 1 {
		currentDigest = digestParts[1]
	}
	if currentDigest == plan.ImageInfo["hash"] {
		return nil
	}
	if plan.Type == "vere" {
		gracefulTimeout := 60
		stopOpts := container.StopOptions{Timeout: &gracefulTimeout}
		zap.L().Info(fmt.Sprintf("Gracefully stopping %s (60s timeout) before update", plan.Name))
		if err := cli.ContainerStop(ctx, plan.Name, stopOpts); err != nil {
			zap.L().Warn(fmt.Sprintf("Graceful stop failed for %s: %v, forcing removal", plan.Name, err))
		}
	}
	if err := cli.ContainerRemove(ctx, plan.Name, container.RemoveOptions{Force: true}); err != nil {
		zap.L().Warn(fmt.Sprintf("Couldn't remove container %v (may not exist yet)", plan.Name))
	}
	if err := createAndStartContainer(ctx, cli, plan); err != nil {
		return err
	}
	zap.L().Info(fmt.Sprintf("Restarted %s with image %s", plan.Name, plan.DesiredImage))
	return nil
}

func ensureRunningContainer(ctx context.Context, cli *client.Client, plan containerPlan) error {
	existingContainer, _ := FindContainer(plan.Name)
	switch {
	case existingContainer == nil:
		if err := createAndStartContainer(ctx, cli, plan); err != nil {
			return err
		}
		zap.L().Info(fmt.Sprintf("%s started with image %s", plan.Name, plan.DesiredImage))
	case existingContainer.State == "exited":
		if err := cli.ContainerRemove(ctx, plan.Name, container.RemoveOptions{Force: true}); err != nil {
			return err
		}
		if err := createAndStartContainer(ctx, cli, plan); err != nil {
			return err
		}
		zap.L().Info(fmt.Sprintf("Started stopped container %s", plan.Name))
	case existingContainer.State == "created":
		if err := cli.ContainerStart(ctx, plan.Name, container.StartOptions{}); err != nil {
			return err
		}
		zap.L().Info(fmt.Sprintf("Started created container %s", plan.Name))
	default:
		if err := recreateContainerIfImageChanged(ctx, cli, plan, existingContainer.Image); err != nil {
			return err
		}
	}
	return nil
}

func ensureCreatedContainer(ctx context.Context, cli *client.Client, plan containerPlan) error {
	_, err := cli.ContainerCreate(ctx, &plan.Config, &plan.HostConfig, nil, nil, plan.Name)
	return err
}

func containerStateFromInspect(plan containerPlan, desiredStatus string, containerDetails container.InspectResponse) structs.ContainerState {
	return structs.ContainerState{
		ID:            containerDetails.ID,           // container id hash
		Name:          plan.Name,                     // name (eg @p)
		Image:         plan.DesiredImage,             // full repo:tag@hash string
		Type:          plan.Type,                     // eg `vere` (corresponds with version server label)
		DesiredStatus: desiredStatus,                 // what the user sets
		ActualStatus:  containerDetails.State.Status, // what the daemon reports
		CreatedAt:     containerDetails.Created,      // this is a string
		Config:        plan.Config,                   // container.Config struct constructed above
		Host:          plan.HostConfig,               // host.Config struct constructed above
	}
}

func StartContainer(containerName string, containerType string) (structs.ContainerState, error) {
	zap.L().Debug(fmt.Sprintf("StartContainer issued for %v", containerName))
	var containerState structs.ContainerState
	plan, err := buildContainerPlan(containerName, containerType)
	if err != nil {
		return containerState, err
	}

	ctx := context.Background()
	cli, err := dockerclient.New()
	if err != nil {
		return containerState, err
	}
	defer cli.Close()
	if err := ensureRunningContainer(ctx, cli, plan); err != nil {
		return containerState, err
	}
	containerDetails, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return containerState, fmt.Errorf("failed to inspect container %s: %v", containerName, err)
	}
	if strings.Contains(containerName, "minio_") {
		if err := setMinIOAdminAccount(containerName); err != nil {
			return containerState, fmt.Errorf("failed to set admin account %s: %v", containerName, err)
		}
	}
	containerState = containerStateFromInspect(plan, "running", containerDetails)
	return containerState, err
}

// create a stopped container
func CreateContainer(containerName string, containerType string) (structs.ContainerState, error) {
	var containerState structs.ContainerState
	plan, err := buildContainerPlan(containerName, containerType)
	if err != nil {
		return containerState, err
	}
	ctx := context.Background()
	cli, err := dockerclient.New()
	if err != nil {
		return containerState, err
	}
	defer cli.Close()
	if err := ensureCreatedContainer(ctx, cli, plan); err != nil {
		return containerState, err
	}
	containerDetails, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return containerState, fmt.Errorf("failed to inspect container %s: %v", containerName, err)
	}
	containerState = containerStateFromInspect(plan, "stopped", containerDetails)
	return containerState, err
}

// convert the version info back into json then a map lol
// so we can easily get the correct repo/release channel/tag/hash
func GetLatestContainerInfo(containerType string) (map[string]string, error) {
	var res map[string]string
	// hardcoded llama stuff for testing
	res = make(map[string]string)
	if containerType == "llama-api" {
		res["tag"] = "dev"
		res["hash"] = "ac2dcfac72bc3d8ee51ee255edecc10072ef9c0f958120971c00be5f4944a6fa"
		res["repo"] = "nativeplanet/llama-gpt"
		return res, nil
	}
	arch := config.Architecture
	hashLabel := arch + "_sha256"
	versionInfo := config.GetVersionChannel()
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
	res["type"] = containerType
	return res, nil
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

// pull the image if it doesn't exist locally
func PullImageIfNotExist(desiredImage string, imageInfo map[string]string) (bool, error) {
	ctx := context.Background()
	cli, err := dockerclient.New()
	if err != nil {
		return false, err
	}
	defer cli.Close()
	images, err := cli.ImageList(ctx, imagetypes.ListOptions{})
	if err != nil {
		return false, err
	}
	for _, img := range images {
		for _, digest := range img.RepoDigests {
			if digest == fmt.Sprintf("%s@sha256:%s", imageInfo["repo"], imageInfo["hash"]) {
				return true, nil
			}
		}
	}
	resp, err := cli.ImagePull(ctx, fmt.Sprintf("%s@sha256:%s", imageInfo["repo"], imageInfo["hash"]), imagetypes.PullOptions{})
	if err != nil {
		return false, err
	}
	defer resp.Close()
	io.Copy(ioutil.Discard, resp) // wait until it's done
	return true, nil
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
	output, err := ioutil.ReadAll(hijackedResp.Reader)
	if err != nil {
		return "", err
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
		for _, n := range container.Names {
			if n == "/"+name {
				return container.ID, nil
			}
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
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
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
