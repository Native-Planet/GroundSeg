package docker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"groundseg/config"
	"groundseg/structs"
)

func TestHermesShipTargetUsesRemoteURLForWireguardShips(t *testing.T) {
	shipConf := structs.UrbitDocker{
		Network: "wireguard",
		WgURL:   "sampel-palnet.nativeplanet.live",
	}
	if openURL := UrbitWebURL("nativeplanet.local", shipConf); openURL != "https://sampel-palnet.nativeplanet.live" {
		t.Fatalf("expected clickable ship URL to use remote HTTPS URL, got %q", openURL)
	}

	target, err := hermesShipTargetForContainer(shipConf)
	if err != nil {
		t.Fatalf("expected remote target, got error: %v", err)
	}
	if target.URL != "https://sampel-palnet.nativeplanet.live" {
		t.Fatalf("expected remote HTTPS URL, got %q", target.URL)
	}
	if len(target.ExtraHosts) != 0 {
		t.Fatalf("expected no extra host mapping for remote URL, got %#v", target.ExtraHosts)
	}
}

func TestHermesShipTargetUsesCustomRemoteURLWhenSelected(t *testing.T) {
	shipConf := structs.UrbitDocker{
		Network:        "wireguard",
		WgURL:          "sampel-palnet.nativeplanet.live",
		CustomUrbitWeb: "chat.example.com",
		ShowUrbitWeb:   "custom",
	}
	if openURL := UrbitWebURL("nativeplanet.local", shipConf); openURL != "https://chat.example.com" {
		t.Fatalf("expected clickable ship URL to use custom alias, got %q", openURL)
	}

	target, err := hermesShipTargetForContainer(shipConf)
	if err != nil {
		t.Fatalf("expected custom remote target, got error: %v", err)
	}
	if target.URL != "https://chat.example.com" {
		t.Fatalf("expected custom HTTPS URL, got %q", target.URL)
	}
}

func TestHermesShipTargetKeepsSchemeOnCustomRemoteURL(t *testing.T) {
	target, err := hermesShipTargetForContainer(structs.UrbitDocker{
		Network:        "wireguard",
		WgURL:          "sampel-palnet.nativeplanet.live",
		CustomUrbitWeb: "https://chat.example.com",
		ShowUrbitWeb:   "custom",
	})
	if err != nil {
		t.Fatalf("expected custom remote target, got error: %v", err)
	}
	if target.URL != "https://chat.example.com" {
		t.Fatalf("expected custom URL scheme to be preserved, got %q", target.URL)
	}
}

func TestHermesShipTargetUsesHostGatewayForLocalShips(t *testing.T) {
	shipConf := structs.UrbitDocker{
		Network:  "bridge",
		HTTPPort: 8080,
	}
	if openURL := UrbitWebURL("nativeplanet.local", shipConf); openURL != "http://nativeplanet.local:8080" {
		t.Fatalf("expected clickable ship URL to use local host URL, got %q", openURL)
	}

	target, err := hermesShipTargetForContainer(shipConf)
	if err != nil {
		t.Fatalf("expected local target, got error: %v", err)
	}
	if target.URL != "http://host.docker.internal:8080" {
		t.Fatalf("expected host-gateway URL, got %q", target.URL)
	}
	if len(target.ExtraHosts) != 1 || target.ExtraHosts[0] != "host.docker.internal:host-gateway" {
		t.Fatalf("expected host-gateway extra host, got %#v", target.ExtraHosts)
	}
}

