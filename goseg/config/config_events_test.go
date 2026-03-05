package config

import (
	"context"
	"groundseg/structs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProcessConfigEventSetsDefaultC2CInterval(t *testing.T) {
	originalBasePath := BasePath()
	originalGlobalConfig := globalConfig

	t.Cleanup(func() {
		SetBasePath(originalBasePath)
		globalConfig = originalGlobalConfig
	})

	tempBasePath := t.TempDir()
	originalEnvBasePath := os.Getenv("GS_BASE_PATH")
	t.Cleanup(func() {
		if originalEnvBasePath == "" {
			_ = os.Unsetenv("GS_BASE_PATH")
		} else {
			_ = os.Setenv("GS_BASE_PATH", originalEnvBasePath)
		}
	})
	if err := os.Setenv("GS_BASE_PATH", tempBasePath); err != nil {
		t.Fatalf("failed to set GS_BASE_PATH: %v", err)
	}
	initializePaths()
	if err := os.MkdirAll(filepath.Dir(ConfigFilePath()), 0o755); err != nil {
		t.Fatalf("failed to prepare config directory: %v", err)
	}
	if err := createDefaultConf(); err != nil {
		t.Fatalf("failed to seed default config file: %v", err)
	}
	globalConfig = structs.SysConfig{}

	processConfigEvent("c2cInterval")
	if got := Config().Connectivity.C2CInterval; got != 600 {
		t.Fatalf("expected C2CInterval to be set to 600, got %d", got)
	}
}

func TestStartConfigEventLoopCancels(t *testing.T) {
	originalBasePath := BasePath()
	originalGlobalConfig := globalConfig

	t.Cleanup(func() {
		SetBasePath(originalBasePath)
		globalConfig = originalGlobalConfig
	})

	tempBasePath := t.TempDir()
	originalEnvBasePath := os.Getenv("GS_BASE_PATH")
	t.Cleanup(func() {
		if originalEnvBasePath == "" {
			_ = os.Unsetenv("GS_BASE_PATH")
		} else {
			_ = os.Setenv("GS_BASE_PATH", originalEnvBasePath)
		}
	})
	if err := os.Setenv("GS_BASE_PATH", tempBasePath); err != nil {
		t.Fatalf("failed to set GS_BASE_PATH: %v", err)
	}
	initializePaths()
	if err := os.MkdirAll(filepath.Dir(ConfigFilePath()), 0o755); err != nil {
		t.Fatalf("failed to prepare config directory: %v", err)
	}
	if err := createDefaultConf(); err != nil {
		t.Fatalf("failed to seed default config file: %v", err)
	}
	globalConfig = structs.SysConfig{}
	globalConfig.Connectivity.C2CInterval = 60

	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan string, 1)
	if err := StartConfigEventLoop(ctx, events); err != nil {
		t.Fatalf("StartConfigEventLoop returned error: %v", err)
	}
	cancel()

	events <- "c2cInterval"
	time.Sleep(50 * time.Millisecond)
	if got := Config().Connectivity.C2CInterval; got != 60 {
		t.Fatalf("expected canceled loop to ignore events, got %d", got)
	}
}

func TestStartConfigEventLoopRejectsNilEventSource(t *testing.T) {
	ctx := context.Background()
	if err := StartConfigEventLoop(ctx, nil); err == nil {
		t.Fatal("expected error for nil event channel")
	}
}
