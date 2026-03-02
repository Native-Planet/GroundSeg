package lifecycle

import (
	"errors"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
)

func TestNewRuntimeSetsDefaultsAndAcceptsOverrides(t *testing.T) {
	called := false
	rt := NewRuntime(
		WithDockerClientFactory(func(opts ...client.Opt) (*client.Client, error) {
			called = true
			return nil, errors.New("client not used")
		}),
		WithDockerPollInterval(42),
	)

	if rt.dockerPollInterval != 42 {
		t.Fatalf("expected poll interval 42, got %v", rt.dockerPollInterval)
	}
	if rt.dockerClientNew == nil || rt.containerConfigResolver == nil || rt.getLatestContainerInfoFn == nil || rt.pullImageIfNotExistFn == nil {
		t.Fatalf("expected lifecycle runtime defaults to be configured")
	}
	if rt.getContainerRunningStatusFn == nil || rt.startContainerFn == nil || rt.dockerPollerTickFn == nil {
		t.Fatalf("expected lifecycle runtime hooks to be configured")
	}

	_, _ = rt.dockerClientNew()
	if !called {
		t.Fatalf("expected docker client override to be used")
	}
}

func TestIsContainerLookupNotFoundChecksText(t *testing.T) {
	if !isContainerLookupNotFound(errdefs.NotFound(errors.New("container netdata not found"))) {
		t.Fatal("expected missing-container error to be identified")
	}
	if isContainerLookupNotFound(errors.New("daemon disconnected")) {
		t.Fatal("did not expect non-match error to be treated as not found")
	}
}

