package lifecycle

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.uber.org/zap"

	"groundseg/docker/network"
	"groundseg/docker/registry"
	"groundseg/dockerclient"
	"groundseg/structs"
)

var (
	dockerClientNew         = dockerclient.New
	containerConfigResolver = func(containerName string, containerType string) (container.Config, container.HostConfig, error) {
		return container.Config{}, container.HostConfig{}, fmt.Errorf("container config resolver is not configured")
	}
	getLatestContainerInfoFn    = registry.GetLatestContainerInfo
	pullImageIfNotExistFn       = registry.PullImageIfNotExist
	dockerPollInterval          = 10 * time.Second
	getContainerRunningStatusFn = GetContainerRunningStatus
	startContainerFn            = StartContainer
	dockerPollerTickFn          = runDockerPollerTick
)

type containerConfigResolverFn func(string, string) (container.Config, container.HostConfig, error)

type imageInfoLookupFn func(string) (map[string]string, error)

type imagePullerFn func(string, map[string]string) (bool, error)

func SetClientFactory(factory func(...client.Opt) (*client.Client, error)) {
	if factory == nil {
		dockerClientNew = dockerclient.New
		return
	}
	dockerClientNew = factory
}

func SetContainerConfigResolver(resolver containerConfigResolverFn) {
	if resolver != nil {
		containerConfigResolver = resolver
	}
}

func SetImageInfoLookup(fn imageInfoLookupFn) {
	if fn != nil {
		getLatestContainerInfoFn = fn
	}
}

func SetImagePuller(fn imagePullerFn) {
	if fn != nil {
		pullImageIfNotExistFn = fn
	}
}

func Initialize() error {
	_, err := FindContainer("groundseg-webui")
	if err == nil {
		if err = StopContainerByName("groundseg-webui"); err != nil {
			zap.L().Warn(fmt.Sprintf("Couldn't stop old webui container: %v", err))
		}
	}
	if err = network.KillContainerUsingPort(80); err != nil {
		zap.L().Warn(fmt.Sprintf("Couldn't stop container on port 80: %v", err))
	}

	cli, err := dockerClientNew()
	if err != nil {
		return fmt.Errorf("error creating docker client: %v", err)
	}
	defer cli.Close()
	version, err := cli.ServerVersion(context.TODO())
	if err != nil {
		return fmt.Errorf("error getting docker version: %v", err)
	}
	zap.L().Info(fmt.Sprintf("Docker version: %s", version.Version))
	return nil
}

