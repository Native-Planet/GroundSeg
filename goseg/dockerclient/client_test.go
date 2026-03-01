package dockerclient

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/docker/docker/client"
)

func resetDockerClientSeams() {
	versionMu.Lock()
	cachedVersion = ""
	versionDetected = false
	versionMu.Unlock()
	ignoredAPIEnvWarned = sync.Once{}
	detectAPIVersionForClient = detectAPIVersion
	newClientWithOptsForClient = client.NewClientWithOpts
}

func TestGetAPIVersionCachesResult(t *testing.T) {
	t.Cleanup(resetDockerClientSeams)

	calls := 0
	detectAPIVersionForClient = func() (string, error) {
		calls++
		return "1.42", nil
	}

	v1, err := getAPIVersion()
	if err != nil {
		t.Fatalf("getAPIVersion failed: %v", err)
	}
	v2, err := getAPIVersion()
	if err != nil {
		t.Fatalf("getAPIVersion failed on second call: %v", err)
	}
	if v1 != "1.42" || v2 != "1.42" || calls != 1 {
		t.Fatalf("expected cached version with one detect call, got v1=%s v2=%s calls=%d", v1, v2, calls)
	}
}

func TestGetAPIVersionErrorNotCached(t *testing.T) {
	t.Cleanup(resetDockerClientSeams)

	calls := 0
	detectAPIVersionForClient = func() (string, error) {
		calls++
		return "", errors.New("detect failed")
	}

	if _, err := getAPIVersion(); err == nil {
		t.Fatalf("expected detect error")
	}
	if _, err := getAPIVersion(); err == nil {
		t.Fatalf("expected detect error on second call")
	}
	if calls != 2 {
		t.Fatalf("expected no cache on errors, detect calls=%d", calls)
	}
}

func TestNewAppliesDetectedVersionOption(t *testing.T) {
	t.Cleanup(resetDockerClientSeams)

	detectCalls := 0
	detectAPIVersionForClient = func() (string, error) {
		detectCalls++
		return "1.43", nil
	}
	factoryCalled := false
	var gotOpts int
	newClientWithOptsForClient = func(opts ...client.Opt) (*client.Client, error) {
		factoryCalled = true
		gotOpts = len(opts)
		return &client.Client{}, nil
	}

	if _, err := New(client.WithHost("unix:///var/run/docker.sock")); err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if !factoryCalled || detectCalls != 1 {
		t.Fatalf("expected detect + client factory call, got detect=%d factory=%v", detectCalls, factoryCalled)
	}
	if gotOpts < 3 {
		t.Fatalf("expected base opts plus extra opt, got %d", gotOpts)
	}
}

func TestFromEnvWithoutAPIVersionIgnoresEnvOverride(t *testing.T) {
	t.Cleanup(resetDockerClientSeams)
	t.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
	t.Setenv(client.EnvOverrideAPIVersion, "9.9")

	cli, err := client.NewClientWithOpts()
	if err != nil {
		t.Skipf("docker client unavailable in test env: %v", err)
	}
	if err := fromEnvWithoutAPIVersion(cli); err != nil && !strings.Contains(err.Error(), "cannot apply host to transport") {
		t.Fatalf("fromEnvWithoutAPIVersion failed: %v", err)
	}
	if cli.ClientVersion() == "9.9" {
		t.Fatalf("expected DOCKER_API_VERSION override to be ignored, got %q", cli.ClientVersion())
	}
}
