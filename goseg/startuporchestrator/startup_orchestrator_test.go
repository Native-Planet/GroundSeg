package startuporchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"groundseg/config"
	"groundseg/structs"
)

func TestRunStartupSubsystemsSkipsOptionalFailures(t *testing.T) {
	called := false
	err := runStartupSubsystems([]startupSubsystemStep{
		{
			name:   "optional failure",
			policy: startupSubsystemOptional,
			initFn: func() error {
				called = true
				return errors.New("optional failure")
			},
		},
	})
	if err != nil {
		t.Fatalf("optional failures should not fail startup: %v", err)
	}
	if !called {
		t.Fatal("expected optional startup subsystem to execute")
	}
}

func TestRunStartupSubsystemsReturnsRequiredFailure(t *testing.T) {
	err := runStartupSubsystems([]startupSubsystemStep{
		{
			name:   "required failure",
			policy: startupSubsystemRequired,
			initFn: func() error {
			return errors.New("required failure")
			},
		},
	})
	if err == nil {
		t.Fatal("expected required startup subsystem failure")
	}
	if !strings.Contains(err.Error(), "required failure") {
		t.Fatalf("unexpected required failure error: %v", err)
	}
}

func TestRunStartupSubsystemSkipsDisabledSubsystem(t *testing.T) {
	called := false
	err := runStartupSubsystem(startupSubsystemStep{
		name:   "disabled subsystem",
		policy: startupSubsystemDisabled,
		initFn: func() error {
		called = true
		return nil
		},
	})
	if err != nil {
		t.Fatalf("disabled startup subsystem should not fail: %v", err)
	}
	if called {
		t.Fatal("disabled startup subsystem should not execute callback")
	}
}

func TestBootstrapRequiresStartupRuntimeCallbacks(t *testing.T) {
	runtime := StartupRuntime{
		// initializeConfig intentionally omitted
		startupInitRuntime: startupInitRuntime{
			initializeAuthFn:          func() error { return nil },
			initializeRouterFn:        func() error { return nil },
			initializeSystemSupportFn: func() error { return nil },
			initializeExporterFn:      func() error { return nil },
			initializeImporterFn:      func() error { return nil },
			initializeBroadcastFn:     func() error { return nil },
			initializeDockerFn:        func() error { return nil },
			initializeWiFiFn:         func() error { return nil },
			startMDNSServerFn:        func() error { return nil },
			initializeResolvedFn:      func() error { return nil },
			networkReachabilityFn:     func(string) bool { return true },
			primeRekorKeyFn:           func() error { return nil },
			ConfigureSwapFn: func(string, int) error { return nil },
			SetupTmpDirFn:   func() error { return nil },
		},
		startupBootstrapRuntime: startupBootstrapRuntime{
			StartStartupContainersFn: func(bool) {},
		},
	}
	if err := runtime.validate(); err == nil {
		t.Fatal("expected startup runtime validation failure when required callbacks are missing")
	}
	err := Bootstrap(context.Background(), StartupOptions{
		HTTPPort: 8080,
		StartServer: func(context.Context, int) error {
			return nil
		},
		StartupRuntime: runtime,
	})
	if err == nil {
		t.Fatal("expected bootstrap to fail with incomplete startup callback configuration")
	}
}

func TestStartBackgroundServicesWithRuntimeCallsExpectedServices(t *testing.T) {
	calls := make(chan string, 8)
	c2cCalled := make(chan struct{}, 1)

	runtime := startBackgroundServicesRuntime{
		startVersionSubsystemFn:     func(context.Context) error { calls <- "version"; return nil },
		startDockerSubsystemFn:      func(context.Context) error { calls <- "docker"; return nil },
		startUrbitTransitionFn:      func(context.Context) error { calls <- "urbit"; return nil },
		startImportShipTransitionFn: func(context.Context) error { calls <- "import"; return nil },
		startRectifyUrbitFn:         func(context.Context) error { calls <- "rectify"; return nil },
		syncRetrieveFn:              func(context.Context) error { calls <- "sync"; return nil },
		startLeakFn:                 func(context.Context) error { calls <- "leak"; return nil },
	}

	runtime = backgroundServiceRuntimeForTest(runtime, map[string]func(context.Context) error{
		"version":                func(context.Context) error { calls <- "version"; return nil },
		"docker":                 func(context.Context) error { calls <- "docker"; return nil },
		"leak":                   func(context.Context) error { calls <- "leak"; return nil },
		"urbit-transition":       func(context.Context) error { calls <- "urbit"; return nil },
		"import-ship-transition": func(context.Context) error { calls <- "import"; return nil },
		"rectify":                func(context.Context) error { calls <- "rectify"; return nil },
		"docker-log-streamer":    func(context.Context) error { calls <- "logstream"; return nil },
		"startram-sync":          func(context.Context) error { calls <- "sync"; return nil },
	})

	services, err := startBackgroundServicesWithRuntime(context.Background(), true, func(_ context.Context) error {
		c2cCalled <- struct{}{}
		return nil
	}, runtime)
	if err != nil {
		t.Fatalf("unexpected startup runtime callback validation error: %v", err)
	}
	defer services.stop()

	expected := map[string]struct{}{
		"version":   {},
		"docker":    {},
		"urbit":     {},
		"import":    {},
		"rectify":   {},
		"leak":      {},
		"sync":      {},
		"logstream": {},
	}

	for len(expected) > 0 {
		select {
		case got := <-calls:
			delete(expected, got)
		case <-time.After(time.Second):
			t.Fatalf("timed out for startup runtime callbacks, remaining=%d", len(expected))
		}
	}

	select {
	case <-c2cCalled:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for c2c callback to start")
	}
}

