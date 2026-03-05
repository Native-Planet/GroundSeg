package lifecycle

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"groundseg/docker/network"
	"groundseg/docker/registry"
	"groundseg/dockerclient"
	"groundseg/structs"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
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

var defaultRuntimeValue atomic.Pointer[Runtime]

func init() {
	defaultRuntimeValue.Store(NewRuntime())
}

func DefaultRuntime() *Runtime {
	runtime := defaultRuntimeValue.Load()
	if runtime == nil {
		fallback := NewRuntime()
		defaultRuntimeValue.Store(fallback)
		return fallback
	}
	return runtime
}

func SetDefaultRuntime(runtime *Runtime) {
	if runtime == nil {
		runtime = NewRuntime()
	}
	defaultRuntimeValue.Store(runtime)
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

type containerConfigResolverFn func(string, string) (container.Config, container.HostConfig, error)

type imageInfoLookupFn func(string) (registry.ImageDescriptor, error)

type imagePullerFn func(string, registry.ImageDescriptor) (bool, error)

func (runtime *Runtime) Initialize() (err error) {
	_, err = runtime.FindContainer("groundseg-webui")
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
	defer closeRuntimeDockerClient(cli, "initialize lifecycle", &err)
	version, err := cli.ServerVersion(context.TODO())
	if err != nil {
		return fmt.Errorf("error getting docker version: %w", err)
	}
	zap.L().Info(fmt.Sprintf("Docker version: %s", version.Version))
	return nil
}

func (runtime *Runtime) PullImageIfNotExist(desiredImage string, imageInfo registry.ImageDescriptor) (bool, error) {
	return runtime.pullImageIfNotExistFn(desiredImage, imageInfo)
}

func Contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}
