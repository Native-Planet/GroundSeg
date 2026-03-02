package lifecycle

import (
	"errors"
	"fmt"
	"testing"

	"groundseg/structs"

	"github.com/docker/docker/errdefs"
)

func TestRunDockerPollerTickSkipsStartForHealthyContainer(t *testing.T) {
	rt := NewRuntime(
		WithGetContainerRunningStatus(func(name string) (string, error) {
			if name != "netdata" {
				t.Fatalf("unexpected container query: %q", name)
			}
			return "Up 2 minutes", nil
		}),
		WithStartContainerFn(func(name, containerType string) (structs.ContainerState, error) {
			t.Fatalf("did not expect StartContainer to be called")
			return structs.ContainerState{}, nil
		}),
	)

	if err := rt.runDockerPollerTick(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDockerPollerTickStartsContainerWhenMissing(t *testing.T) {
	started := false
	rt := NewRuntime(
		WithGetContainerRunningStatus(func(name string) (string, error) {
			if name != "netdata" {
				t.Fatalf("unexpected container query: %q", name)
			}
			return "", errdefs.NotFound(errors.New("container with name netdata not found"))
		}),
		WithStartContainerFn(func(name, containerType string) (structs.ContainerState, error) {
			if name != "netdata" || containerType != "netdata" {
				t.Fatalf("unexpected start call: name=%q type=%q", name, containerType)
			}
			started = true
			return structs.ContainerState{Name: name}, nil
		}),
	)

	if err := rt.runDockerPollerTick(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !started {
		t.Fatal("expected netdata container to be started when not found")
	}
}

func TestRunDockerPollerTickReturnsErrorForStatusFailure(t *testing.T) {
	started := false
	rt := NewRuntime(
		WithGetContainerRunningStatus(func(name string) (string, error) {
			if name != "netdata" {
				t.Fatalf("unexpected container query: %q", name)
			}
			return "", fmt.Errorf("unexpected daemon error")
		}),
		WithStartContainerFn(func(name string, containerType string) (structs.ContainerState, error) {
			started = true
			return structs.ContainerState{}, nil
		}),
	)

	if err := rt.runDockerPollerTick(); err == nil {
		t.Fatal("expected poller tick error when status lookup fails with unexpected error")
	}
	if started {
		t.Fatal("did not expect start attempt on unexpected status lookup failure")
	}
}

func TestEnsureMonitoredContainerHealthyRestartsStoppedContainer(t *testing.T) {
	started := false
	rt := NewRuntime(
		WithGetContainerRunningStatus(func(name string) (string, error) {
			return "Exited (0)", nil
		}),
		WithStartContainerFn(func(name string, containerType string) (structs.ContainerState, error) {
			started = true
			return structs.ContainerState{Name: name}, nil
		}),
	)

	if err := rt.runDockerPollerTick(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !started {
		t.Fatal("expected start to be attempted for non-running monitored container")
	}
}