func TestHermesContainerAPIEnvRequiresExplicitToggle(t *testing.T) {
	tests := []struct {
		name        string
		apiEnabled  bool
		apiKey      string
		wantEnabled string
		wantKey     bool
	}{
		{
			name:        "disabled omits saved key",
			apiEnabled:  false,
			apiKey:      "saved-api-key",
			wantEnabled: "false",
			wantKey:     false,
		},
		{
			name:        "enabled includes key",
			apiEnabled:  true,
			apiKey:      "enabled-api-key",
			wantEnabled: "true",
			wantKey:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupHermesContainerConfTest(t, tt.apiEnabled, tt.apiKey)

			containerConfig, _, err := hermesContainerConf(HermesContainerName)
			if err != nil {
				t.Fatalf("expected Hermes container config, got error: %v", err)
			}

			if got, ok := envValue(containerConfig.Env, "API_SERVER_ENABLED"); !ok || got != tt.wantEnabled {
				t.Fatalf("API_SERVER_ENABLED = %q, %t; want %q, true", got, ok, tt.wantEnabled)
			}
			gotKey, hasKey := envValue(containerConfig.Env, "API_SERVER_KEY")
			if hasKey != tt.wantKey {
				t.Fatalf("API_SERVER_KEY present = %t, want %t", hasKey, tt.wantKey)
			}
			if hasKey && gotKey != tt.apiKey {
				t.Fatalf("API_SERVER_KEY = %q, want %q", gotKey, tt.apiKey)
			}
			if _, ok := envValue(containerConfig.Env, "TLON_HOSTING"); ok {
				t.Fatalf("TLON_HOSTING should not be set")
			}
		})
	}
}

func TestHermesContainerRunsGatewayInTmuxSession(t *testing.T) {
	setupHermesContainerConfTest(t, false, "")

	containerConfig, _, err := hermesContainerConf(HermesContainerName)
	if err != nil {
		t.Fatalf("expected Hermes container config, got error: %v", err)
	}

	if len(containerConfig.Cmd) != 3 || containerConfig.Cmd[0] != "bash" || containerConfig.Cmd[1] != "-lc" {
		t.Fatalf("expected bash -lc container command, got %#v", containerConfig.Cmd)
	}
	command := containerConfig.Cmd[2]
	for _, want := range []string{
		"tmux new-session -d -s hermes -n gateway",
		"tmux new-window -d -t hermes -n shell",
		"tmux select-window -t hermes:shell",
		"hermes gateway run --replace --accept-hooks",
		"/opt/data/logs/gateway.log",
	} {
		if !strings.Contains(command, want) {
			t.Fatalf("expected Hermes command to contain %q", want)
		}
	}
}

func setupHermesContainerConfTest(t *testing.T, apiEnabled bool, apiKey string) {
	t.Helper()
	oldBasePath := config.BasePath
	oldUrbits := config.UrbitsConfig
	t.Cleanup(func() {
		config.BasePath = oldBasePath
		config.UrbitsConfig = oldUrbits
	})

	config.BasePath = t.TempDir()
	config.UrbitsConfig = make(map[string]structs.UrbitDocker)

	pier := "sampel-palnet"
	pierConf := structs.UrbitDocker{
		PierName: pier,
		Network:  "bridge",
		HTTPPort: 8080,
	}
	pierPath := filepath.Join(config.BasePath, "settings", "pier", pier+".json")
	if err := os.MkdirAll(filepath.Dir(pierPath), 0o755); err != nil {
		t.Fatalf("failed to create pier config dir: %v", err)
	}
	pierJSON, err := json.Marshal(pierConf)
	if err != nil {
		t.Fatalf("failed to encode pier config: %v", err)
	}
	if err := os.WriteFile(pierPath, pierJSON, 0o644); err != nil {
		t.Fatalf("failed to write pier config: %v", err)
	}

	hermesConf := structs.HermesConfig{
		Enabled:        true,
		Ship:           "~" + pier,
		Owner:          "~zod",
		Port:           DefaultHermesDashboardHostPort,
		ModelProvider:  DefaultHermesModelProvider,
		Model:          DefaultHermesModel,
		ProviderAPIKey: "provider-api-key",
		APIEnabled:     apiEnabled,
		APIKey:         apiKey,
		AccessCode:     "access-code",
	}
	if err := config.UpdateHermesConfig(hermesConf); err != nil {
		t.Fatalf("failed to write Hermes config: %v", err)
	}
}

func envValue(env []string, key string) (string, bool) {
	prefix := key + "="
	for _, item := range env {
		if after, ok := strings.CutPrefix(item, prefix); ok {
			return after, true
		}
	}
	return "", false
}
