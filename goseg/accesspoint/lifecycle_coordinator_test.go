package accesspoint

import (
	"errors"
	"testing"
)

func TestLifecycleCoordinatorStartConfiguresRouter(t *testing.T) {
	calls := struct {
		ensureRoot  bool
		checkDeps   bool
		checkParams bool
		isRunning   bool
		writeConfig bool
		startRouter bool
	}{}

	rt := AccessPointRuntime{
		Wlan:              "wlan0",
		HostapdConfigPath: "/tmp/hostapd.config",
		RootDir:           "/tmp/root",
		EnsureRootDirFn: func(_ string) error {
			calls.ensureRoot = true
			return nil
		},
		CheckDependenciesFn: func() error {
			calls.checkDeps = true
			return nil
		},
		CheckParametersFn: func(_ AccessPointRuntime) error {
			calls.checkParams = true
			return nil
		},
		IsRunningFn: func(_ AccessPointRuntime) (bool, error) {
			calls.isRunning = true
			return false, nil
		},
		WriteHostapdConfigFn: func(_ string, _, _, _ string) error {
			calls.writeConfig = true
			return nil
		},
		StartRouterFn: func(_ AccessPointRuntime) error {
			calls.startRouter = true
			return nil
		},
	}

	coordinator := accessPointLifecycleCoordinator{}
	if err := coordinator.Start(rt); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if !calls.ensureRoot || !calls.checkDeps || !calls.checkParams || !calls.isRunning || !calls.writeConfig || !calls.startRouter {
		t.Fatalf("expected every start step, got %+v", calls)
	}
}

func TestLifecycleCoordinatorStartStopsWhenAlreadyRunningAndNoForceRestart(t *testing.T) {
	calls := struct {
		writeConfig bool
		start       bool
	}{}

	rt := AccessPointRuntime{
		Wlan:                "wlan0",
		HostapdConfigPath:   "/tmp/hostapd.config",
		RootDir:             "/tmp/root",
		EnsureRootDirFn:     func(_ string) error { return nil },
		CheckDependenciesFn: func() error { return nil },
		CheckParametersFn:   func(_ AccessPointRuntime) error { return nil },
		IsRunningFn: func(_ AccessPointRuntime) (bool, error) {
			return true, nil
		},
		WriteHostapdConfigFn: func(_ string, _, _, _ string) error {
			calls.writeConfig = true
			return nil
		},
		StartRouterFn: func(_ AccessPointRuntime) error {
			calls.start = true
			return nil
		},
	}

	coordinator := accessPointLifecycleCoordinator{}
	if err := coordinator.Start(rt); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if calls.writeConfig {
		t.Fatal("expected skip config write when already running and not forced")
	}
	if calls.start {
		t.Fatal("expected skip router start when already running and not forced")
	}
}

func TestLifecycleCoordinatorStartReturnsMissingRuntimeError(t *testing.T) {
	rt := AccessPointRuntime{
		Wlan: "wlan0",
	}
	coordinator := accessPointLifecycleCoordinator{}
	if err := coordinator.Start(rt); err == nil {
		t.Fatal("expected missing runtime error")
	}
}

func TestLifecycleCoordinatorStopStopsWhenRunning(t *testing.T) {
	calls := struct {
		checkParams bool
		isRunning   bool
		stopRouter  bool
	}{}

	rt := AccessPointRuntime{
		Wlan: "wlan0",
		CheckParametersFn: func(_ AccessPointRuntime) error {
			calls.checkParams = true
			return nil
		},
		IsRunningFn: func(_ AccessPointRuntime) (bool, error) {
			calls.isRunning = true
			return true, nil
		},
		StopRouterFn: func(_ AccessPointRuntime) error {
			calls.stopRouter = true
			return nil
		},
	}

	coordinator := accessPointLifecycleCoordinator{}
	if err := coordinator.Stop(rt); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if !calls.checkParams || !calls.isRunning || !calls.stopRouter {
		t.Fatalf("expected stop workflow, got %+v", calls)
	}
}

func TestLifecycleCoordinatorStopShortCircuitsWhenNotRunning(t *testing.T) {
	rt := AccessPointRuntime{
		Wlan:              "wlan0",
		CheckParametersFn: func(_ AccessPointRuntime) error { return nil },
		IsRunningFn:       func(_ AccessPointRuntime) (bool, error) { return false, nil },
		StopRouterFn: func(_ AccessPointRuntime) error {
			return errors.New("should not run")
		},
	}
	coordinator := accessPointLifecycleCoordinator{}
	if err := coordinator.Stop(rt); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
}