func GetShipStatus(patps []string) (map[string]string, error) {
	statuses := make(map[string]string)
	cli, err := dockerClientNew()
	if err != nil {
		errmsg := fmt.Sprintf("Error getting Docker info: %v", err)
		zap.L().Error(errmsg)
		return statuses, err
	}
	defer cli.Close()
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		errmsg := fmt.Sprintf("Error getting containers: %v", err)
		zap.L().Error(errmsg)
		return statuses, err
	}

	for _, pier := range patps {
		found := false
		for _, cont := range containers {
			for _, name := range cont.Names {
				fasPier := "/" + pier
				if name == fasPier {
					statuses[pier] = cont.Status
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
	return statuses, nil
}

// GetContainerImageTag returns the image tag for a container name if the container exists.
func GetContainerImageTag(containerName string) (string, error) {
	ctx := context.Background()
	cli, err := dockerClientNew()
	if err != nil {
		return "", fmt.Errorf("unable to create docker client: %v", err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %v", err)
	}
	for _, cont := range containers {
		for _, name := range cont.Names {
			if strings.TrimPrefix(name, "/") == containerName {
				imageParts := strings.Split(cont.Image, ":")
				if len(imageParts) > 1 {
					return strings.Split(imageParts[1], "@")[0], nil
				}
				return "latest", nil
			}
		}
	}
	return "", fmt.Errorf("no exact match found for container %s", containerName)
}

// GetContainerRunningStatus returns status for a container by exact name.
func GetContainerRunningStatus(containerName string) (string, error) {
	var status string
	cli, err := dockerClientNew()
	if err != nil {
		return status, fmt.Errorf("unable to create docker client: %v", err)
	}
	defer cli.Close()
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		return status, err
	}
	for _, cont := range containers {
		for _, name := range cont.Names {
			if name == "/"+containerName {
				return cont.Status, nil
			}
		}
	}
	return status, fmt.Errorf("Unable to get container running status: %v", containerName)
}

type containerPlan struct {
	Name         string
	Type         string
	Config       container.Config
	HostConfig   container.HostConfig
	ImageInfo    map[string]string
	DesiredImage string
}

func containerPlanFor(containerName string, containerType string) (containerPlan, error) {
	plan := containerPlan{Name: containerName, Type: containerType}
	containerConfig, hostConfig, err := containerConfigResolver(containerName, containerType)
	if err != nil {
		return plan, err
	}
	plan.Config = containerConfig
	plan.HostConfig = hostConfig

	imageInfo, err := getLatestContainerInfoFn(containerType)
	if err != nil {
		return plan, err
	}
	plan.ImageInfo = imageInfo
	plan.DesiredImage = fmt.Sprintf("%s:%s@sha256:%s", imageInfo["repo"], imageInfo["tag"], imageInfo["hash"])
	if _, err := pullImageIfNotExistFn(plan.DesiredImage, imageInfo); err != nil {
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
	existingContainer, err := FindContainer(plan.Name)
	if err != nil {
		if !isContainerLookupNotFound(err) {
			return err
		}
		existingContainer = nil
	}
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
		ID:            containerDetails.ID,
		Name:          plan.Name,
		Image:         plan.DesiredImage,
		Type:          plan.Type,
		DesiredStatus: desiredStatus,
		ActualStatus:  containerDetails.State.Status,
		CreatedAt:     containerDetails.Created,
		Config:        plan.Config,
		Host:          plan.HostConfig,
	}
}

func StartContainer(containerName string, containerType string) (structs.ContainerState, error) {
	zap.L().Debug(fmt.Sprintf("StartContainer issued for %v", containerName))
	var containerState structs.ContainerState
	if err := cleanupMinIOContainerForStart(containerName, containerType); err != nil {
		return containerState, err
	}
	plan, err := containerPlanFor(containerName, containerType)
	if err != nil {
		return containerState, err
	}
	ctx := context.Background()
	cli, err := dockerClientNew()
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
	containerState = containerStateFromInspect(plan, "running", containerDetails)
	return containerState, err
}

func CreateContainer(containerName string, containerType string) (structs.ContainerState, error) {
	var containerState structs.ContainerState
	if err := cleanupMinIOContainerForStart(containerName, containerType); err != nil {
		return containerState, err
	}
	plan, err := containerPlanFor(containerName, containerType)
	if err != nil {
		return containerState, err
	}
	ctx := context.Background()
	cli, err := dockerClientNew()
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

// StopContainerByName stops a container by name.
func StopContainerByName(containerName string) error {
	ctx := context.Background()
	cli, err := dockerClientNew()
	if err != nil {
		return err
	}
	defer cli.Close()
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return err
	}
	for _, cont := range containers {
		for _, name := range cont.Names {
			if name == "/"+containerName {
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

// DeleteContainer deletes a container by its name.
func DeleteContainer(name string) error {
	cli, err := dockerClientNew()
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker client: %v : %v", name, err)
		return errmsg
	}
	defer cli.Close()
	if err := cli.ContainerRemove(context.Background(), name, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("Failed to delete docker container: %v : %v", name, err)
	}
	zap.L().Info(fmt.Sprintf("Deleted Container: %s", name))
	return nil
}

func cleanupMinIOContainerForStart(containerName string, containerType string) error {
	if containerType != "minio" {
		return nil
	}
	existingContainer, err := FindContainer(containerName)
	if err != nil {
		if isContainerLookupNotFound(err) {
			return nil
		}
		return err
	}
	if existingContainer == nil {
		return nil
	}
	return DeleteContainer(containerName)
}

func isContainerLookupNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

// ExecDockerCommand executes a command inside a container and returns stdout/stderr plus the exit code.
func ExecDockerCommand(containerName string, cmd []string) (string, int, error) {
	cli, err := dockerClientNew()
	if err != nil {
		return "", -1, err
	}
	defer cli.Close()

	execConfig := container.ExecOptions{AttachStdout: true, AttachStderr: true, Cmd: cmd}
	ctx := context.Background()

	containerID, err := GetContainerIDByName(ctx, cli, containerName)
	if err != nil {
		return "", -1, err
	}

	resp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", -1, err
	}
	hijackedResp, err := cli.ContainerExecAttach(ctx, resp.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", -1, err
	}
	defer hijackedResp.Close()

	output, err := ioutil.ReadAll(hijackedResp.Reader)
	if err != nil {
		return "", -1, err
	}
	execID := resp.ID
	deadline := time.Now().Add(30 * time.Second)
	for {
		execState, inspectErr := cli.ContainerExecInspect(ctx, execID)
		if inspectErr != nil {
			return string(output), -1, fmt.Errorf("failed to inspect exec command %s in %s: %v", execID, containerName, inspectErr)
		}
		if !execState.Running && execState.ExitCode != 0 {
			return string(output), execState.ExitCode, fmt.Errorf("command failed with exit code %d: %s", execState.ExitCode, strings.TrimSpace(string(output)))
		}
		if !execState.Running && execState.ExitCode == 0 {
			return string(output), 0, nil
		}
		if time.Now().After(deadline) {
			return string(output), -1, fmt.Errorf("timed out waiting for command completion in %s", containerName)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func GetContainerIDByName(ctx context.Context, cli *client.Client, name string) (string, error) {
	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, cont := range containers {
		for _, n := range cont.Names {
			if n == "/"+name {
				return cont.ID, nil
			}
		}
	}
	return "", fmt.Errorf("Container not found")
}

// RestartContainer restarts a running container.
func RestartContainer(name string) error {
	ctx := context.Background()
	cli, err := dockerClientNew()
	if err != nil {
		return fmt.Errorf("Couldn't create client: %v", err)
	}
	defer cli.Close()

	containerID, err := GetContainerIDByName(ctx, cli, name)
	if err != nil {
		return fmt.Errorf("Couldn't get ID: %v", err)
	}
	timeout := 30
	stopOptions := container.StopOptions{Timeout: &timeout}
	if err := cli.ContainerRestart(ctx, containerID, stopOptions); err != nil {
		return fmt.Errorf("Couldn't restart container: %v", err)
	}
	return nil
}

func FindContainer(containerName string) (*container.Summary, error) {
	cli, err := dockerClientNew()
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.TrimPrefix(name, "/") == containerName {
				return &container, nil
			}
		}
	}
	return nil, fmt.Errorf("Container %v not found", containerName)
}

func DockerPoller() {
	ticker := time.NewTicker(dockerPollInterval)
	defer ticker.Stop()
	for {
		<-ticker.C
		zap.L().Info("polling docker")
		if err := dockerPollerTickFn(); err != nil {
			zap.L().Error(fmt.Sprintf("Docker poller tick failed: %v", err))
		}
	}
}

func runDockerPollerTick() error {
	if err := ensureMonitoredContainerHealthy("netdata", "netdata"); err != nil {
		return err
	}
	return nil
}

func ensureMonitoredContainerHealthy(containerName, containerType string) error {
	status, err := getContainerRunningStatusFn(containerName)
	if err == nil {
		if strings.HasPrefix(status, "Up") {
			return nil
		}
		zap.L().Warn(fmt.Sprintf("Container %s is not running (%q); attempting restart", containerName, status))
		if _, err := startContainerFn(containerName, containerType); err != nil {
			return fmt.Errorf("failed to restart container %s: %w", containerName, err)
		}
		return nil
	}
	if !isContainerLookupNotFound(err) {
		return fmt.Errorf("container status check failed for %s: %w", containerName, err)
	}
	zap.L().Info(fmt.Sprintf("Container %s is not found; attempting start", containerName))
	if _, err := startContainerFn(containerName, containerType); err != nil {
		return fmt.Errorf("failed to start container %s: %w", containerName, err)
	}
	return nil
}

func Contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

func PullImageIfNotExist(desiredImage string, imageInfo map[string]string) (bool, error) {
	return pullImageIfNotExistFn(desiredImage, imageInfo)
}
