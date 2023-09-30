package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"goseg/config"
	"goseg/logger"
	"goseg/structs"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

var (
	UTransBus          = make(chan structs.UrbitTransition, 100)
	NewShipTransBus    = make(chan structs.NewShipTransition, 100)
	ImportShipTransBus = make(chan structs.UploadTransition, 100)
	ContainerStats     = make(map[string]structs.ContainerStats)
	ContainerStatList  []string
)

func init() {
	// kill old webui container if running
	killContainerUsingPort(80)
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Error creating Docker client: %v", err))
		return
	}
	version, err := cli.ServerVersion(context.TODO())
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Error getting Docker version: %v", err))
		if strings.Contains(err.Error(), "is too new") {
			updateDocker()
		}
		return
	}
	go getContainerStats()
	logger.Logger.Info(fmt.Sprintf("Docker version: %s", version.Version))
}

func killContainerUsingPort(n uint16) {
	// Initialize Docker client
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	// Prepare filters to get only running containers
	filters := filters.NewArgs()
	filters.Add("status", "running")

	// List running containers
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{Filters: filters})
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Unable to get container list. Failed to kill container using port %v", n))
		return
	}

	// Check if any container is using host's port 80 and stop it
	for _, cont := range containers {
		for _, port := range cont.Ports {
			if port.PublicPort == n {
				logger.Logger.Debug(fmt.Sprintf("Stopping container %s using host's port %v", cont.ID, n))
				options := container.StopOptions{}
				if err := cli.ContainerStop(ctx, cont.ID, options); err != nil {
					logger.Logger.Error(fmt.Sprintf("failed to stop container %s: %v", cont.ID, err))
				}
				return
			}
		}
	}
}

// attempt to update docker daemon (ubuntu/mint only)
func updateDocker() {
	logger.Logger.Info("Unsupported Docker version detected -- attempting to upgrade")
	packages := []string{"docker.io", "docker-doc", "docker-compose", "podman-docker", "containerd", "runc"}
	for _, pkg := range packages {
		out, err := exec.Command("apt-get", "remove", "-y", pkg).CombinedOutput()
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Error removing package %s: %v\n%s", pkg, err, out))
			return
		}
	}
	commands := []string{
		"apt-get update",
		"apt-get install -y ca-certificates curl gnupg",
		"install -m 0755 -d /etc/apt/keyrings",
		`curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor --yes -o /etc/apt/keyrings/docker.gpg`, // Added --yes
		"chmod a+r /etc/apt/keyrings/docker.gpg",
	}
	for _, cmd := range commands {
		out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Error executing command '%s': %v\n%s", cmd, err, out))
		}
	}
	out, err := exec.Command("sh", "-c", ". /etc/os-release && echo $VERSION_CODENAME").Output()
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Error fetching version codename: %v\n%s", err, out))
		return
	}
	codename := strings.TrimSpace(string(out))
	if contains([]string{"ulyana", "ulyssa", "uma", "una"}, codename) {
		codename = "focal"
	} else if contains([]string{"vanessa", "vera", "victoria"}, codename) {
		codename = "jammy"
	}
	archOut, archErr := exec.Command("sh", "-c", "dpkg --print-architecture").Output()
	if archErr != nil {
		logger.Logger.Error(fmt.Sprintf("Error fetching system architecture: %v\n%s", archErr, archOut))
		return
	}
	architecture := strings.TrimSpace(string(archOut))
	sourcesList := fmt.Sprintf("deb [arch=%s signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu %s stable", architecture, codename)
	cmd := fmt.Sprintf("echo '%s' | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null", sourcesList)
	out, err = exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Error updating Docker sources list: %v\n%s", err, out))
		return
	}
	dockerPackages := []string{"install", "-y", "docker-ce", "docker-ce-cli", "containerd.io"}
	out, err = exec.Command("apt-get", dockerPackages...).CombinedOutput()
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Error installing Docker packages: %v\n%s", err, out))
		return
	}
	logger.Logger.Info("Successfully updated Docker")
}

