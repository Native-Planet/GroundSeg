package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitializePathsUsesEnvBasePath(t *testing.T) {
	originalBasePath := BasePath()
	t.Cleanup(func() {
		SetBasePath(originalBasePath)
	})

	expectedBasePath := filepath.Join(t.TempDir(), "groundseg")
	t.Setenv("GS_BASE_PATH", expectedBasePath)

	initializePaths()

	if got := BasePath(); got != expectedBasePath {
		t.Fatalf("initializePaths base path mismatch: got %q want %q", got, expectedBasePath)
	}
}

func TestInitializeDebugModeFollowsCommandArgs(t *testing.T) {
	originalArgs := os.Args
	originalDebugMode := DebugMode()
	t.Cleanup(func() {
		os.Args = originalArgs
		SetDebugMode(originalDebugMode)
	})

	os.Args = []string{"groundseg", "dev"}
	initializeDebugMode()
	if !DebugMode() {
		t.Fatal("expected debug mode enabled when args contain dev")
	}

	os.Args = []string{"groundseg"}
	initializeDebugMode()
	if DebugMode() {
		t.Fatal("expected debug mode disabled when args omit dev")
	}
}

func TestConfigPathHelpersUseRuntimeBasePath(t *testing.T) {
	originalBasePath := BasePath()
	t.Cleanup(func() {
		SetBasePath(originalBasePath)
	})

	expectedBasePath := filepath.Join(t.TempDir(), "groundseg")
	SetBasePath(expectedBasePath)

	configPath := ConfigFilePath()
	if configPath != filepath.Join(expectedBasePath, "settings", "system.json") {
		t.Fatalf("ConfigFilePath mismatch: got %q", configPath)
	}

	keyPath := SessionKeyPath()
	if keyPath != filepath.Join(expectedBasePath, "settings", "session.key") {
		t.Fatalf("SessionKeyPath mismatch: got %q", keyPath)
	}
}
