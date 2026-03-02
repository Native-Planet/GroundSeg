package lifecycle

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"go.uber.org/zap"

	"groundseg/docker/network"
	"groundseg/docker/registry"
	"groundseg/dockerclient"
	"groundseg/structs"
)

type Runtime struct {
	dockerClientNew             func(...client.Opt) (*client.Client, error)
	containerConfigResolver     containerConfigResolverFn
	getLatestContainerInfoFn    imageInfoLookupFn
	pullImageIfNotExistFn       imagePullerFn
	dockerPollInterval          time.Duration
	getContainerRunningStatusFn func(string) (string, error)
	startContainerFn            func(string, string) (structs.ContainerState, error)
	dockerPollerTickFn          func() error
}

type RuntimeOption func(*Runtime)

func WithDockerClientFactory(factory func(...client.Opt) (*client.Client, error)) RuntimeOption {
	return func(rt *Runtime) {
		rt.dockerClientNew = factory
	}
}

func WithContainerConfigResolver(resolver containerConfigResolverFn) RuntimeOption {
	return func(rt *Runtime) {
		rt.containerConfigResolver = resolver
	}
}

func WithImageInfoLookup(fn imageInfoLookupFn) RuntimeOption {
	return func(rt *Runtime) {
		rt.getLatestContainerInfoFn = fn
	}
}

func WithImagePuller(fn imagePullerFn) RuntimeOption {
	return func(rt *Runtime) {
		rt.pullImageIfNotExistFn = fn
	}
}

func WithGetContainerRunningStatus(fn func(string) (string, error)) RuntimeOption {
	return func(rt *Runtime) {
		rt.getContainerRunningStatusFn = fn
	}
}

func WithStartContainerFn(fn func(string, string) (structs.ContainerState, error)) RuntimeOption {
	return func(rt *Runtime) {
		rt.startContainerFn = fn
	}
}

func WithDockerPollerTickFn(fn func() error) RuntimeOption {
	return func(rt *Runtime) {
		rt.dockerPollerTickFn = fn
	}
}

func WithDockerPollInterval(interval time.Duration) RuntimeOption {
	return func(rt *Runtime) {
		rt.dockerPollInterval = interval
	}
}

func NewRuntime(opts ...RuntimeOption) *Runtime {
	rt := &Runtime{
		dockerClientNew: dockerclient.New,
		containerConfigResolver: func(containerName string, containerType string) (container.Config, container.HostConfig, error) {
			return container.Config{}, container.HostConfig{}, fmt.Errorf("container config resolver is not configured")
		},
		getLatestContainerInfoFn: registry.GetLatestContainerInfo,
		pullImageIfNotExistFn:    registry.PullImageIfNotExist,
		dockerPollInterval:       10 * time.Second,
	}
	rt.getContainerRunningStatusFn = rt.GetContainerRunningStatus
	rt.startContainerFn = rt.StartContainer
	rt.dockerPollerTickFn = rt.runDockerPollerTick
	for _, opt := range opts {
		if opt != nil {
			opt(rt)
		}
	}
	return rt
}

var DefaultRuntime = NewRuntime()

type containerConfigResolverFn func(string, string) (container.Config, container.HostConfig, error)

type imageInfoLookupFn func(string) (map[string]string, error)

type imagePullerFn func(string, map[string]string) (bool, error)

func (runtime *Runtime) Initialize() error {
	_, err := runtime.FindContainer("groundseg-webui")
	if err == nil {
		if err = runtime.StopContainerByName("groundseg-webui"); err != nil {
			zap.L().Warn(fmt.Sprintf("Couldn't stop old webui container: %v", err))
		}
	}
	if err = network.NewNetworkRuntime().KillContainerUsingPort(80); err != nil {
		zap.L().Warn(fmt.Sprintf("Couldn't stop container on port 80: %v", err))
	}

	cli, err := runtime.dockerClientNew()
	if err != nil {
		return fmt.Errorf("error creating docker client: %w", err)
	}
	defer cli.Close()
	version, err := cli.ServerVersion(context.TODO())
	if err != nil {
		return fmt.Errorf("error getting docker version: %w", err)
	}
	zap.L().Info(fmt.Sprintf("Docker version: %s", version.Version))
	return nil
}

func (runtime *Runtime) GetShipStatus(patps []string) (map[string]string, error) {
	statuses := make(map[string]string)
	cli, err := runtime.dockerClientNew()
	if err != nil {
		errmsg := fmt.Errorf("unable to create docker client: %w", err)
		zap.L().Error(errmsg.Error())
		return statuses, errmsg
	}
	defer cli.Close()
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		errmsg := fmt.Errorf("failed to list containers: %w", err)
		zap.L().Error(errmsg.Error())
		return statuses, errmsg
	}

	statusIndex := NewContainerStatusIndex(containers)
	return ResolveStatuses(statusIndex, patps), nil
}

