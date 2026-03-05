package defaults

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSysConfigDefaults(t *testing.T) {
	basePath := "/tmp/groundseg-test"
	cfg := SysConfig(basePath)

	if cfg.Runtime.Setup != "start" {
		t.Fatalf("expected default setup=start, got %q", cfg.Runtime.Setup)
	}
	if cfg.Runtime.DockerData != DockerData(basePath) {
		t.Fatalf("expected DockerData to follow supplied base path, got %q", cfg.Runtime.DockerData)
	}
	if cfg.AuthSession.KeyFile != filepath.Join(basePath, "settings", "session.key") {
		t.Fatalf("unexpected keyFile path: %q", cfg.AuthSession.KeyFile)
	}
	if cfg.Runtime.SwapFile != filepath.Join(basePath, "swapfile") {
		t.Fatalf("unexpected swap file path: %q", cfg.Runtime.SwapFile)
	}
}

func TestGetBasePathShape(t *testing.T) {
	path := getBasePath()
	if path == "" {
		t.Fatal("expected non-empty base path")
	}
	if !strings.HasPrefix(path, "/") {
		t.Fatalf("expected absolute base path, got %q", path)
	}
}