// return the container status of a slice of ships
func GetShipStatus(patps []string) (map[string]string, error) {
	statuses := make(map[string]string)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		errmsg := fmt.Sprintf("Error getting Docker info: %v", err)
		logger.Logger.Error(errmsg)
		return statuses, err
	} else {
		containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
		if err != nil {
			errmsg := fmt.Sprintf("Error getting containers: %v", err)
			logger.Logger.Error(errmsg)
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
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
	}
	// List containers
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
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
func getContainerStats() (structs.ContainerStats, error) {
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ticker.C:
			for _, pier := range ContainerStatList {
				var res structs.ContainerStats
				cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
				if err != nil {
					return res, err
				}
				defer cli.Close()
				inspect, err := cli.ContainerInspect(context.Background(), pier)
				if err != nil {
					return res, err
				}
				var totalSize int64
				for _, mount := range inspect.Mounts {
					if mount.Type == "volume" {
						size, err := getDirSize(mount.Source)
						if err != nil {
							return res, err
						}
						totalSize += size
					}
				}
				statsResp, err := cli.ContainerStats(context.Background(), pier, false)
				if err != nil {
					return res, err
				}
				defer statsResp.Body.Close()
				var stat types.StatsJSON
				if err := json.NewDecoder(statsResp.Body).Decode(&stat); err != nil {
					return res, err
				}
				memUsage := stat.MemoryStats.Usage
				ContainerStats[pier] = structs.ContainerStats{
					MemoryUsage: memUsage,
					DiskUsage:   totalSize,
				}
			}
		}
	}
}

func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

// creates a volume by name
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
	logger.Logger.Info(fmt.Sprintf("Created volume: %s", vol.Name))
	return nil
}

// deletes a volume by its name
func DeleteVolume(name string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
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
	logger.Logger.Info(fmt.Sprintf("Deleted volume: %s", name))
	return nil
}

// deletes a container by its name
func DeleteContainer(name string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker client: %v : %v", name, err)
		return errmsg
	}
	defer cli.Close()
	// Force-remove the container
	err = cli.ContainerRemove(context.Background(), name, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		errmsg := fmt.Errorf("Failed to delete docker container: %v : %v", name, err)
		return errmsg
	}
	// Output created volume information
	logger.Logger.Info(fmt.Sprintf("Deleted Container: %s", name))
	return nil
}

// Write a file to a specific location in a volume
func WriteFileToVolume(name string, file string, content string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker client: %v : %v", name, err)
		return errmsg
	}
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
	logger.Logger.Info(fmt.Sprintf("Successfully wrote to file: %s", fullPath))
	return nil
}

// start a container by name + type
// contructs a container.Config, then runs through whether to boot/restart/etc
// saves the current container state in memory after completion
func StartContainer(containerName string, containerType string) (structs.ContainerState, error) {
	logger.Logger.Debug(fmt.Sprintf("StartContainer issued for %v",containerName))
	// bundle of info about container
	var containerState structs.ContainerState
	// config params for container
	var containerConfig container.Config
	// host config for container
	var hostConfig container.HostConfig
	// init error
	var err error
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
	case "miniomc":
		containerConfig, hostConfig, err = mcContainerConf()
		if err != nil {
			return containerState, err
		}
	case "wireguard":
		containerConfig, hostConfig, err = wgContainerConf()
		if err != nil {
			return containerState, err
		}
	case "llama-api":
		containerConfig, hostConfig, err = llamaApiContainerConf()
		if err != nil {
			return containerState, err
		}
	case "llama-ui":
		containerConfig, hostConfig, err = llamaUIContainerConf()
		if err != nil {
			return containerState, err
		}
	default:
		errmsg := fmt.Errorf("Unrecognized container type %s", containerType)
		return containerState, errmsg
	}
	// get the desired tag and hash from config
	imageInfo, err := GetLatestContainerInfo(containerType)
	if err != nil {
		return containerState, err
	}
	// check if the desired image is available locally
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", imageInfo["repo"], imageInfo["tag"], imageInfo["hash"])
	_, err = PullImageIfNotExist(desiredImage, imageInfo)
	if err != nil {
		return containerState, err
	}
	// check if container exists
	existingContainer, err := FindContainer(containerName)
	if err != nil {
		return containerState, err
	}
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return containerState, err
	}
	switch {
	case existingContainer == nil:
		// if the container does not exist, create and start it
		_, err := cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, containerName)
		if err != nil {
			return containerState, err
		}
		err = cli.ContainerStart(ctx, containerName, types.ContainerStartOptions{})
		if err != nil {
			return containerState, err
		}
		msg := fmt.Sprintf("%s started with image %s", containerName, desiredImage)
		logger.Logger.Info(msg)
	case existingContainer.State == "exited":
		err := cli.ContainerRemove(ctx, containerName, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			return containerState, err
		}
		_, err = cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, containerName)
		if err != nil {
			return containerState, err
		}
		err = cli.ContainerStart(ctx, containerName, types.ContainerStartOptions{})
		if err != nil {
			return containerState, err
		}
		msg := fmt.Sprintf("Started stopped container %s", containerName)
		logger.Logger.Info(msg)
	default:
		// if container is running, check the image digest
		currentImage := existingContainer.Image
		digestParts := strings.Split(currentImage, "@sha256:")
		currentDigest := ""
		if len(digestParts) > 1 {
			currentDigest = digestParts[1]
		}
		if currentDigest != imageInfo["hash"] {
			// if the hashes don't match, recreate the container with the new one
			err := cli.ContainerRemove(ctx, containerName, types.ContainerRemoveOptions{Force: true})
			if err != nil {
				return containerState, err
			}
			_, err = cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, containerName)
			if err != nil {
				return containerState, err
			}
			err = cli.ContainerStart(ctx, containerName, types.ContainerStartOptions{})
			if err != nil {
				return containerState, err
			}
			msg := fmt.Sprintf("Restarted %s with image %s", containerName, desiredImage)
			logger.Logger.Info(msg)
		}
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

