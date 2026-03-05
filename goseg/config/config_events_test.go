package config

import (
	"context"
	"groundseg/structs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStartConfigEventLoopSetsDefaultC2CInterval(t *testing.T) {
	originalBasePath := BasePath()
	originalGlobalConfig := globalConfig

	t.Cleanup(func() {
		SetBasePath(originalBasePath)
		globalConfig = originalGlobalConfig
	})

	SetBasePath(t.TempDir())
	initializePaths()
	if err := os.MkdirAll(filepath.Dir(ConfigFilePath()), 0o755); err != nil {
		t.Fatalf("failed to prepare config directory: %v", err)
	}
	globalConfig = structs.SysConfig{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := make(chan string, 1)
	if err := StartConfigEventLoop(ctx, events); err != nil {
		t.Fatalf("StartConfigEventLoop returned error: %v", err)
	}

	events <- "c2cInterval"
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if Config().Connectivity.C2CInterval == 600 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected C2CInterval to be set to 600, got %d", Config().Connectivity.C2CInterval)
}

func TestStartConfigEventLoopCancels(t *testing.T) {
	originalBasePath := BasePath()
	originalGlobalConfig := globalConfig

	t.Cleanup(func() {
		SetBasePath(originalBasePath)
		globalConfig = originalGlobalConfig
	})

	SetBasePath(t.TempDir())
	initializePaths()
	if err := os.MkdirAll(filepath.Dir(ConfigFilePath()), 0o755); err != nil {
		t.Fatalf("failed to prepare config directory: %v", err)
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
