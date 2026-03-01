package config

import (
	"os"
	"path/filepath"
	"testing"

	"groundseg/structs"
)

func TestRandStringReturnsEncodedData(t *testing.T) {
	got := RandString(32)
	if got == "" {
		t.Fatal("expected non-empty random string")
	}
	// base64 output should be longer than input bytes.
	if len(got) <= 32 {
		t.Fatalf("expected encoded string longer than 32, got %d", len(got))
	}
}

func TestGetSHA256(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hash.txt")
	if err := os.WriteFile(path, []byte("groundseg"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	got, err := GetSHA256(path)
	if err != nil {
		t.Fatalf("GetSHA256 returned error: %v", err)
	}
	const want = "62bdeec0e24a451b75e5f518384e1ec2856cb5dc2366913d65bb9313899bc2e2"
	if got != want {
		t.Fatalf("unexpected hash: want %s got %s", want, got)
	}
}

func TestMergeConfigsNewInstallSetupAndSalt(t *testing.T) {
	defaultCfg := structs.SysConfig{
		Setup:       "complete",
		EndpointUrl: "default.endpoint",
	}
	customCfg := structs.SysConfig{
		PwHash: "",
	}

	merged := mergeConfigs(defaultCfg, customCfg)
	if merged.Setup != "start" {
		t.Fatalf("expected setup=start for new install, got %q", merged.Setup)
	}
	if merged.Salt == "" {
		t.Fatal("expected salt to be generated for new install")
	}
}

func TestMergeConfigsPrefersCustomEndpoint(t *testing.T) {
	defaultCfg := structs.SysConfig{EndpointUrl: "default.endpoint"}
	customCfg := structs.SysConfig{EndpointUrl: "custom.endpoint", PwHash: "hash"}

	merged := mergeConfigs(defaultCfg, customCfg)
	if merged.EndpointUrl != "custom.endpoint" {
		t.Fatalf("expected custom endpoint to win, got %q", merged.EndpointUrl)
	}
}
