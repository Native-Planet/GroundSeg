package docker

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"groundseg/config"
	"groundseg/structs"

	"github.com/docker/docker/api/types/container"
)

func TestGetLatestContainerInfoUsesBundledBeszelSlots(t *testing.T) {
	originalVersionInfo := config.VersionInfo
	originalArchitecture := config.Architecture
	t.Cleanup(func() {
		config.VersionInfo = originalVersionInfo
		config.Architecture = originalArchitecture
	})

	config.VersionInfo = structs.Channel{}
	config.Architecture = "amd64"

	tests := []struct {
		containerType string
		repo          string
		hash          string
	}{
		{
			containerType: "beszel",
			repo:          "registry.hub.docker.com/henrygd/beszel",
			hash:          "1c5a4b2a277b6878b2d612419f02fc88957f870435ab2bf000c8bc14835c1aa5",
		},
		{
			containerType: "beszel-agent",
			repo:          "registry.hub.docker.com/henrygd/beszel-agent",
			hash:          "a45ca92579eca9b5e7304e2f5982de3c3e6325151e48dbc59e4fa0c5a28d55a1",
		},
	}

	for _, test := range tests {
		t.Run(test.containerType, func(t *testing.T) {
			info, err := GetLatestContainerInfo(test.containerType)
			if err != nil {
				t.Fatalf("GetLatestContainerInfo(%q) returned an error: %v", test.containerType, err)
			}
			if info["repo"] != test.repo {
				t.Fatalf("repo = %q, want %q", info["repo"], test.repo)
			}
			if info["tag"] != "0.18.7" {
				t.Fatalf("tag = %q, want %q", info["tag"], "0.18.7")
			}
			if info["hash"] != test.hash {
				t.Fatalf("hash = %q, want %q", info["hash"], test.hash)
			}
		})
	}
}

func TestBeszelContainerConfigsPreserveMonitoringEndpoint(t *testing.T) {
	originalVersionInfo := config.VersionInfo
	originalArchitecture := config.Architecture
	t.Cleanup(func() {
		config.VersionInfo = originalVersionInfo
		config.Architecture = originalArchitecture
	})

	config.VersionInfo = structs.Channel{}
	config.Architecture = "amd64"

	hubConfig, hubHostConfig, err := beszelHubContainerConf("bootstrap-secret")
	if err != nil {
		t.Fatalf("beszelHubContainerConf returned an error: %v", err)
	}
	if bindings := hubHostConfig.PortBindings["8090/tcp"]; len(bindings) != 1 || bindings[0].HostPort != "19999" {
		t.Fatalf("hub port bindings = %#v, want container 8090 mapped to host 19999", hubHostConfig.PortBindings)
	}
	assertContains(t, hubConfig.Env, "AUTO_LOGIN=admin@groundseg.local")
	assertContains(t, hubConfig.Env, "USER_EMAIL=admin@groundseg.local")
	assertContains(t, hubConfig.Env, "USER_PASSWORD=bootstrap-secret")
	assertContains(t, hubHostConfig.Binds, "beszel_data:/beszel_data")

	agentConfig, agentHostConfig, err := beszelAgentContainerConf("ssh-ed25519 test-key", "registration-token")
	if err != nil {
		t.Fatalf("beszelAgentContainerConf returned an error: %v", err)
	}
	if agentHostConfig.NetworkMode != "host" {
		t.Fatalf("agent network mode = %q, want host", agentHostConfig.NetworkMode)
	}
	assertContains(t, agentConfig.Env, "HUB_URL=http://127.0.0.1:19999")
	assertContains(t, agentConfig.Env, "KEY=ssh-ed25519 test-key")
	assertContains(t, agentConfig.Env, "TOKEN=registration-token")
	assertContains(t, agentConfig.Env, "SYSTEM_NAME=GroundSeg")
	assertContains(t, agentConfig.Env, "DISABLE_SSH=true")
	assertContains(t, agentHostConfig.Binds, "beszel_agent_data:/var/lib/beszel-agent")
	assertContains(t, agentHostConfig.Binds, "/var/run/docker.sock:/var/run/docker.sock:ro")
}

