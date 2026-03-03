package dockerclient

import (
	"sync"
	"testing"

	"github.com/docker/docker/api"
	docker "github.com/docker/docker/client"
)

func TestFromEnvWithoutAPIVersionIgnoresEnvOverride(t *testing.T) {
	t.Setenv(docker.EnvOverrideAPIVersion, "1.43")

	cli, err := docker.NewClientWithOpts(fromEnvWithoutAPIVersion)
	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}
	defer cli.Close()

	if got := cli.ClientVersion(); got != api.DefaultVersion {
		t.Fatalf("expected client version %s, got %s", api.DefaultVersion, got)
	}
}

func TestFromEnvWithoutAPIVersionStillUsesDockerHost(t *testing.T) {
	const host = "unix:///tmp/groundseg-docker.sock"
	t.Setenv(docker.EnvOverrideHost, host)

	cli, err := docker.NewClientWithOpts(fromEnvWithoutAPIVersion)
	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}
	defer cli.Close()

	if got := cli.DaemonHost(); got != host {
		t.Fatalf("expected docker host %s, got %s", host, got)
	}
}

func TestNewDoesNotPinDockerAPIVersionEnvOnFallback(t *testing.T) {
	t.Setenv(docker.EnvOverrideHost, "unix:///tmp/groundseg-docker-missing.sock")
	t.Setenv(docker.EnvOverrideAPIVersion, "1.43")

	versionMu.Lock()
	cachedVersion = ""
	versionDetected = false
	versionMu.Unlock()
	ignoredAPIEnvWarned = sync.Once{}

	cli, err := New()
	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}
	defer cli.Close()

	if got := cli.ClientVersion(); got == "1.43" {
		t.Fatalf("expected docker client not to be pinned to DOCKER_API_VERSION, got %s", got)
	}
}