// GetContainerImageTag returns the image tag for a container name if the container exists.
func (runtime *Runtime) GetContainerImageTag(containerName string) (string, error) {
	ctx := context.Background()
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return "", fmt.Errorf("unable to create docker client: %w", err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}
	for _, cont := range containers {
		for _, name := range cont.Names {
			if strings.TrimPrefix(name, "/") == containerName {
				if tag := imageTagFromReference(cont.Image); tag != "" {
					return tag, nil
				}
				return "latest", nil
			}
		}
	}
	return "", fmt.Errorf("no exact match found for container %s", containerName)
}

func imageTagFromReference(image string) string {
	imageWithoutDigest := strings.SplitN(image, "@", 2)[0]
	lastColon := strings.LastIndex(imageWithoutDigest, ":")
	if lastColon == -1 {
		return ""
	}
	lastSlash := strings.LastIndex(imageWithoutDigest, "/")
	if lastSlash > lastColon {
		return ""
	}
	return imageWithoutDigest[lastColon+1:]
}

// GetContainerRunningStatus returns status for a container by exact name.
func (runtime *Runtime) GetContainerRunningStatus(containerName string) (string, error) {
	var status string
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return status, fmt.Errorf("unable to create docker client: %w", err)
	}
	defer cli.Close()
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		return status, fmt.Errorf("failed to list containers: %w", err)
	}
	for _, cont := range containers {
		for _, name := range cont.Names {
			if name == "/"+containerName {
				return cont.Status, nil
			}
		}
	}
	return status, fmt.Errorf("unable to get container running status for %s", containerName)
}

type containerPlan struct {
	Name         string
	Type         string
	Config       container.Config
	HostConfig   container.HostConfig
	ImageInfo    map[string]string
	DesiredImage string
}

func (runtime *Runtime) containerPlanFor(containerName string, containerType string) (containerPlan, error) {
	plan := containerPlan{Name: containerName, Type: containerType}
	containerConfig, hostConfig, err := runtime.containerConfigResolver(containerName, containerType)
	if err != nil {
		return plan, fmt.Errorf("resolve container config for %s/%s: %w", containerName, containerType, err)
	}
	plan.Config = containerConfig
	plan.HostConfig = hostConfig

	imageInfo, err := runtime.getLatestContainerInfoFn(containerType)
	if err != nil {
		return plan, fmt.Errorf("lookup latest image for container type %s: %w", containerType, err)
	}
	plan.ImageInfo = imageInfo
	plan.DesiredImage = fmt.Sprintf("%s:%s@sha256:%s", imageInfo["repo"], imageInfo["tag"], imageInfo["hash"])
	if _, err := runtime.pullImageIfNotExistFn(plan.DesiredImage, imageInfo); err != nil {
		return plan, fmt.Errorf("ensure image exists %s: %w", plan.DesiredImage, err)
	}
	return plan, nil
}

func createAndStartContainer(ctx context.Context, cli *client.Client, plan containerPlan) error {
	if _, err := cli.ContainerCreate(ctx, &plan.Config, &plan.HostConfig, nil, nil, plan.Name); err != nil {
		return fmt.Errorf("create container %s: %w", plan.Name, err)
	}
	if err := cli.ContainerStart(ctx, plan.Name, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container %s: %w", plan.Name, err)
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
		zap.L().Warn(fmt.Sprintf("Couldn't remove container %v (may not exist yet): %v", plan.Name, err))
	}
	if err := createAndStartContainer(ctx, cli, plan); err != nil {
		return fmt.Errorf("recreate container %s: %w", plan.Name, err)
	}
	zap.L().Info(fmt.Sprintf("Restarted %s with image %s", plan.Name, plan.DesiredImage))
	return nil
}

