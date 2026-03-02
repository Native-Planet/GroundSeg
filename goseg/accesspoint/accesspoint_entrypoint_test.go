package accesspoint

import (
	"errors"
	"strings"
	"testing"
)

func TestStartRunsDependenciesAndWritesHostapdConfig(t *testing.T) {
	rt := accessPointRuntime()
	rt.Wlan = "wlan_test"
	rt.RootDir = t.TempDir()
	rt.HostapdConfigPath = "/tmp/groundseg-hostapd.config"

	var wrotePath string
	wroteConfig := false
	dependenciesCalled := false
	rt.CheckDependenciesFn = func() error {
		dependenciesCalled = true
		return nil
	}
	rt.CheckParametersFn = func(_ AccessPointRuntime) error {
		return nil
	}
	rt.IsRunningFn = func(_ AccessPointRuntime) (bool, error) {
		return false, nil
	}
	rt.WriteHostapdConfigFn = func(path, gotWlan, gotSSID, gotPassword string) error {
		wroteConfig = true
		wrotePath = path
		if gotWlan != "wlan_test" || gotSSID == "" || !strings.HasPrefix(gotPassword, "np-") {
			t.Fatalf("unexpected hostapd config inputs: wlan=%s ssid=%s password=%s", gotWlan, gotSSID, gotPassword)
		}
		return nil
	}
	rt.StartRouterFn = func(_ AccessPointRuntime) error { return nil }

	if err := StartWithRuntime(rt); err != nil {
		t.Fatalf("Start returned unexpected error: %v", err)
	}
	if !dependenciesCalled {
		t.Fatal("expected dependency check to run")
	}
	if !wroteConfig {
		t.Fatal("expected hostapd config write hook to run")
	}
	if wrotePath != rt.HostapdConfigPath {
		t.Fatalf("expected config path %q got %q", rt.HostapdConfigPath, wrotePath)
	}
}

func TestStartNoopsWhenAlreadyRunning(t *testing.T) {
	rt := accessPointRuntime()
	rt.Wlan = "wlan_test"
	rt.RootDir = t.TempDir()
	rt.HostapdConfigPath = "/tmp/groundseg-hostapd.config"

	runningChecked := false
	startCalled := 0
	writeCalled := 0

	rt.CheckDependenciesFn = func() error { return nil }
	rt.CheckParametersFn = func(_ AccessPointRuntime) error { return nil }
	rt.IsRunningFn = func(_ AccessPointRuntime) (bool, error) {
		runningChecked = true
		return true, nil
	}
	rt.WriteHostapdConfigFn = func(_ string, _ string, _ string, _ string) error {
		writeCalled++
		return nil
	}
	rt.StartRouterFn = func(_ AccessPointRuntime) error {
		startCalled++
		return nil
	}
	rt.StopRouterFn = func(_ AccessPointRuntime) error { return nil }

	if err := StartWithRuntime(rt); err != nil {
		t.Fatalf("Start returned unexpected error: %v", err)
	}
	if !runningChecked {
		t.Fatal("expected running check")
	}
	if writeCalled != 0 {
		t.Fatalf("expected no hostapd write when already running, got %d", writeCalled)
	}
	if startCalled != 0 {
		t.Fatalf("expected no router start when already running, got %d", startCalled)
	}
}

func TestStartPropagatesStartFailureFromDependencies(t *testing.T) {
	depErr := errors.New("missing dependency")

	rt := accessPointRuntime()
	rt.Wlan = "wlan0"
	rt.RootDir = t.TempDir()
	rt.CheckDependenciesFn = func() error { return depErr }
	rt.CheckParametersFn = func(_ AccessPointRuntime) error { return nil }
	rt.IsRunningFn = func(_ AccessPointRuntime) (bool, error) { return false, nil }
	rt.WriteHostapdConfigFn = func(_ string, _ string, _ string, _ string) error { return nil }
	rt.StartRouterFn = func(_ AccessPointRuntime) error { return nil }
	rt.StopRouterFn = func(_ AccessPointRuntime) error { return nil }

	err := StartWithRuntime(rt)
	if err == nil {
		t.Fatal("expected Start to return dependency failure")
	}
	if !strings.Contains(err.Error(), depErr.Error()) {
		t.Fatalf("expected dependency error in output, got: %v", err)
	}
}

func TestStopSkipsStopWhenNotRunning(t *testing.T) {
	rt := accessPointRuntime()
	rt.Wlan = "wlan0"
	rt.RootDir = t.TempDir()

	stopCommands := 0
	rt.IsRunningFn = func(_ AccessPointRuntime) (bool, error) { return false, nil }
	rt.CheckParametersFn = func(_ AccessPointRuntime) error { return nil }
	rt.StopRouterFn = func(_ AccessPointRuntime) error {
		stopCommands++
		return nil
	}
	rt.StartRouterFn = func(_ AccessPointRuntime) error { return nil }

	rt = normalizeAccessPointRuntime(rt)
	if err := StopWithRuntime(rt); err != nil {
		t.Fatalf("Stop should no-op when router is not running: %v", err)
	}
	if stopCommands != 0 {
		t.Fatalf("expected no stop command when router is not running, got %d commands", stopCommands)
	}
}

func TestStartNoopsWhenRouterNotConfigured(t *testing.T) {
	commands := 0
	rt := accessPointRuntime()
	rt.Wlan = "wlan0"
	rt.RootDir = t.TempDir()
	rt.CheckDependenciesFn = func() error { return nil }
	rt.CheckParametersFn = func(_ AccessPointRuntime) error { return nil }
	rt.IsRunningFn = func(_ AccessPointRuntime) (bool, error) { return false, nil }
	rt.WriteHostapdConfigFn = func(_ string, _ string, _ string, _ string) error { return nil }
	rt.StartRouterFn = func(_ AccessPointRuntime) error {
		commands++
		return nil
	}

	if err := StartWithRuntime(rt); err != nil {
		t.Fatalf("expected start without custom commands, got: %v", err)
	}
	if commands != 1 {
		t.Fatalf("expected start command to run once, got %d", commands)
	}
}
