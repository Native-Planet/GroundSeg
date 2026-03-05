package orchestration

import (
	"testing"

	"groundseg/config/runtimecontext"
)

func TestRuntimeSeamBundleReadsRuntimeContextPackage(t *testing.T) {
	original := runtimecontext.Snapshot()
	t.Cleanup(func() {
		runtimecontext.Set(original)
	})

	runtimecontext.Set(runtimecontext.RuntimeContext{
		BasePath:     "/tmp/groundseg-runtime-context",
		Architecture: "arm64",
		DebugMode:    true,
		DockerDir:    "/var/lib/docker/",
	})
	ops := buildRuntimeSeamBundle().contextOps
	if got := ops.BasePathFn(); got != "/tmp/groundseg-runtime-context" {
		t.Fatalf("unexpected base path from runtime context ops: %s", got)
	}
	if got := ops.ArchitectureFn(); got != "arm64" {
		t.Fatalf("unexpected architecture from runtime context ops: %s", got)
	}
	if !ops.DebugModeFn() {
		t.Fatal("expected debug mode from runtime context ops")
	}
	if got := ops.DockerDirFn(); got != "/var/lib/docker/" {
		t.Fatalf("unexpected docker dir from runtime context ops: %s", got)
	}
}

func TestDefaultRuntimeOpsProvideCoreCallbacks(t *testing.T) {
	containerOps := defaultRuntimeContainerOps()
	if containerOps.StartContainerFn == nil || containerOps.DeleteContainerFn == nil || containerOps.GetShipStatusFn == nil {
		t.Fatalf("expected default container ops callbacks to be wired")
	}

	urbitOps := defaultRuntimeUrbit()
	if urbitOps.UpdateUrbitFn == nil || urbitOps.UpdateUrbitSectionFn == nil || urbitOps.GetLusCodeFn == nil {
		t.Fatalf("expected default urbit ops callbacks to be wired")
	}

	snapshotOps := defaultRuntimeSnapshot()
	if snapshotOps.GetStartramConfigFn == nil || snapshotOps.ShipSettingsSnapshotFn == nil {
		t.Fatalf("expected default snapshot ops callbacks to be wired")
	}

	startupOps := defaultRuntimeStartupOps()
	if startupOps.UpdateConfigTypedFn == nil || startupOps.LoadWireguardFn == nil || startupOps.SvcDeleteFn == nil {
		t.Fatalf("expected default startup ops callbacks to be wired")
	}
	if wireguardOps := defaultRuntimeWireguardOps(); wireguardOps.GetWgConfFn == nil || wireguardOps.CopyFileToVolumeFn == nil {
		t.Fatal("expected default wireguard ops callbacks to be wired")
	}

	if defaultStartupBootstrap().InitializeFn == nil {
		t.Fatal("expected startup bootstrap initialize callback")
	}
	if defaultStartupImage().GetLatestContainerInfoFn == nil {
		t.Fatal("expected startup image metadata callback")
	}
	if defaultStartupLoad().LoadWireguardFn == nil {
		t.Fatal("expected startup load callbacks")
	}
}

func TestDefaultRuntimeStartramOpsReturnMissingDependencyErrors(t *testing.T) {
	startramOps := defaultRuntimeStartramOps()
	if err := startramOps.GetStartramServicesFn(); err == nil {
		t.Fatal("expected missing startram services loader error")
	}
	if err := startramOps.LoadStartramRegionsFn(); err == nil {
		t.Fatal("expected missing startram regions loader error")
	}
}