func ensureRunningContainer(runtime *Runtime, ctx context.Context, cli *client.Client, plan containerPlan) error {
	existingContainer, err := runtime.FindContainer(plan.Name)
	if err != nil {
		if !isContainerLookupNotFound(err) {
			return fmt.Errorf("lookup container %s: %w", plan.Name, err)
		}
		existingContainer = nil
	}
	switch {
	case existingContainer == nil:
		if err := createAndStartContainer(ctx, cli, plan); err != nil {
			return fmt.Errorf("start container %s: %w", plan.Name, err)
		}
		zap.L().Info(fmt.Sprintf("%s started with image %s", plan.Name, plan.DesiredImage))
	case existingContainer.State == "exited":
		if err := cli.ContainerRemove(ctx, plan.Name, container.RemoveOptions{Force: true}); err != nil {
			return fmt.Errorf("remove exited container %s: %w", plan.Name, err)
		}
		if err := createAndStartContainer(ctx, cli, plan); err != nil {
			return fmt.Errorf("restart exited container %s: %w", plan.Name, err)
		}
		zap.L().Info(fmt.Sprintf("Started stopped container %s", plan.Name))
	case existingContainer.State == "created":
		if err := cli.ContainerStart(ctx, plan.Name, container.StartOptions{}); err != nil {
			return fmt.Errorf("start created container %s: %w", plan.Name, err)
		}
		zap.L().Info(fmt.Sprintf("Started created container %s", plan.Name))
	default:
		if err := recreateContainerIfImageChanged(ctx, cli, plan, existingContainer.Image); err != nil {
			return fmt.Errorf("reconcile existing container %s: %w", plan.Name, err)
		}
	}
	return nil
}

func ensureCreatedContainer(ctx context.Context, cli *client.Client, plan containerPlan) error {
	_, err := cli.ContainerCreate(ctx, &plan.Config, &plan.HostConfig, nil, nil, plan.Name)
	if err != nil {
		return fmt.Errorf("create container %s: %w", plan.Name, err)
	}
	return nil
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

func (runtime *Runtime) StartContainer(containerName string, containerType string) (structs.ContainerState, error) {
	zap.L().Debug(fmt.Sprintf("StartContainer issued for %v", containerName))
	var containerState structs.ContainerState
	if err := runtime.cleanupMinIOContainerForStart(containerName, containerType); err != nil {
		return containerState, fmt.Errorf("cleanup container %s before start: %w", containerName, err)
	}
	plan, err := runtime.containerPlanFor(containerName, containerType)
	if err != nil {
		return containerState, fmt.Errorf("build container plan for %s/%s: %w", containerName, containerType, err)
	}
	ctx := context.Background()
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return containerState, fmt.Errorf("create docker client for %s: %w", containerName, err)
	}
	defer cli.Close()
	if err := ensureRunningContainer(runtime, ctx, cli, plan); err != nil {
		return containerState, fmt.Errorf("start container %s: %w", containerName, err)
	}
	containerDetails, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return containerState, fmt.Errorf("failed to inspect container %s: %w", containerName, err)
	}
	containerState = containerStateFromInspect(plan, "running", containerDetails)
	return containerState, err
}

func (runtime *Runtime) CreateContainer(containerName string, containerType string) (structs.ContainerState, error) {
	var containerState structs.ContainerState
	if err := runtime.cleanupMinIOContainerForStart(containerName, containerType); err != nil {
		return containerState, fmt.Errorf("cleanup container %s before create: %w", containerName, err)
	}
	plan, err := runtime.containerPlanFor(containerName, containerType)
	if err != nil {
		return containerState, fmt.Errorf("build container plan for %s/%s: %w", containerName, containerType, err)
	}
	ctx := context.Background()
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return containerState, fmt.Errorf("create docker client for %s: %w", containerName, err)
	}
	defer cli.Close()
	if err := ensureCreatedContainer(ctx, cli, plan); err != nil {
		return containerState, fmt.Errorf("create container %s: %w", containerName, err)
	}
	containerDetails, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return containerState, fmt.Errorf("failed to inspect container %s: %w", containerName, err)
	}
	containerState = containerStateFromInspect(plan, "stopped", containerDetails)
	return containerState, err
}

func (runtime *Runtime) StopContainerByName(containerName string) error {
	ctx := context.Background()
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return fmt.Errorf("unable to create docker client: %w", err)
	}
	defer cli.Close()
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}
	for _, cont := range containers {
		for _, name := range cont.Names {
			if name == "/"+containerName {
				options := container.StopOptions{}
				if err := cli.ContainerStop(ctx, cont.ID, options); err != nil {
					return fmt.Errorf("failed to stop container %s: %w", containerName, err)
				}
				zap.L().Info(fmt.Sprintf("Successfully stopped container %s\n", containerName))
				return nil
			}
		}
	}
	return errdefs.NotFound(fmt.Errorf("container %s not found", containerName))
}

func (runtime *Runtime) DeleteContainer(name string) error {
	cli, err := runtime.dockerClientNew()
	if err != nil {
		errmsg := fmt.Errorf("Failed to create docker client %s: %w", name, err)
		return errmsg
	}
	defer cli.Close()
	if err := cli.ContainerRemove(context.Background(), name, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("Failed to delete docker container %s: %w", name, err)
	}
	zap.L().Info(fmt.Sprintf("Deleted Container: %s", name))
	return nil
}

