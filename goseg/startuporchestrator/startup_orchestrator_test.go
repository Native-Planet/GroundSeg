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
	"groundseg/internal/testruntime"
	"groundseg/structs"
)

func withStartupRuntimeDefaults(overrides ...func(*StartupRuntime)) StartupRuntime {
	return testruntime.Apply(defaultStartupRuntime(), overrides...)
}

type startBackgroundServicesRuntimeOption func(*startBackgroundServicesRuntime)

func withStartBackgroundServicesRuntime(overrides ...startBackgroundServicesRuntimeOption) startBackgroundServicesRuntime {
	runtime := startBackgroundServicesRuntimeWithDefaults(nil, startBackgroundServicesRuntime{})
	for _, apply := range overrides {
		if apply == nil {
			continue
		}
		apply(&runtime)
	}
	return runtime
}

func withStartBackgroundServiceSpecs(specs ...backgroundServiceSpec) startBackgroundServicesRuntimeOption {
	return func(runtime *startBackgroundServicesRuntime) {
		runtime.overrideServiceSpecsFn = func(startramWgRegistered bool) []backgroundServiceSpec {
			return specs
		}
	}
}

func withStartBackgroundServiceCallback(name startupBackgroundServiceName, callback func(context.Context) error) startBackgroundServicesRuntimeOption {
	return func(runtime *startBackgroundServicesRuntime) {
		runtime.setServiceCallback(name, callback)
	}
}

func unsetStartupInitConfig(runtime *StartupRuntime) {
	runtime.startupInitRuntime.InitializeConfigFn = nil
}

