package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRunStartupSubsystemsSkipsOptionalFailures(t *testing.T) {
	called := false
	err := runStartupSubsystems([]startupSubsystemStep{
		startupSubsystemOptionalStep("optional failure", func() error {
			called = true
			return errors.New("optional failure")
		}),
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
		startupSubsystemRequiredStep("required failure", func() error {
			return errors.New("required failure")
		}),
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
	err := runStartupSubsystem(startupSubsystemAction("disabled subsystem", startupSubsystemDisabled, func() error {
		called = true
		return nil
	}))
	if err != nil {
		t.Fatalf("disabled startup subsystem should not fail: %v", err)
	}
	if called {
		t.Fatal("disabled startup subsystem should not execute callback")
	}
}

func TestBootstrapRequiresStartupRuntimeCallbacks(t *testing.T) {
	runtime := startupRuntime{
		// InitializeConfigFn intentionally omitted
		Initialization: startupSubsystemInitializationRuntime{
			InitializeAuthFn:          func() error { return nil },
			InitializeRouterFn:        func() error { return nil },
			InitializeSystemSupportFn: func() error { return nil },
			InitializeExporterFn:      func() error { return nil },
			InitializeImporterFn:      func() error { return nil },
			InitializeBroadcastFn:     func() error { return nil },
			InitializeResolvedFn:      func() error { return nil },
			InitializeDockerFn:        func() error { return nil },
			StartMDNSServerFn:        func() error { return nil },
			InitializeWiFiFn:          func() error { return nil },
			PrimeRekorKeyFn:           func() error { return nil },
		},
		Control: startupRuntimeControlRuntime{
			StartStartupContainersFn: func(bool) {},
			NetworkReachabilityFn:    func(string) bool { return true },
			ConfigureSwapFn:          func(string, int) error { return nil },
			SetupTmpDirFn:            func() error { return nil },
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
		Transition: transitionBackgroundServicesRuntime{
			StartVersionSubsystemFn:            func(context.Context) error { calls <- "version"; return nil },
			StartDockerSubsystemFn:             func(context.Context) error { calls <- "docker"; return nil },
			StartUrbitTransitionHandlerFn:      func(context.Context) error { calls <- "urbit"; return nil },
			StartSystemTransitionHandlerFn:     func(context.Context) error { return nil },
			StartNewShipTransitionHandlerFn:    func(context.Context) error { return nil },
			StartImportShipTransitionHandlerFn: func(context.Context) error { calls <- "import"; return nil },
			StartRectifyUrbitFn:                func(context.Context) error { calls <- "rectify"; return nil },
			StartLeakFn:                        func(context.Context) error { calls <- "leak"; return nil },
		},
		Streaming: streamingBackgroundServicesRuntime{
			StartDockerLogStreamerFn:    func(context.Context) error { calls <- "logstream"; return nil },
			StartDockerLogConnRemoverFn: func(context.Context) error { return nil },
			StartSysLogStreamerFn:       func(context.Context) error { return nil },
		},
		Maintenance: maintenanceBackgroundServicesRuntime{
			StartOldLogsCleanerFn:   func(context.Context) error { return nil },
			StartDiskUsageWarningFn: func(context.Context) error { return nil },
			StartSmartDiskCheckFn:   func(context.Context) error { return nil },
			StartPackScheduleLoopFn: func(context.Context) error { return nil },
			StartChopRoutinesFn:     func(context.Context) error { return nil },
			StartBackupRoutinesFn:   func(context.Context) error { return nil },
		},
		Startram: startramBackgroundServicesRuntime{
			SyncRetrieveFn:                func() error { calls <- "sync"; return nil },
			StartStartramRenewalReminderFn: func(context.Context) error { return nil },
		},
	}

	services, err := startBackgroundServicesWithRuntime(context.Background(), true, func(_ context.Context) {
		c2cCalled <- struct{}{}
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
		Transition: transitionBackgroundServicesRuntime{
			StartVersionSubsystemFn:            func(context.Context) error { return nil },
			StartDockerSubsystemFn:             func(context.Context) error { return nil },
			StartUrbitTransitionHandlerFn:      func(context.Context) error { return nil },
			StartSystemTransitionHandlerFn:     func(context.Context) error { return nil },
			StartNewShipTransitionHandlerFn:    func(context.Context) error { return nil },
			StartImportShipTransitionHandlerFn: func(context.Context) error { return nil },
			StartRectifyUrbitFn:                func(context.Context) error { return nil },
			StartLeakFn:                        func(context.Context) error { return nil },
		},
		Streaming: streamingBackgroundServicesRuntime{
			StartDockerLogStreamerFn:    func(context.Context) error { return nil },
			StartDockerLogConnRemoverFn: func(context.Context) error { return nil },
			StartSysLogStreamerFn:       func(context.Context) error { return nil },
		},
		Maintenance: maintenanceBackgroundServicesRuntime{
			StartOldLogsCleanerFn:   func(context.Context) error { return nil },
			StartDiskUsageWarningFn: func(context.Context) error { return nil },
			StartSmartDiskCheckFn:   func(context.Context) error { return nil },
			StartPackScheduleLoopFn: func(context.Context) error { return nil },
			StartChopRoutinesFn:     func(context.Context) error { return nil },
			StartBackupRoutinesFn:   func(context.Context) error { return nil },
		},
		Startram: startramBackgroundServicesRuntime{
			SyncRetrieveFn: func() error {
				callCount++
				return errors.New("unexpected sync call")
			},
			StartStartramRenewalReminderFn: func(context.Context) error { return nil },
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

func TestStartupRuntimeValidation(t *testing.T) {
	err := startupRuntime{}.validate()
	if err == nil {
		t.Fatal("expected startup runtime validation failure when required callbacks are missing")
	}
	if !strings.Contains(err.Error(), "startup runtime missing required callbacks") {
		t.Fatalf("unexpected error: %v", err)
	}
}
