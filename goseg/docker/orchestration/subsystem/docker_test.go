package subsystem

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"groundseg/structs"
	"groundseg/transition"
)

type dockerTransitionOpsStub = dockerTransitionOps
type dockerHealthOpsStub = dockerHealthRuntime
type dockerBroadcastOpsStub = dockerRoutineBroadcastOps
type dockerWireguardOpsStub = dockerRoutineWireguardOps
type dockerSystemOpsStub = dockerRoutineSystemOps
type dockerHTTPOpsStub = dockerRoutineHTTPOps
type dockerTimerStub = dockerRoutineTimer

func testDockerRoutineRuntime() dockerRoutineRuntime {
	rt := newDockerRoutineRuntime()
	transitionOps := rt.transitionOps
	transitionOps.GetContainerStateFn = func() map[string]structs.ContainerState { return map[string]structs.ContainerState{} }
	transitionOps.UpdateContainerFn = func(string, structs.ContainerState) {}
	transitionOps.StartContainerFn = func(string, string) (structs.ContainerState, error) {
		return structs.ContainerState{}, nil
	}
	transitionOps.LoadUrbitConfigFn = func(string) error { return nil }
	transitionOps.UrbitConfFn = func(string) structs.UrbitDocker { return structs.UrbitDocker{} }
	transitionOps.ClearLusCodeFn = func(string) {}
	healthOps := rt.healthOps
	healthOps.GetContainerNetworkFn = func(string) (string, error) { return "default", nil }
	healthOps.GetLusCodeFn = func(string) (string, error) { return "", nil }
	healthOps.GetShipStatusFn = func([]string) (map[string]string, error) { return map[string]string{}, nil }
	healthOps.ShipSettingsSnapshotFn = func() dockerShipSettings { return dockerShipSettings{} }
	healthOps.Check502SettingsSnapshotFn = func() dockerCheck502Settings { return dockerCheck502Settings{} }
	rt.transitionOps = transitionOps
	rt.healthOps = healthOps
	rt.broadcastOps = dockerBroadcastOpsStub{
		getBroadcastStateFn: func() structs.AuthBroadcast { return structs.AuthBroadcast{} },
		updateBroadcastFn:   func(structs.AuthBroadcast) {},
		broadcastClientsFn:  func() error { return nil },
		updateWgOnFn:        func(bool) error { return nil },
	}
	rt.wireguardOps = dockerWireguardOpsStub{
		recoverWireguardFn: func([]string, bool) error { return nil },
	}
	rt.systemOps = dockerSystemOpsStub{
		barExitFn: func(string) error { return nil },
	}
	rt.httpOps = dockerHTTPOpsStub{
		getFn: func(string) (*http.Response, error) { return nil, nil },
	}
	rt.timer = dockerTimerStub{
		sleepFn: func(time.Duration) {},
	}
	return rt
}