func TestRunStartupSubsystemsSkipsOptionalFailures(t *testing.T) {
	called := false
	err := executeStartupSubsystems([]startupSubsystemStep{
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
	err := executeStartupSubsystems([]startupSubsystemStep{
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
	err := executeStartupSubsystem(startupSubsystemStep{
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

func startupInitStepByName(t *testing.T, runtime startupInitRuntime, name string) startupSubsystemStep {
	t.Helper()
	for _, step := range runtime.initSubsystems() {
		if step.name == name {
			return step
		}
	}
	t.Fatalf("startup init step %q not found", name)
	return startupSubsystemStep{}
}

func TestRunStartupSubsystemRequiresConfigureSwapCallbackWhenSwapConfigured(t *testing.T) {
	runtime := startupInitRuntime{
		startupInitStorageRuntime: startupInitStorageRuntime{
			SwapSettingsFn: func() config.SwapSettings {
				return config.SwapSettings{SwapFile: "/swapfile", SwapVal: 2}
			},
			ConfigureSwapFn: nil,
		},
	}

	step := startupInitStepByName(t, runtime, "configure swap")
	err := executeStartupSubsystem(step)
	if err == nil {
		t.Fatal("expected configure swap to fail when callback is missing and swap is configured")
	}
	if !strings.Contains(err.Error(), "configure swap callback is not configured") {
		t.Fatalf("unexpected configure swap error: %v", err)
	}
}

func TestRunStartupSubsystemSkipsConfigureSwapWhenSwapNotConfigured(t *testing.T) {
	runtime := startupInitRuntime{
		startupInitStorageRuntime: startupInitStorageRuntime{
			SwapSettingsFn: func() config.SwapSettings {
				return config.SwapSettings{}
			},
			ConfigureSwapFn: nil,
		},
	}

	step := startupInitStepByName(t, runtime, "configure swap")
	if err := executeStartupSubsystem(step); err != nil {
		t.Fatalf("expected configure swap to be skipped when swap settings are empty: %v", err)
	}
}

func TestBootstrapRequiresStartupRuntimeCallbacks(t *testing.T) {
	runtime := withStartupRuntimeDefaults(func(startupRuntime *StartupRuntime) {
		// initializeConfig intentionally omitted
		unsetStartupInitConfig(startupRuntime)
	})
	if err := runtime.validate(); err == nil {
		t.Fatal("expected startup runtime validation failure when required callbacks are missing")
	}
	err := Bootstrap(context.Background(), StartupOptions{
		HTTPPort: 8080,
		StartServer: func(context.Context, int, int) error {
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

	runtime := withStartBackgroundServicesRuntime(
		withStartBackgroundServiceCallback(startBackgroundServiceVersion, func(context.Context) error { calls <- "version"; return nil }),
		withStartBackgroundServiceCallback(startBackgroundServiceDocker, func(context.Context) error { calls <- "docker"; return nil }),
		withStartBackgroundServiceCallback(startBackgroundServiceLeak, func(context.Context) error { calls <- "leak"; return nil }),
		withStartBackgroundServiceCallback(startBackgroundServiceSysLogStreamer, func(context.Context) error { calls <- "logstream"; return nil }),
		withStartBackgroundServiceCallback(startBackgroundServiceUrbitTransition, func(context.Context) error { calls <- "urbit"; return nil }),
		withStartBackgroundServiceCallback(startBackgroundServiceImportShipTransition, func(context.Context) error { calls <- "import"; return nil }),
		withStartBackgroundServiceCallback(startBackgroundServiceRectify, func(context.Context) error { calls <- "rectify"; return nil }),
		withStartBackgroundServiceCallback(startBackgroundServiceStartramSync, func(context.Context) error { calls <- "sync"; return nil }),
	)

	services, err := startBackgroundServicesWithRuntime(context.Background(), true, func(_ context.Context) error {
		c2cCalled <- struct{}{}
		return nil
	}, runtime, nil)
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

func TestStartBackgroundServicesWithRuntimeKeepsStartingAfterFailFastNil(t *testing.T) {
	failfastStarted := make(chan struct{}, 1)
	normalStarted := make(chan struct{}, 1)
	runtime := withStartBackgroundServicesRuntime(
		withStartBackgroundServiceSpecs(
			backgroundServiceSpec{
				name:     "failfast-success",
				failFast: true,
				startFn: func(context.Context) error {
					failfastStarted <- struct{}{}
					return nil
				},
			},
			backgroundServiceSpec{
				name:     "non-failfast",
				failFast: false,
				startFn: func(context.Context) error {
					normalStarted <- struct{}{}
					return nil
				},
			},
		),
	)

	services, err := startBackgroundServicesWithRuntime(context.Background(), false, nil, runtime, nil)
	if err != nil {
		t.Fatalf("unexpected startup background services error: %v", err)
	}
	defer services.stop()

	select {
	case <-failfastStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for fail-fast service to start")
	}
	select {
	case <-normalStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for normal service to start")
	}
}

func TestStartBackgroundServicesHealthDetectsFailFastErrors(t *testing.T) {
	failfastErr := errors.New("failfast startup fault")
	failfastStarted := make(chan struct{}, 1)
	runtime := withStartBackgroundServicesRuntime(
		withStartBackgroundServiceSpecs(
			backgroundServiceSpec{
				name:     "slow-failfast",
				failFast: true,
				startFn: func(context.Context) error {
					failfastStarted <- struct{}{}
					time.Sleep(20 * time.Millisecond)
					return failfastErr
				},
			},
		),
	)

	services, err := startBackgroundServicesWithRuntime(context.Background(), false, nil, runtime, nil)
	if err != nil {
		t.Fatalf("unexpected startup background services error: %v", err)
	}
	defer services.stop()

	select {
	case <-failfastStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for fail-fast service to start")
	}

	healthErr := services.health(context.Background())
	if healthErr == nil {
		t.Fatal("expected delayed fail-fast error to be detected by health")
	}
	if !strings.Contains(healthErr.Error(), "failfast startup fault") {
		t.Fatalf("unexpected fail-fast error: %v", healthErr)
	}
}

func TestStartBackgroundServicesSkipsStartramSyncWhenNotRegistered(t *testing.T) {
	callCount := 0
	runtime := withStartBackgroundServicesRuntime(
		withStartBackgroundServiceCallback(startBackgroundServiceStartramSync, func(context.Context) error {
			callCount++
			return errors.New("unexpected sync call")
		}),
	)
	services, err := startBackgroundServicesWithRuntime(context.Background(), false, nil, runtime, nil)
	if err != nil {
		t.Fatalf("unexpected startup runtime callback validation error: %v", err)
	}
	defer services.stop()
	if callCount != 0 {
		t.Fatalf("expected no startram sync call when unregistered, got %d", callCount)
	}
}

func TestStartupBackgroundServicesAddRetainsHandles(t *testing.T) {
	runtimeServices := &startupBackgroundServices{}
	runtimeServices.add(backgroundServiceHandle{name: "a"})
	runtimeServices.add(backgroundServiceHandle{name: "b"})

	if got := len(runtimeServices.services); got != 2 {
		t.Fatalf("expected startup background services to retain appended handles, got %d", got)
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
	_, err := startBackgroundServicesWithRuntime(context.Background(), true, nil, startBackgroundServicesRuntime{}, nil)
	if err != nil {
		t.Fatalf("expected defaults to make empty background runtime valid, got %v", err)
	}
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
