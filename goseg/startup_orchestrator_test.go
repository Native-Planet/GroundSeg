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
		// initializeConfigFn intentionally omitted
		initializeAuthFn:          func() error { return nil },
		initializeRouterFn:        func() error { return nil },
		initializeSystemSupportFn: func() error { return nil },
		initializeExporterFn:      func() error { return nil },
		initializeImporterFn:      func() error { return nil },
		initializeBroadcastFn:     func() error { return nil },
		initializeResolvedFn:      func() error { return nil },
		initializeDockerFn:        func() error { return nil },
		startStartupContainersFn:  func(bool) {},
		networkReachabilityFn:     func(string) bool { return true },
		configureSwapFn:           func(string, int) error { return nil },
		setupTmpDirFn:             func() error { return nil },
		startMDNSServerFn:         func() error { return nil },
		initializeWiFiFn:          func() error { return nil },
		primeRekorKeyFn:           func() error { return nil },
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
		startVersionSubsystemFn:            func(context.Context) error { calls <- "version"; return nil },
		startDockerSubsystemFn:             func(context.Context) error { calls <- "docker"; return nil },
		startUrbitTransitionHandlerFn:      func(context.Context) error { calls <- "urbit"; return nil },
		startSystemTransitionHandlerFn:     func(context.Context) error { return nil },
		startNewShipTransitionHandlerFn:    func(context.Context) error { return nil },
		startImportShipTransitionHandlerFn: func(context.Context) error { calls <- "import"; return nil },
		startRectifyUrbitFn:                func(context.Context) error { calls <- "rectify"; return nil },
		syncRetrieveFn:                     func() error { calls <- "sync"; return nil },
		startLeakFn:                        func(context.Context) error { calls <- "leak"; return nil },
		startDockerLogStreamerFn:           func(context.Context) error { calls <- "logstream"; return nil },
		startDockerLogConnRemoverFn:        func(context.Context) error { return nil },
		startSysLogStreamerFn:              func(context.Context) error { return nil },
		startOldLogsCleanerFn:              func(context.Context) error { return nil },
		startDiskUsageWarningFn:            func(context.Context) error { return nil },
		startSmartDiskCheckFn:              func(context.Context) error { return nil },
		startStartramRenewalReminderFn:     func(context.Context) error { return nil },
		startPackScheduleLoopFn:            func(context.Context) error { return nil },
		startChopRoutinesFn:                func(context.Context) error { return nil },
		startBackupRoutinesFn:              func(context.Context) error { return nil },
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
		startVersionSubsystemFn:            func(context.Context) error { return nil },
		startDockerSubsystemFn:             func(context.Context) error { return nil },
		startUrbitTransitionHandlerFn:      func(context.Context) error { return nil },
		startSystemTransitionHandlerFn:     func(context.Context) error { return nil },
		startNewShipTransitionHandlerFn:    func(context.Context) error { return nil },
		startImportShipTransitionHandlerFn: func(context.Context) error { return nil },
		startRectifyUrbitFn:                func(context.Context) error { return nil },
		syncRetrieveFn: func() error {
			callCount++
			return errors.New("unexpected sync call")
		},
		startLeakFn:                    func(context.Context) error { return nil },
		startDockerLogStreamerFn:       func(context.Context) error { return nil },
		startDockerLogConnRemoverFn:    func(context.Context) error { return nil },
		startSysLogStreamerFn:          func(context.Context) error { return nil },
		startOldLogsCleanerFn:          func(context.Context) error { return nil },
		startDiskUsageWarningFn:        func(context.Context) error { return nil },
		startSmartDiskCheckFn:          func(context.Context) error { return nil },
		startStartramRenewalReminderFn: func(context.Context) error { return nil },
		startPackScheduleLoopFn:        func(context.Context) error { return nil },
		startChopRoutinesFn:            func(context.Context) error { return nil },
		startBackupRoutinesFn:          func(context.Context) error { return nil },
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
	_, err := startBackgroundServicesWithRuntime(context.Background(), true, nil, startBackgroundServicesRuntime{})
	if err == nil {
		t.Fatal("expected validation failure when required background services are missing")
	}
	if !strings.Contains(err.Error(), "start background services runtime missing required callbacks") {
		t.Fatalf("unexpected error: %v", err)
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
