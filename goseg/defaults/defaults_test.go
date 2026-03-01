package defaults

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSysConfigDefaults(t *testing.T) {
	basePath := "/tmp/groundseg-test"
	cfg := SysConfig(basePath)

	if cfg.Setup != "start" {
		t.Fatalf("expected default setup=start, got %q", cfg.Setup)
	}
	if cfg.KeyFile != filepath.Join(basePath, "settings", "session.key") {
		t.Fatalf("unexpected keyFile path: %q", cfg.KeyFile)
	}
	if cfg.SwapFile != filepath.Join(basePath, "swapfile") {
		t.Fatalf("unexpected swap file path: %q", cfg.SwapFile)
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