func TestMakeBroadcastWireguardStartTransitionUsesBroadcastTransition(t *testing.T) {
	rt := testDockerRoutineRuntime()
	state := structs.AuthBroadcast{}
	wgOnUpdates := []bool{}
	broadcastCalls := 0
	transitionOps := rt.transitionOps
	transitionOps.GetContainerStateFn = func() map[string]structs.ContainerState {
		return map[string]structs.ContainerState{
			"wireguard": {
				Type: string(transition.ContainerTypeWireguard),
			},
		}
	}
	transitionOps.UpdateContainerFn = func(string, structs.ContainerState) {}
	rt.transitionOps = transitionOps
	rt.broadcastOps = dockerBroadcastOpsStub{
		getBroadcastStateFn: func() structs.AuthBroadcast {
			return state
		},
		updateBroadcastFn: func(next structs.AuthBroadcast) {
			state = next
		},
		broadcastClientsFn: func() error {
			broadcastCalls++
			return nil
		},
		updateWgOnFn: func(wgOn bool) error {
			wgOnUpdates = append(wgOnUpdates, wgOn)
			return nil
		},
		setStartramRunningFn: func(running bool) error {
			if state.Profile.Startram.Info.Running == running {
				return nil
			}
			state.Profile.Startram.Info.Running = running
			return nil
		},
	}

	if _, err := updateContainerTransition(rt, "wireguard", func(state *structs.ContainerState) error {
		state.ActualStatus = string(transition.ContainerStatusRunning)
		if err := dockerStartAfterTransition(rt, "wireguard", state); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("wireguard start transition failed: %v", err)
	}
	if err := rt.broadcastOps.broadcastClientsFn(); err != nil {
		t.Fatalf("broadcast after start failed: %v", err)
	}
	if state.Profile.Startram.Info.Running != true {
		t.Fatalf("expected startram running state true after wireguard start")
	}
	if broadcastCalls != 1 {
		t.Fatalf("expected broadcast to run once, got %d", broadcastCalls)
	}
	if len(wgOnUpdates) != 1 || !wgOnUpdates[0] {
		t.Fatalf("expected wgOn update to run once with true, got %+v", wgOnUpdates)
	}
}

func TestMakeBroadcastNonWireguardOnlyBroadcasts(t *testing.T) {
	rt := testDockerRoutineRuntime()
	updateWgOnCalled := false
	broadcastCalls := 0
	transitionOps := rt.transitionOps
	transitionOps.GetContainerStateFn = func() map[string]structs.ContainerState {
		return map[string]structs.ContainerState{
			"minio_zod": {
				Type: "minio_zod",
			},
		}
	}
	transitionOps.UpdateContainerFn = func(string, structs.ContainerState) {}
	rt.transitionOps = transitionOps
	rt.broadcastOps = dockerBroadcastOpsStub{
		getBroadcastStateFn: func() structs.AuthBroadcast { return structs.AuthBroadcast{} },
		updateBroadcastFn:   func(structs.AuthBroadcast) {},
		broadcastClientsFn: func() error {
			broadcastCalls++
			return nil
		},
		updateWgOnFn: func(bool) error {
			updateWgOnCalled = true
			return nil
		},
	}

	if _, err := updateContainerTransition(rt, "minio_zod", func(state *structs.ContainerState) error {
		state.ActualStatus = string(transition.ContainerStatusRunning)
		return nil
	}); err != nil {
		t.Fatalf("non-wireguard transition failed: %v", err)
	}
	if err := rt.broadcastOps.broadcastClientsFn(); err != nil {
		t.Fatalf("broadcast failed: %v", err)
	}

	if updateWgOnCalled {
		t.Fatal("wgOn update should not be called for non-wireguard containers")
	}
	if broadcastCalls != 1 {
		t.Fatalf("expected exactly one broadcast, got %d", broadcastCalls)
	}
}

func TestGracefulShipExitStopsRunningShips(t *testing.T) {
	rt := testDockerRoutineRuntime()
	zodStatusChecks := 0
	healthOps := rt.healthOps
	healthOps.GetShipStatusFn = func(piers []string) (map[string]string, error) {
		if len(piers) == 2 {
			return map[string]string{
				"zod": "Up 2 minutes",
				"nec": "Exited",
			}, nil
		}
		if len(piers) == 1 && piers[0] == "zod" {
			zodStatusChecks++
			if zodStatusChecks == 1 {
				return map[string]string{"zod": "Up 1 second"}, nil
			}
			return map[string]string{"zod": "Exited"}, nil
		}
		return map[string]string{}, nil
	}
	healthOps.ShipSettingsSnapshotFn = func() dockerShipSettings {
		return dockerShipSettings{Piers: []string{"zod", "nec"}}
	}
	rt.healthOps = healthOps
	rt.systemOps = dockerSystemOpsStub{
		barExitFn: func(patp string) error {
			if patp != "zod" {
				t.Fatalf("unexpected ship targeted for BarExit: %s", patp)
			}
			return nil
		},
	}
	rt.timer = dockerTimerStub{
		sleepFn: func(time.Duration) {},
	}

	err := gracefulShipExit(rt)
	if err != nil {
		t.Fatalf("GracefulShipExit returned error: %v", err)
	}
	if DisableShipRestart {
		t.Fatal("DisableShipRestart should be reset to false after GracefulShipExit")
	}
}

func TestGracefulShipExitReturnsErrorWhenBarExitFails(t *testing.T) {
	rt := testDockerRoutineRuntime()
	healthOps := rt.healthOps
	healthOps.GetShipStatusFn = func(piers []string) (map[string]string, error) {
		if len(piers) == 1 {
			return map[string]string{"zod": "Up"}, nil
		}
		return map[string]string{}, nil
	}
	healthOps.ShipSettingsSnapshotFn = func() dockerShipSettings {
		return dockerShipSettings{Piers: []string{"zod"}}
	}
	rt.healthOps = healthOps
	rt.systemOps = dockerSystemOpsStub{
		barExitFn: func(string) error { return errors.New("exit failed") },
	}
	rt.timer = dockerTimerStub{
		sleepFn: func(time.Duration) {},
	}

	err := gracefulShipExit(rt)
	if err == nil {
		t.Fatal("expected gracefulShipExit to report bar exit failure")
	}
	if !strings.Contains(err.Error(), "failed to stop zod with |exit for daemon restart") {
		t.Fatalf("unexpected error for bar exit failure: %v", err)
	}
}

func TestGracefulShipExitReturnsErrorOnInitialStatusFailure(t *testing.T) {
	rt := testDockerRoutineRuntime()
	healthOps := rt.healthOps
	healthOps.GetShipStatusFn = func([]string) (map[string]string, error) {
		return nil, errors.New("docker unavailable")
	}
	healthOps.ShipSettingsSnapshotFn = func() dockerShipSettings {
		return dockerShipSettings{Piers: []string{"zod"}}
	}
	rt.healthOps = healthOps
	rt.systemOps = dockerSystemOpsStub{
		barExitFn: func(string) error { return nil },
	}
	rt.timer = dockerTimerStub{
		sleepFn: func(time.Duration) {},
	}

	err := gracefulShipExit(rt)
	if err == nil {
		t.Fatal("expected gracefulShipExit to fail when initial status fetch fails")
	}
	if !strings.Contains(err.Error(), "Failed to retrieve ship information") {
		t.Fatalf("unexpected error for initial status failure: %v", err)
	}
}