// convert the version info back into json then a map lol
// so we can easily get the correct repo/release channel/tag/hash
func GetLatestContainerInfo(containerType string) (map[string]string, error) {
	var res map[string]string
	// hardcoded llama stuff for testing
	res = make(map[string]string)
	if containerType == "llama-api" {
		res["tag"] = "latest"
		res["hash"] = "b6d21ff8c4d9baad65e1fa741a0f8c898d68735fff3f3cd777e3f0c6a1839dd4"
		res["repo"] = "ghcr.io/abetlen/llama-cpp-python"
		return res, nil
	} else if containerType == "llama-ui" {
		res["tag"] = "latest"
		res["hash"] = "bf4811fe07c11a3a78b760f58b01ee11a61e0e9d6ec8a9e8832d3e14af428200"
		res["repo"] = "nativeplanet/llama-gpt-ui"
		return res, nil
	}
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
				logger.Logger.Info(fmt.Sprintf("Successfully stopped container %s\n", containerName))
				return nil
			}
		}
	}
	return fmt.Errorf("container with name %s not found", containerName)
}

// pull the image if it doesn't exist locally
func PullImageIfNotExist(desiredImage string, imageInfo map[string]string) (bool, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return false, err
	}
	images, err := cli.ImageList(ctx, types.ImageListOptions{})
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
	resp, err := cli.ImagePull(ctx, fmt.Sprintf("%s@sha256:%s", imageInfo["repo"], imageInfo["hash"]), types.ImagePullOptions{})
	if err != nil {
		return false, err
	}
	defer resp.Close()
	io.Copy(ioutil.Discard, resp) // wait until it's done
	return true, nil
}

// looks for a container with the given name and returns it, or nil if not found
func FindContainer(containerName string) (*types.Container, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	// Fetch list of running containers
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
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
	return nil, nil
}

// periodically poll docker in case we miss something
func DockerPoller() {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			logger.Logger.Info("polling docker")
			// todo (maybe not necessary?)
			// fetch the status of all containers and compare with app's state
			// if there's a change, send an event to the EventBus
			return
		}
	}
}

// execute command
func ExecDockerCommand(containerName string, cmd []string) (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return "", err
	}
	// Create an Exec configuration
	execConfig := types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	}
	// Context
	ctx := context.Background()

	// Get container ID by name
	containerID, err := getContainerIDByName(ctx, cli, containerName)
	if err != nil {
		return "", err
	}
	// Create an exec instance, replace 'container_id_here' with your container ID
	resp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", err
	}

	// Start the exec command
	hijackedResp, err := cli.ContainerExecAttach(ctx, resp.ID, types.ExecStartCheck{})
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
func getContainerIDByName(ctx context.Context, cli *client.Client, name string) (string, error) {
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
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
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("Couldn't create client: %v", err)
	}
	defer cli.Close()
	containerID, err := getContainerIDByName(ctx, cli, name)
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
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false, fmt.Errorf("Failed to create client: %v", err)
	}
	volumeList, err := cli.VolumeList(context.Background(), volume.ListOptions{})
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
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("Failed to create client: %v", err)
	}
	networks, err := cli.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		return "", fmt.Errorf("Failed to list networks: %v", err)
	}
	for _, network := range networks {
		if network.Name == networkName {
			return network.ID, nil
		}
	}
	networkResponse, err := cli.NetworkCreate(context.Background(), networkName, types.NetworkCreate{
		Driver: "bridge",
		Scope:  "local",
	})
	if err != nil {
		return "", fmt.Errorf("Failed to create custom bridge network: %v", err)
	}
	return networkResponse.ID, nil
}
