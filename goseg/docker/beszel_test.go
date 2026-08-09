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
)

func TestGetLatestContainerInfoUsesBundledBeszelForLegacyNetdataSlot(t *testing.T) {
	originalVersionInfo := config.VersionInfo
	originalArchitecture := config.Architecture
	t.Cleanup(func() {
		config.VersionInfo = originalVersionInfo
		config.Architecture = originalArchitecture
	})

	config.VersionInfo = structs.Channel{
		Netdata: structs.VersionDetails{
			Amd64Sha256: "legacy-netdata-amd64",
			Arm64Sha256: "legacy-netdata-arm64",
			Repo:        "registry.hub.docker.com/netdata/netdata",
			Tag:         "latest",
		},
	}
	config.Architecture = "amd64"

	tests := []struct {
		containerType string
		repo          string
		hash          string
	}{
		{
			containerType: "netdata",
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

func TestBeszelContainerConfigsPreserveNetdataEndpoint(t *testing.T) {
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

func assertContains(t *testing.T, values []string, expected string) {
	t.Helper()
	for _, value := range values {
		if strings.TrimSpace(value) == expected {
			return
		}
	}
	t.Fatalf("%q not found in %#v", expected, values)
}