func TestContainerPlanForPropagatesResolverErrors(t *testing.T) {
	rt := NewRuntime(
		WithContainerConfigResolver(func(_, _ string) (container.Config, container.HostConfig, error) {
			return container.Config{}, container.HostConfig{}, errors.New("resolver broken")
		}),
	)
	if _, err := rt.containerPlanFor("netdata", "netdata"); err == nil || !strings.Contains(err.Error(), "resolve container config") {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestContainerPlanForPropagatesImageLookupAndPullErrors(t *testing.T) {
	rt := NewRuntime(
		WithContainerConfigResolver(func(_, _ string) (container.Config, container.HostConfig, error) {
			return container.Config{}, container.HostConfig{}, nil
		}),
		WithImageInfoLookup(func(_ string) (map[string]string, error) {
			return nil, errors.New("lookup failed")
		}),
	)
	if _, err := rt.containerPlanFor("netdata", "netdata"); err == nil || !strings.Contains(err.Error(), "lookup latest image") {
		t.Fatalf("expected image lookup error, got %v", err)
	}

	rt = NewRuntime(
		WithContainerConfigResolver(func(_, _ string) (container.Config, container.HostConfig, error) {
			return container.Config{}, container.HostConfig{}, nil
		}),
		WithImageInfoLookup(func(_ string) (map[string]string, error) {
			return map[string]string{"repo": "groundseg", "tag": "v1", "hash": "abc123"}, nil
		}),
		WithImagePuller(func(image string, _ map[string]string) (bool, error) {
			if image != "groundseg:v1@sha256:abc123" {
				t.Fatalf("unexpected desired image: %q", image)
			}
			return false, errors.New("pull failed")
		}),
	)
	if _, err := rt.containerPlanFor("netdata", "netdata"); err == nil || !strings.Contains(err.Error(), "ensure image exists") {
		t.Fatalf("expected pull error, got %v", err)
	}
}

func TestContainerPlanForReturnsExpectedValuesOnSuccess(t *testing.T) {
	rt := NewRuntime(
		WithContainerConfigResolver(func(name, containerType string) (container.Config, container.HostConfig, error) {
			if name != "netdata" || containerType != "netdata" {
				t.Fatalf("unexpected plan args %q %q", name, containerType)
			}
			return container.Config{Image: "groundseg"}, container.HostConfig{}, nil
		}),
		WithImageInfoLookup(func(_ string) (map[string]string, error) {
			return map[string]string{"repo": "groundseg", "tag": "v1", "hash": "abc123"}, nil
		}),
		WithImagePuller(func(_ string, _ map[string]string) (bool, error) {
			return true, nil
		}),
	)

	plan, err := rt.containerPlanFor("netdata", "netdata")
	if err != nil {
		t.Fatalf("unexpected plan error: %v", err)
	}
	if plan.Name != "netdata" || plan.Type != "netdata" || plan.DesiredImage != "groundseg:v1@sha256:abc123" {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if plan.Config.Image != "groundseg" {
		t.Fatalf("expected config image to be propagated")
	}
}

func TestContainerStateFromInspectMapsFields(t *testing.T) {
	state := containerStateFromInspect(
		containerPlan{
			Name:         "netdata",
			Type:         "netdata",
			DesiredImage: "groundseg:v1",
			Config:       container.Config{Hostname: "host"},
			HostConfig:   container.HostConfig{NetworkMode: "bridge"},
		},
		"running",
		container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{
				ID:      "cid",
				Created: "2024-01-01",
				State: &container.State{
					Status: "running",
				},
			},
			Config: &container.Config{},
		},
	)

	if state.ID != "cid" || state.Image != "groundseg:v1" || state.ActualStatus != "running" || state.DesiredStatus != "running" {
		t.Fatalf("unexpected container state: %+v", state)
	}
}

func TestGetContainerImageTagReturnsErrorForClientFailure(t *testing.T) {
	rt := NewRuntime(WithDockerClientFactory(func(opts ...client.Opt) (*client.Client, error) {
		return nil, errors.New("client failed")
	}))
	if _, err := rt.GetContainerImageTag("netdata"); err == nil {
		t.Fatal("expected client creation error")
	}
}

func TestGetContainerRunningStatusReturnsErrorForClientFailure(t *testing.T) {
	rt := NewRuntime(WithDockerClientFactory(func(opts ...client.Opt) (*client.Client, error) {
		return nil, errors.New("client failed")
	}))
	if _, err := rt.GetContainerRunningStatus("netdata"); err == nil {
		t.Fatal("expected client creation error")
	}
}

func TestGetShipStatusReturnsErrorForClientFailure(t *testing.T) {
	rt := NewRuntime(WithDockerClientFactory(func(opts ...client.Opt) (*client.Client, error) {
		return nil, errors.New("client failed")
	}))
	statuses, err := rt.GetShipStatus([]string{"~zod"})
	if err == nil {
		t.Fatal("expected client creation error")
	}
	if len(statuses) != 0 {
		t.Fatalf("expected empty status map, got %d", len(statuses))
	}
}

func TestEnsureMonitoredContainerHealthyUnexpectedError(t *testing.T) {
	rt := NewRuntime(
		WithGetContainerRunningStatus(func(_ string) (string, error) {
			return "", errors.New("daemon disconnected")
		}),
	)
	if err := rt.ensureMonitoredContainerHealthy("netdata", "netdata"); err == nil {
		t.Fatal("expected unexpected status check error to propagate")
	}
}

func TestLifecycleClientErrorPaths(t *testing.T) {
	rt := NewRuntime(WithDockerClientFactory(func(opts ...client.Opt) (*client.Client, error) {
		return nil, errors.New("client failed")
	}))

	if err := rt.StopContainerByName("netdata"); err == nil {
		t.Fatal("expected stop to return client error")
	}
	if err := rt.DeleteContainer("netdata"); err == nil {
		t.Fatal("expected delete to return client error")
	}
	if err := rt.RestartContainer("netdata"); err == nil {
		t.Fatal("expected restart to return client error")
	}
	if _, err := rt.FindContainer("netdata"); err == nil {
		t.Fatal("expected find to return client error")
	}
}

func TestContainsAndPullImageWrapper(t *testing.T) {
	if !Contains([]string{"a", "b", "c"}, "b") {
		t.Fatal("expected contains true for existing value")
	}
	if Contains([]string{"a", "b"}, "z") {
		t.Fatal("expected contains false for missing value")
	}

	called := false
	rt := NewRuntime(WithImagePuller(func(image string, _ map[string]string) (bool, error) {
		called = true
		if image != "groundseg:v1@sha256:abc123" {
			t.Fatalf("unexpected image argument %q", image)
		}
		return true, nil
	}))
	if pulled, err := rt.PullImageIfNotExist("groundseg:v1@sha256:abc123", map[string]string{"repo": "groundseg", "tag": "v1", "hash": "abc123"}); err != nil {
		t.Fatalf("unexpected pull error: %v", err)
	} else if !pulled {
		t.Fatalf("expected pulled=true")
	}
	if !called {
		t.Fatal("expected puller to be invoked")
	}
}
