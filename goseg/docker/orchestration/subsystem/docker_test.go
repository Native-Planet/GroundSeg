package subsystem

import (
	"errors"
	"strings"
	"testing"
	"time"

	"groundseg/config"
	"groundseg/structs"
	"groundseg/transition"
)

type dockerBroadcastOpsStub = dockerRoutineBroadcastOps
type dockerWireguardOpsStub = dockerRoutineWireguardOps
type dockerSystemOpsStub = dockerRoutineSystemOps
type dockerHTTPOpsStub = dockerRoutineHTTPOps
type dockerTimerStub = dockerRoutineTimer

type dockerRoutineRuntimeOption func(*dockerRoutineRuntime)

func testDockerRoutineRuntime(overrides ...dockerRoutineRuntimeOption) dockerRoutineRuntime {
	rt := newDockerRoutineRuntimeForTests()
	for _, apply := range overrides {
		apply(&rt)
	}
	return rt
}

func withDockerContainerState(state map[string]structs.ContainerState) dockerRoutineRuntimeOption {
	return func(runtime *dockerRoutineRuntime) {
		runtime.GetContainerStateFn = func() map[string]structs.ContainerState {
			return state
		}
	}
}

func withDockerBroadcastOps(ops dockerBroadcastOpsStub) dockerRoutineRuntimeOption {
	return func(runtime *dockerRoutineRuntime) {
		runtime.broadcastOps = ops
	}
}

func withDockerSystemOps(ops dockerSystemOpsStub) dockerRoutineRuntimeOption {
	return func(runtime *dockerRoutineRuntime) {
		runtime.systemOps = ops
	}
}

func withDockerShipStatusFn(fn func([]string) (map[string]string, error)) dockerRoutineRuntimeOption {
	return func(runtime *dockerRoutineRuntime) {
		runtime.GetShipStatusFn = fn
	}
}

func withDockerShipSettingsSnapshotFn(fn func() config.ShipSettings) dockerRoutineRuntimeOption {
	return func(runtime *dockerRoutineRuntime) {
		runtime.ShipSettingsSnapshotFn = fn
	}
}

func withDockerTimerFn(sleepFn func(time.Duration)) dockerRoutineRuntimeOption {
	return func(runtime *dockerRoutineRuntime) {
		runtime.timer = dockerTimerStub{
			sleepFn: sleepFn,
		}
	}
}

func TestMakeBroadcastWireguardStartTransitionUsesBroadcastTransition(t *testing.T) {
	state := structs.AuthBroadcast{}
	wgOnUpdates := []bool{}
	broadcastCalls := 0
	rt := testDockerRoutineRuntime(
		withDockerContainerState(map[string]structs.ContainerState{
			"wireguard": {
				Type: string(transition.ContainerTypeWireguard),
			},
		}),
		withDockerBroadcastOps(dockerBroadcastOpsStub{
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
		}),
	)

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
	updateWgOnCalled := false
	broadcastCalls := 0
	rt := testDockerRoutineRuntime(
		withDockerContainerState(map[string]structs.ContainerState{
			"minio_zod": {
				Type: "minio_zod",
			},
		}),
		withDockerBroadcastOps(dockerBroadcastOpsStub{
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
		}),
	)

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
	zodStatusChecks := 0
	rt := testDockerRoutineRuntime(
		withDockerShipStatusFn(func(piers []string) (map[string]string, error) {
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
		}),
		withDockerShipSettingsSnapshotFn(func() config.ShipSettings {
			return config.ShipSettings{Piers: []string{"zod", "nec"}}
		}),
		withDockerSystemOps(dockerSystemOpsStub{
			barExitFn: func(patp string) error {
				if patp != "zod" {
					t.Fatalf("unexpected ship targeted for BarExit: %s", patp)
				}
				return nil
			},
		}),
		withDockerTimerFn(func(time.Duration) {}),
	)

	err := gracefulShipExit(rt)
	if err != nil {
		t.Fatalf("GracefulShipExit returned error: %v", err)
	}
	if DisableShipRestart {
		t.Fatal("DisableShipRestart should be reset to false after GracefulShipExit")
	}
}

func TestGracefulShipExitReturnsErrorWhenBarExitFails(t *testing.T) {
	rt := testDockerRoutineRuntime(
		withDockerShipStatusFn(func(piers []string) (map[string]string, error) {
			if len(piers) == 1 {
				return map[string]string{"zod": "Up"}, nil
			}
			return map[string]string{}, nil
		}),
		withDockerShipSettingsSnapshotFn(func() config.ShipSettings {
			return config.ShipSettings{Piers: []string{"zod"}}
		}),
		withDockerSystemOps(dockerSystemOpsStub{
			barExitFn: func(string) error { return errors.New("exit failed") },
		}),
		withDockerTimerFn(func(time.Duration) {}),
	)

	err := gracefulShipExit(rt)
	if err == nil {
		t.Fatal("expected gracefulShipExit to report bar exit failure")
	}
	if !strings.Contains(err.Error(), "failed to stop zod with |exit for daemon restart") {
		t.Fatalf("unexpected error for bar exit failure: %v", err)
	}
}

func TestGracefulShipExitReturnsErrorOnInitialStatusFailure(t *testing.T) {
	rt := testDockerRoutineRuntime(
		withDockerShipStatusFn(func([]string) (map[string]string, error) {
			return nil, errors.New("docker unavailable")
		}),
		withDockerShipSettingsSnapshotFn(func() config.ShipSettings {
			return config.ShipSettings{Piers: []string{"zod"}}
		}),
		withDockerSystemOps(dockerSystemOpsStub{
			barExitFn: func(string) error { return nil },
		}),
		withDockerTimerFn(func(time.Duration) {}),
	)

	err := gracefulShipExit(rt)
	if err == nil {
		t.Fatal("expected gracefulShipExit to fail when initial status fetch fails")
	}
	if !strings.Contains(err.Error(), "Failed to retrieve ship information") {
		t.Fatalf("unexpected error for initial status failure: %v", err)
	}
}