func (runtime *Runtime) cleanupMinIOContainerForStart(containerName string, containerType string) error {
	if containerType != "minio" {
		return nil
	}
	existingContainer, err := runtime.FindContainer(containerName)
	if err != nil {
		if isContainerLookupNotFound(err) {
			return nil
		}
		return fmt.Errorf("find minio container %s for cleanup: %w", containerName, err)
	}
	if existingContainer == nil {
		return nil
	}
	return runtime.DeleteContainer(containerName)
}

func isContainerLookupNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errdefs.IsNotFound(err)
}

// ExecDockerCommand executes a command inside a container and returns stdout/stderr plus the exit code.
func (runtime *Runtime) ExecDockerCommand(containerName string, cmd []string) (string, int, error) {
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return "", -1, fmt.Errorf("unable to create docker client: %w", err)
	}
	defer cli.Close()

	execConfig := container.ExecOptions{AttachStdout: true, AttachStderr: true, Cmd: cmd}
	ctx := context.Background()

	containerID, err := GetContainerIDByName(ctx, cli, containerName)
	if err != nil {
		return "", -1, fmt.Errorf("lookup container %s: %w", containerName, err)
	}

	resp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", -1, fmt.Errorf("failed to create exec for container %s: %w", containerName, err)
	}
	hijackedResp, err := cli.ContainerExecAttach(ctx, resp.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", -1, fmt.Errorf("failed to attach to exec for container %s: %w", containerName, err)
	}
	defer hijackedResp.Close()

	output, err := ioutil.ReadAll(hijackedResp.Reader)
	if err != nil {
		return "", -1, fmt.Errorf("read exec output from %s: %w", containerName, err)
	}
	execID := resp.ID
	deadline := time.Now().Add(30 * time.Second)
	for {
		execState, inspectErr := cli.ContainerExecInspect(ctx, execID)
		if inspectErr != nil {
			return string(output), -1, fmt.Errorf("failed to inspect exec command %s in %s: %w", execID, containerName, inspectErr)
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
		return "", fmt.Errorf("failed to list containers: %w", err)
	}
	for _, cont := range containers {
		for _, n := range cont.Names {
			if n == "/"+name {
				return cont.ID, nil
			}
		}
	}
	return "", errdefs.NotFound(fmt.Errorf("container %s not found", name))
}

func (runtime *Runtime) RestartContainer(name string) error {
	ctx := context.Background()
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return fmt.Errorf("Couldn't create client: %w", err)
	}
	defer cli.Close()

	containerID, err := GetContainerIDByName(ctx, cli, name)
	if err != nil {
		return fmt.Errorf("Couldn't get ID for %s: %w", name, err)
	}
	timeout := 30
	stopOptions := container.StopOptions{Timeout: &timeout}
	if err := cli.ContainerRestart(ctx, containerID, stopOptions); err != nil {
		return fmt.Errorf("Couldn't restart container %s: %w", name, err)
	}
	return nil
}

func (runtime *Runtime) FindContainer(containerName string) (*container.Summary, error) {
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.TrimPrefix(name, "/") == containerName {
				return &container, nil
			}
		}
	}
	return nil, errdefs.NotFound(fmt.Errorf("container %s not found", containerName))
}

func (runtime *Runtime) DockerPoller() {
	ticker := time.NewTicker(runtime.dockerPollInterval)
	defer ticker.Stop()
	for {
		<-ticker.C
		zap.L().Info("polling docker")
		if err := runtime.dockerPollerTickFn(); err != nil {
			zap.L().Error(fmt.Sprintf("Docker poller tick failed: %v", err))
		}
	}
}

func (runtime *Runtime) runDockerPollerTick() error {
	if err := runtime.ensureMonitoredContainerHealthy("netdata", "netdata"); err != nil {
		return err
	}
	return nil
}

func (runtime *Runtime) ensureMonitoredContainerHealthy(containerName, containerType string) error {
	status, err := runtime.getContainerRunningStatusFn(containerName)
	if err == nil {
		if strings.HasPrefix(status, "Up") {
			return nil
		}
		zap.L().Warn(fmt.Sprintf("Container %s is not running (%q); attempting restart", containerName, status))
		if _, err := runtime.startContainerFn(containerName, containerType); err != nil {
			return fmt.Errorf("failed to restart container %s: %w", containerName, err)
		}
		return nil
	}
	if !isContainerLookupNotFound(err) {
		return fmt.Errorf("container status check failed for %s: %w", containerName, err)
	}
	zap.L().Info(fmt.Sprintf("Container %s is not found; attempting start", containerName))
	if _, err := runtime.startContainerFn(containerName, containerType); err != nil {
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

func (runtime *Runtime) PullImageIfNotExist(desiredImage string, imageInfo map[string]string) (bool, error) {
	return runtime.pullImageIfNotExistFn(desiredImage, imageInfo)
}
