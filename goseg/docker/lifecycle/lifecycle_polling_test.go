package lifecycle

import (
	"fmt"
	"testing"

	"groundseg/structs"
)

func TestRunDockerPollerTickSkipsStartForHealthyContainer(t *testing.T) {
	origGetContainerRunningStatusFn := getContainerRunningStatusFn
	origStartContainerFn := startContainerFn
	t.Cleanup(func() {
		getContainerRunningStatusFn = origGetContainerRunningStatusFn
		startContainerFn = origStartContainerFn
	})

	getContainerRunningStatusFn = func(name string) (string, error) {
		if name != "netdata" {
			t.Fatalf("unexpected container query: %q", name)
		}
		return "Up 2 minutes", nil
	}
	startCalled := false
	startContainerFn = func(name string, containerType string) (structs.ContainerState, error) {
		startCalled = true
		if name != "netdata" || containerType != "netdata" {
			t.Fatalf("unexpected start call: name=%q type=%q", name, containerType)
		}
		return structs.ContainerState{}, nil
	}

	if err := runDockerPollerTick(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if startCalled {
		t.Fatal("did not expect netdata container to be restarted when status is healthy")
	}
}

func TestRunDockerPollerTickStartsContainerWhenMissing(t *testing.T) {
	origGetContainerRunningStatusFn := getContainerRunningStatusFn
	origStartContainerFn := startContainerFn
	t.Cleanup(func() {
		getContainerRunningStatusFn = origGetContainerRunningStatusFn
		startContainerFn = origStartContainerFn
	})

	getContainerRunningStatusFn = func(name string) (string, error) {
		if name != "netdata" {
			t.Fatalf("unexpected container query: %q", name)
		}
		return "", fmt.Errorf("container with name %s not found", name)
	}
	startCalled := false
	startContainerFn = func(name string, containerType string) (structs.ContainerState, error) {
		startCalled = true
		if name != "netdata" || containerType != "netdata" {
			t.Fatalf("unexpected start call: name=%q type=%q", name, containerType)
		}
		return structs.ContainerState{Name: name}, nil
	}

	if err := runDockerPollerTick(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !startCalled {
		t.Fatal("expected netdata container to be started when not found")
	}
}

func TestRunDockerPollerTickReturnsErrorForStatusFailure(t *testing.T) {
	origGetContainerRunningStatusFn := getContainerRunningStatusFn
	origStartContainerFn := startContainerFn
	t.Cleanup(func() {
		getContainerRunningStatusFn = origGetContainerRunningStatusFn
		startContainerFn = origStartContainerFn
	})

	getContainerRunningStatusFn = func(name string) (string, error) {
		if name != "netdata" {
			t.Fatalf("unexpected container query: %q", name)
		}
		return "", fmt.Errorf("unexpected daemon error")
	}
	startCalled := false
	startContainerFn = func(name string, containerType string) (structs.ContainerState, error) {
		startCalled = true
		return structs.ContainerState{}, nil
	}

	if err := runDockerPollerTick(); err == nil {
		t.Fatal("expected poller tick error when status lookup fails with unexpected error")
	}
	if startCalled {
		t.Fatal("did not expect start attempt on unexpected status lookup failure")
	}
}

func TestEnsureMonitoredContainerHealthyRestartsStoppedContainer(t *testing.T) {
	origGetContainerRunningStatusFn := getContainerRunningStatusFn
	origStartContainerFn := startContainerFn
	t.Cleanup(func() {
		getContainerRunningStatusFn = origGetContainerRunningStatusFn
		startContainerFn = origStartContainerFn
	})

	getContainerRunningStatusFn = func(name string) (string, error) {
		return "Exited (0)", nil
	}
	startCalled := false
	startContainerFn = func(name string, containerType string) (structs.ContainerState, error) {
		startCalled = true
		return structs.ContainerState{Name: name}, nil
	}

	if err := runDockerPollerTick(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !startCalled {
		t.Fatal("expected start to be attempted for non-running monitored container")
	}
}