func TestGetLatestContainerInfoUsesBeszelHubVersionSlot(t *testing.T) {
	originalVersionInfo := config.VersionInfo
	originalArchitecture := config.Architecture
	t.Cleanup(func() {
		config.VersionInfo = originalVersionInfo
		config.Architecture = originalArchitecture
	})

	config.VersionInfo = structs.Channel{
		Beszel: structs.VersionDetails{
			Amd64Sha256: "server-hub-amd64",
			Arm64Sha256: "server-hub-arm64",
			Repo:        "registry.example.com/beszel",
			Tag:         "server-version",
		},
	}
	config.Architecture = "amd64"

	info, err := GetLatestContainerInfo("beszel")
	if err != nil {
		t.Fatalf("GetLatestContainerInfo returned an error: %v", err)
	}
	if info["repo"] != "registry.example.com/beszel" {
		t.Fatalf("repo = %q, want version-server repo", info["repo"])
	}
	if info["tag"] != "server-version" || info["hash"] != "server-hub-amd64" {
		t.Fatalf("version-server Hub details were not used: %#v", info)
	}
}

func TestGetLatestContainerInfoUsesBeszelAgentVersionSlot(t *testing.T) {
	originalVersionInfo := config.VersionInfo
	originalArchitecture := config.Architecture
	t.Cleanup(func() {
		config.VersionInfo = originalVersionInfo
		config.Architecture = originalArchitecture
	})

	config.VersionInfo = structs.Channel{
		BeszelAgent: structs.VersionDetails{
			Amd64Sha256: "server-agent-amd64",
			Arm64Sha256: "server-agent-arm64",
			Repo:        "registry.example.com/beszel-agent",
			Tag:         "server-version",
		},
	}
	config.Architecture = "amd64"

	info, err := GetLatestContainerInfo("beszel-agent")
	if err != nil {
		t.Fatalf("GetLatestContainerInfo returned an error: %v", err)
	}
	if info["repo"] != "registry.example.com/beszel-agent" {
		t.Fatalf("repo = %q, want version-server repo", info["repo"])
	}
	if info["tag"] != "server-version" || info["hash"] != "server-agent-amd64" {
		t.Fatalf("version-server Agent details were not used: %#v", info)
	}
}

func TestBootstrapBeszelEnablesPermanentUniversalToken(t *testing.T) {
	const token = "registration token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/beszel/info":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"key":"ssh-ed25519 hub-key"}`)
		case "/api/beszel/universal-token":
			if r.URL.Query().Get("enable") != "1" || r.URL.Query().Get("permanent") != "1" {
				t.Errorf("universal-token query = %q, want enable=1 and permanent=1", r.URL.RawQuery)
			}
			if r.URL.Query().Get("token") != token {
				t.Errorf("token = %q, want %q", r.URL.Query().Get("token"), token)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"active":true,"permanent":true,"token":%q}`, token)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	key, err := bootstrapBeszel(ctx, server.URL, token)
	if err != nil {
		t.Fatalf("bootstrapBeszel returned an error: %v", err)
	}
	if key != "ssh-ed25519 hub-key" {
		t.Fatalf("key = %q, want %q", key, "ssh-ed25519 hub-key")
	}
}

func TestRemoveExistingBeszelAgentAllowsFirstRun(t *testing.T) {
	removeCalled := false
	err := removeExistingBeszelAgent(
		func(string) (*container.Summary, error) {
			return nil, fmt.Errorf("%w: %s", ErrContainerNotFound, beszelAgentContainerName)
		},
		func(string) error {
			removeCalled = true
			return nil
		},
	)
	if err != nil {
		t.Fatalf("removeExistingBeszelAgent returned an error: %v", err)
	}
	if removeCalled {
		t.Fatal("removeExistingBeszelAgent tried to remove an absent first-run Agent")
	}
}

func TestRemoveLegacyNetdataContainerDeletesContainer(t *testing.T) {
	var removed string
	err := removeLegacyNetdataContainer(
		func(name string) (*container.Summary, error) {
			if name != "netdata" {
				t.Fatalf("looked up %q, want netdata", name)
			}
			return &container.Summary{}, nil
		},
		func(name string) error {
			removed = name
			return nil
		},
	)
	if err != nil {
		t.Fatalf("removeLegacyNetdataContainer returned an error: %v", err)
	}
	if removed != "netdata" {
		t.Fatalf("removed %q, want netdata", removed)
	}
}

func assertContains(t *testing.T, values []string, expected string) {
	t.Helper()
	for _, value := range values {
		if strings.TrimSpace(value) == expected {
			return
		}
	}
	t.Fatalf("%q not found in %#v", expected, values)
}
