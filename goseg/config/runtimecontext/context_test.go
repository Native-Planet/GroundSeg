package runtimecontext

import (
	"os"
	"testing"
)

func TestBasePathFromEnvUsesEnvironmentOverride(t *testing.T) {
	original := os.Getenv("GS_BASE_PATH")
	t.Cleanup(func() {
		if original == "" {
			_ = os.Unsetenv("GS_BASE_PATH")
			return
		}
		_ = os.Setenv("GS_BASE_PATH", original)
	})

	const override = "/tmp/groundseg-custom"
	if err := os.Setenv("GS_BASE_PATH", override); err != nil {
		t.Fatalf("set env override: %v", err)
	}
	if got := BasePathFromEnv(); got != override {
		t.Fatalf("expected base path override %q, got %q", override, got)
	}
}

func TestBasePathFromEnvFallsBackToDefault(t *testing.T) {
	original := os.Getenv("GS_BASE_PATH")
	t.Cleanup(func() {
		if original == "" {
			_ = os.Unsetenv("GS_BASE_PATH")
			return
		}
		_ = os.Setenv("GS_BASE_PATH", original)
	})
	_ = os.Unsetenv("GS_BASE_PATH")
	if got := BasePathFromEnv(); got != "/opt/nativeplanet/groundseg" {
		t.Fatalf("expected default base path, got %q", got)
	}
}

func TestArchitectureFromRuntimeReturnsKnownTarget(t *testing.T) {
	got := ArchitectureFromRuntime()
	if got != "amd64" && got != "arm64" {
		t.Fatalf("expected architecture amd64 or arm64, got %q", got)
	}
}

func TestDebugModeFromArgsReadsDevArgument(t *testing.T) {
	if !DebugModeFromArgs([]string{"dev"}) {
		t.Fatal("expected debug mode to be enabled when dev arg is present")
	}
	if DebugModeFromArgs([]string{"serve"}) {
		t.Fatal("expected debug mode to be disabled when dev arg is absent")
	}
}
