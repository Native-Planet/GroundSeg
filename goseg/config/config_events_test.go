package config

import (
	"context"
	"groundseg/structs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStartConfEventLoopSetsDefaultC2CInterval(t *testing.T) {
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
	if err := StartConfEventLoop(ctx, events); err != nil {
		t.Fatalf("StartConfEventLoop returned error: %v", err)
	}

	events <- "c2cInterval"
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if Conf().Connectivity.C2cInterval == 600 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
		t.Fatalf("expected C2cInterval to be set to 600, got %d", Conf().Connectivity.C2cInterval)
}

func TestStartConfEventLoopCancels(t *testing.T) {
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
	globalConfig.Connectivity.C2cInterval = 60

	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan string, 1)
	if err := StartConfEventLoop(ctx, events); err != nil {
		t.Fatalf("StartConfEventLoop returned error: %v", err)
	}
	cancel()

	events <- "c2cInterval"
	time.Sleep(50 * time.Millisecond)
		if got := Conf().Connectivity.C2cInterval; got != 60 {
		t.Fatalf("expected canceled loop to ignore events, got %d", got)
	}
}

func TestStartConfEventLoopRejectsNilEventSource(t *testing.T) {
	ctx := context.Background()
	if err := StartConfEventLoop(ctx, nil); err == nil {
		t.Fatal("expected error for nil event channel")
	}
}
