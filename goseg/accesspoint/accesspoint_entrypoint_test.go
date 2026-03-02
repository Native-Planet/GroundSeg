package accesspoint

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStartRunsDependenciesAndWritesHostapdConfig(t *testing.T) {
	origWlan := wlan
	origHostapdPath := hostapdConfigPath
	origDependencies := checkDependenciesFn
	origParams := checkParametersFn
	origWriter := writeHostapdConfigFn
	origExec := executeRouterShellFn
	origSleep := routerSleepFn
	origHostapd := hostapdConfigPath

	t.Cleanup(func() {
		wlan = origWlan
		hostapdConfigPath = origHostapdPath
		checkDependenciesFn = origDependencies
		checkParametersFn = origParams
		writeHostapdConfigFn = origWriter
		executeRouterShellFn = origExec
		routerSleepFn = origSleep
		os.Remove(origHostapd)
	})

	var wrotePath string
	wroteConfig := false
	dependenciesCalled := false

	wlan = "wlan_test"
	hostapdConfigPath = "/tmp/groundseg-hostapd.config"
	routerSleepFn = func(_ time.Duration) {}
	executeRouterShellFn = func(_ string) (string, error) { return "", nil }
	checkDependenciesFn = func() error {
		dependenciesCalled = true
		return nil
	}
	checkParametersFn = func() error {
		return nil
	}
	writeHostapdConfigFn = func(path, gotWlan, gotSSID, gotPassword string) error {
		wroteConfig = true
		wrotePath = path
		if gotWlan != "wlan_test" || gotSSID == "" || !strings.HasPrefix(gotPassword, "np-") {
			t.Fatalf("unexpected hostapd config inputs: wlan=%s ssid=%s password=%s", gotWlan, gotSSID, gotPassword)
		}
		return nil
	}

	if err := Start("wlan_test"); err != nil {
		t.Fatalf("Start returned unexpected error: %v", err)
	}
	if !dependenciesCalled {
		t.Fatal("expected dependency check to run")
	}
	if !wroteConfig {
		t.Fatal("expected hostapd config write hook to run")
	}
	if wrotePath != hostapdConfigPath {
		t.Fatalf("expected config path %q got %q", hostapdConfigPath, wrotePath)
	}
}

func TestStartNoopsWhenAlreadyRunning(t *testing.T) {
	origParameters := checkParametersFn
	origRunning := isRunningFn
	origDependencies := checkDependenciesFn
	origWriteConfig := writeHostapdConfigFn
	origWlan := wlan
	origHostapdPath := hostapdConfigPath
	t.Cleanup(func() {
		checkParametersFn = origParameters
		isRunningFn = origRunning
		checkDependenciesFn = origDependencies
		writeHostapdConfigFn = origWriteConfig
		wlan = origWlan
		hostapdConfigPath = origHostapdPath
	})

	wlan = "wlan_test"
	hostapdConfigPath = "/tmp/groundseg-hostapd.config"
	runningChecked := false
	startCalled := 0
	writeCalled := 0

	checkDependenciesFn = func() error { return nil }
	checkParametersFn = func() error { return nil }
	isRunningFn = func() (bool, error) {
		runningChecked = true
		return true, nil
	}
	writeHostapdConfigFn = func(path, _, _, _ string) error {
		writeCalled++
		return nil
	}
	rt := accessPointRuntime()
	rt.Wlan = "wlan_test"
	rt.RootDir = t.TempDir()
	rt.HostapdConfigPath = ""
	rt.CheckDependenciesFn = nil
	rt.CheckDependenciesFn = checkDependenciesFn
	rt.CheckParametersFn = func(_ AccessPointRuntime) error { return nil }
	rt.EnsureRootDirFn = func(string) error { return nil }
	rt.IsRunningFn = func(AccessPointRuntime) (bool, error) {
		runningChecked = true
		return true, nil
	}
	rt.WriteHostapdConfigFn = func(path, _, _, _ string) error {
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
	origDependencies := checkDependenciesFn
	origParams := checkParametersFn
	origExec := executeRouterShellFn
	origSleep := routerSleepFn
	t.Cleanup(func() {
		checkDependenciesFn = origDependencies
		checkParametersFn = origParams
		executeRouterShellFn = origExec
		routerSleepFn = origSleep
	})

	wlan = "wlan0"
	checkDependenciesFn = func() error { return depErr }
	checkParametersFn = func() error { return nil }
	executeRouterShellFn = func(_ string) (string, error) { return "", nil }
	routerSleepFn = func(_ time.Duration) {}

	err := Start("wlan0")
	if err == nil {
		t.Fatal("expected Start to return dependency failure")
	}
	if !strings.Contains(err.Error(), depErr.Error()) {
		t.Fatalf("expected dependency error in output, got: %v", err)
	}
}

func TestStopSkipsStopWhenNotRunning(t *testing.T) {
	origParameters := checkParametersFn
	origIsRunning := isRunningFn
	origExec := executeRouterShellFn
	origSleep := routerSleepFn
	t.Cleanup(func() {
		checkParametersFn = origParameters
		isRunningFn = origIsRunning
		executeRouterShellFn = origExec
		routerSleepFn = origSleep
	})

	stopCommands := 0
	isRunningFn = func() (bool, error) { return false, nil }
	checkParametersFn = func() error { return nil }
	executeRouterShellFn = func(_ string) (string, error) {
		stopCommands++
		return "", nil
	}
	routerSleepFn = func(_ time.Duration) {}

	if err := Stop("wlan0"); err != nil {
		t.Fatalf("Stop should no-op when not running: %v", err)
	}
	if stopCommands != 0 {
		t.Fatalf("expected no stop command when router is not running, got %d commands", stopCommands)
	}
}