func TestStartBackgroundServicesSkipsStartramSyncWhenNotRegistered(t *testing.T) {
	callCount := 0
	runtime := startBackgroundServicesRuntime{
		syncRetrieveFn: func(context.Context) error {
			callCount++
			return errors.New("unexpected sync call")
		},
	}
	services, err := startBackgroundServicesWithRuntime(context.Background(), false, nil, runtime)
	if err != nil {
		t.Fatalf("unexpected startup runtime callback validation error: %v", err)
	}
	defer services.stop()
	if callCount != 0 {
		t.Fatalf("expected no startram sync call when unregistered, got %d", callCount)
	}
}

func TestStartBackgroundServicesRuntimeValidation(t *testing.T) {
	var runtime startBackgroundServicesRuntime
	if err := runtime.validate(true); err == nil {
		t.Fatal("expected validation failure when required background services are missing")
	}
	if err := runtime.validate(true); err == nil || !strings.Contains(err.Error(), "start background services runtime missing required callbacks") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartBackgroundServicesDefaultsApply(t *testing.T) {
	_, err := startBackgroundServicesWithRuntime(context.Background(), true, nil, startBackgroundServicesRuntime{})
	if err != nil {
		t.Fatalf("expected defaults to make empty background runtime valid, got %v", err)
	}
}

func backgroundServiceRuntimeForTest(runtime startBackgroundServicesRuntime, overrides map[string]func(context.Context) error) startBackgroundServicesRuntime {
	for name, callback := range overrides {
		switch name {
		case "version":
			runtime.startVersionSubsystemFn = callback
		case "docker":
			runtime.startDockerSubsystemFn = callback
		case "urbit-transition":
			runtime.startUrbitTransitionFn = callback
		case "system-transition":
			runtime.startSystemTransitionFn = callback
		case "new-ship-transition":
			runtime.startNewShipTransitionFn = callback
		case "import-ship-transition":
			runtime.startImportShipTransitionFn = callback
		case "rectify":
			runtime.startRectifyUrbitFn = callback
		case "startram-sync":
			runtime.syncRetrieveFn = callback
		case "leak":
			runtime.startLeakFn = callback
		case "sys-log-streamer":
			runtime.startSysLogStreamerFn = callback
		case "docker-log-streamer":
			runtime.startDockerLogStreamerFn = callback
		case "docker-log-conn-remover":
			runtime.startDockerLogConnRemoverFn = callback
		case "old-logs-cleaner":
			runtime.startOldLogsCleanerFn = callback
		case "disk-usage-warning":
			runtime.startDiskUsageWarningFn = callback
		case "smart-disk-check":
			runtime.startSmartDiskCheckFn = callback
		case "startram-renewal":
			runtime.startStartramRenewalReminderFn = callback
		case "pack-schedule":
			runtime.startPackScheduleLoopFn = callback
		case "chop-routines":
			runtime.startChopRoutinesFn = callback
		case "backup-routines":
			runtime.startBackupRoutinesFn = callback
		default:
			panic("unknown background service: " + name)
		}
	}
	return runtime
}

func TestWaitForVersionDiscoveryUsesRemoteVersionWhenSuccessful(t *testing.T) {
	branch := "stable"
	remoteChannel := structs.Channel{Groundseg: structs.VersionDetails{Repo: "remote-repo"}}
	localChannel := structs.Channel{Groundseg: structs.VersionDetails{Repo: "local-repo"}}

	oldBasePath := config.BasePath()
	config.SetBasePath(t.TempDir())
	t.Cleanup(func() {
		config.SetBasePath(oldBasePath)
	})
	if err := os.MkdirAll(filepath.Join(config.BasePath(), "settings"), 0o755); err != nil {
		t.Fatalf("mkdir settings failed: %v", err)
	}
	localVersion := structs.Version{Groundseg: map[string]structs.Channel{
		branch: localChannel,
	}}
	rawLocal, err := json.MarshalIndent(localVersion, "", "  ")
	if err != nil {
		t.Fatalf("marshal local version failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(config.BasePath(), "settings", "version_info.json"), rawLocal, 0o644); err != nil {
		t.Fatalf("write local version file failed: %v", err)
	}

	config.SetVersionChannel(remoteChannel)
	versionUpdate := make(chan bool, 1)
	versionUpdate <- true
	waitForVersionDiscovery(true, versionUpdate, branch)

	got := config.GetVersionChannel()
	if got.Groundseg.Repo != remoteChannel.Groundseg.Repo {
		t.Fatalf("expected remote version to be used, got %v", got.Groundseg.Repo)
	}
}

func TestWaitForVersionDiscoveryFallsBackToLocalChannelOnRemoteFailure(t *testing.T) {
	branch := "stable"
	remoteChannel := structs.Channel{Groundseg: structs.VersionDetails{Repo: "remote-repo"}}
	localChannel := structs.Channel{Groundseg: structs.VersionDetails{Repo: "local-repo"}}

	oldBasePath := config.BasePath()
	config.SetBasePath(t.TempDir())
	t.Cleanup(func() {
		config.SetBasePath(oldBasePath)
	})
	if err := os.MkdirAll(filepath.Join(config.BasePath(), "settings"), 0o755); err != nil {
		t.Fatalf("mkdir settings failed: %v", err)
	}
	localVersion := structs.Version{Groundseg: map[string]structs.Channel{
		branch: localChannel,
	}}
	rawLocal, err := json.MarshalIndent(localVersion, "", "  ")
	if err != nil {
		t.Fatalf("marshal local version failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(config.BasePath(), "settings", "version_info.json"), rawLocal, 0o644); err != nil {
		t.Fatalf("write local version file failed: %v", err)
	}

	config.SetVersionChannel(remoteChannel)
	versionUpdate := make(chan bool, 1)
	versionUpdate <- false
	waitForVersionDiscovery(true, versionUpdate, branch)

	got := config.GetVersionChannel()
	if got.Groundseg.Repo != localChannel.Groundseg.Repo {
		t.Fatalf("expected local fallback to be used, got %v", got.Groundseg.Repo)
	}
}

func TestWaitForVersionDiscoveryFallsBackToExistingChannelWhenBranchMissing(t *testing.T) {
	branch := "stable"
	remoteChannel := structs.Channel{Groundseg: structs.VersionDetails{Repo: "remote-repo"}}

	oldBasePath := config.BasePath()
	config.SetBasePath(t.TempDir())
	t.Cleanup(func() {
		config.SetBasePath(oldBasePath)
	})
	if err := os.MkdirAll(filepath.Join(config.BasePath(), "settings"), 0o755); err != nil {
		t.Fatalf("mkdir settings failed: %v", err)
	}
	localVersion := structs.Version{Groundseg: map[string]structs.Channel{
		"other": {Groundseg: structs.VersionDetails{Repo: "other-repo"}},
	}}
	rawLocal, err := json.MarshalIndent(localVersion, "", "  ")
	if err != nil {
		t.Fatalf("marshal local version failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(config.BasePath(), "settings", "version_info.json"), rawLocal, 0o644); err != nil {
		t.Fatalf("write local version file failed: %v", err)
	}

	config.SetVersionChannel(remoteChannel)
	versionUpdate := make(chan bool, 1)
	versionUpdate <- false
	waitForVersionDiscovery(true, versionUpdate, branch)

	got := config.GetVersionChannel()
	if got.Groundseg.Repo != remoteChannel.Groundseg.Repo {
		t.Fatalf("expected existing channel when branch missing, got %v", got.Groundseg.Repo)
	}
}

func TestStartupRuntimeValidation(t *testing.T) {
	err := StartupRuntime{}.validate()
	if err == nil {
		t.Fatal("expected startup runtime validation failure when required callbacks are missing")
	}
	if !strings.Contains(err.Error(), "startup runtime missing required callbacks") {
		t.Fatalf("unexpected error: %v", err)
	}
}
