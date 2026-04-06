package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"groundseg/structs"
)

func TestLoadAndUpdateUrbitConfigKeepsLegacyCustomS3Field(t *testing.T) {
	oldBasePath := BasePath
	oldConfigs := UrbitsConfig
	t.Cleanup(func() {
		BasePath = oldBasePath
		UrbitsConfig = oldConfigs
	})

	BasePath = t.TempDir()
	UrbitsConfig = make(map[string]structs.UrbitDocker)

	pier := "zod"
	path := filepath.Join(BasePath, "settings", "pier", pier+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create pier config dir: %v", err)
	}

	initial := map[string]any{
		"custom_s3_web": "legacy.storage.example.com",
	}
	initialJSON, err := json.Marshal(initial)
	if err != nil {
		t.Fatalf("failed to encode legacy config: %v", err)
	}
	if err := os.WriteFile(path, initialJSON, 0o644); err != nil {
		t.Fatalf("failed to write legacy config: %v", err)
	}

	if err := LoadUrbitConfig(pier); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	conf := UrbitConf(pier)
	if conf.CustomS3WebLocal != "legacy.storage.example.com" {
		t.Fatalf("expected local custom domain to inherit legacy value, got %q", conf.CustomS3WebLocal)
	}
	if conf.CustomS3WebRemote != "legacy.storage.example.com" {
		t.Fatalf("expected remote custom domain to inherit legacy value, got %q", conf.CustomS3WebRemote)
	}
	if conf.CustomS3Web != "legacy.storage.example.com" {
		t.Fatalf("expected legacy compatibility field to remain populated, got %q", conf.CustomS3Web)
	}

	conf.CustomS3WebLocal = "local.storage.example.com"
	conf.CustomS3WebRemote = "remote.storage.example.com"
	if err := UpdateUrbitConfig(map[string]structs.UrbitDocker{pier: conf}); err != nil {
		t.Fatalf("failed to update config: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read updated config: %v", err)
	}

	var saved structs.UrbitDocker
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatalf("failed to decode updated config: %v", err)
	}

	if saved.CustomS3Web != "local.storage.example.com" {
		t.Fatalf("expected legacy field to stay populated with local/default domain, got %q", saved.CustomS3Web)
	}
	if saved.CustomS3WebLocal != "local.storage.example.com" {
		t.Fatalf("expected local field to persist, got %q", saved.CustomS3WebLocal)
	}
	if saved.CustomS3WebRemote != "remote.storage.example.com" {
		t.Fatalf("expected remote field to persist, got %q", saved.CustomS3WebRemote)
	}
}
