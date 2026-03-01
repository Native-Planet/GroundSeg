package routines

import (
	"errors"
	"strings"
	"testing"
	"time"

	"groundseg/config"
	"groundseg/structs"
)

func testDockerRoutineRuntime() dockerRoutineRuntime {
	rt := newDockerRoutineRuntime()
	rt.getContainerState = func() map[string]structs.ContainerState { return map[string]structs.ContainerState{} }
	rt.updateContainerState = func(string, structs.ContainerState) {}
	rt.getState = func() structs.AuthBroadcast { return structs.AuthBroadcast{} }
	rt.updateBroadcast = func(structs.AuthBroadcast) {}
	rt.broadcastClients = func() error { return nil }
	rt.updateWgOn = func(bool) error { return nil }
	rt.getShipStatus = func([]string) (map[string]string, error) { return map[string]string{}, nil }
	rt.getShipSettings = func() config.ShipSettings { return config.ShipSettings{} }
	rt.getCheck502Settings = func() config.Check502Settings { return config.Check502Settings{} }
	rt.barExit = func(string) error { return nil }
	rt.sleep = func(time.Duration) {}
	rt.loadUrbitConfig = func(string) error { return nil }
	rt.urbitConf = func(string) structs.UrbitDocker { return structs.UrbitDocker{} }
	rt.clearLusCode = func(string) {}
	rt.startContainer = func(string, string) (structs.ContainerState, error) { return structs.ContainerState{}, nil }
	rt.getContainerNetwork = func(string) (string, error) { return "default", nil }
	rt.getLusCode = func(string) (string, error) { return "", nil }
	rt.httpGet = nil
	rt.recoverWireguard = func([]string, bool) error { return nil }
	return rt
}

func TestMakeBroadcastWireguardStartAndDie(t *testing.T) {
	rt := testDockerRoutineRuntime()
	state := structs.AuthBroadcast{}
	rt.getState = func() structs.AuthBroadcast {
		return state
	}
	rt.updateBroadcast = func(next structs.AuthBroadcast) {
		state = next
	}
	broadcastCalls := 0
	rt.broadcastClients = func() error {
		broadcastCalls++
		return nil
	}
	var wgOnUpdates []bool
	rt.updateWgOn = func(wgOn bool) error {
		wgOnUpdates = append(wgOnUpdates, wgOn)
		return nil
	}

	makeBroadcastWithRuntime(rt, "wireguard", "start")
	if len(wgOnUpdates) != 1 || !wgOnUpdates[0] {
		t.Fatalf("expected wgOn update to true, got %+v", wgOnUpdates)
	}
	if !state.Profile.Startram.Info.Running {
		t.Fatal("expected startram running state true after wireguard start")
	}

	makeBroadcastWithRuntime(rt, "wireguard", "die")
	if len(wgOnUpdates) != 2 || wgOnUpdates[1] {
		t.Fatalf("expected wgOn update to false on die, got %+v", wgOnUpdates)
	}
	if state.Profile.Startram.Info.Running {
		t.Fatal("expected startram running state false after wireguard die")
	}
	if broadcastCalls != 2 {
		t.Fatalf("expected broadcast to run twice, got %d", broadcastCalls)
	}
}

func TestMakeBroadcastNonWireguardOnlyBroadcasts(t *testing.T) {
	rt := testDockerRoutineRuntime()
	updateWgOnCalled := false
	rt.updateWgOn = func(bool) error {
		updateWgOnCalled = true
		return nil
	}
	broadcastCalls := 0
	rt.broadcastClients = func() error {
		broadcastCalls++
		return nil
	}
	rt.getState = func() structs.AuthBroadcast { return structs.AuthBroadcast{} }
	rt.updateBroadcast = func(structs.AuthBroadcast) {}

	makeBroadcastWithRuntime(rt, "minio_zod", "start")
	if updateWgOnCalled {
		t.Fatal("wgOn update should not be called for non-wireguard containers")
	}
	if broadcastCalls != 1 {
		t.Fatalf("expected exactly one broadcast, got %d", broadcastCalls)
	}
}

func TestGracefulShipExitStopsRunningShips(t *testing.T) {
	rt := testDockerRoutineRuntime()
	DisableShipRestart = false
	rt.getShipSettings = func() config.ShipSettings {
		return config.ShipSettings{Piers: []string{"zod", "nec"}}
	}
	zodStatusChecks := 0
	rt.getShipStatus = func(piers []string) (map[string]string, error) {
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

	barExitCalls := 0
	rt.barExit = func(patp string) error {
		if patp != "zod" {
			t.Fatalf("unexpected ship targeted for BarExit: %s", patp)
		}
		barExitCalls++
		return nil
	}
	rt.sleep = func(time.Duration) {}

	if err := gracefulShipExit(rt); err != nil {
		t.Fatalf("GracefulShipExit returned error: %v", err)
	}
	if barExitCalls != 1 {
		t.Fatalf("expected one BarExit call, got %d", barExitCalls)
	}
	if DisableShipRestart {
		t.Fatal("DisableShipRestart should be reset to false after GracefulShipExit")
	}
}

func TestGracefulShipExitReturnsErrorWhenBarExitFails(t *testing.T) {
	rt := testDockerRoutineRuntime()
	rt.getShipSettings = func() config.ShipSettings {
		return config.ShipSettings{Piers: []string{"zod"}}
	}
	rt.getShipStatus = func(piers []string) (map[string]string, error) {
		if len(piers) == 1 {
			return map[string]string{"zod": "Up"}, nil
		}
		return map[string]string{}, nil
	}
	rt.barExit = func(string) error {
		return errors.New("exit failed")
	}
	rt.sleep = func(time.Duration) {}

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
	rt.getShipSettings = func() config.ShipSettings {
		return config.ShipSettings{Piers: []string{"zod"}}
	}
	rt.getShipStatus = func([]string) (map[string]string, error) {
		return nil, errors.New("docker unavailable")
	}
	rt.barExit = func(string) error { return nil }
	rt.sleep = func(time.Duration) {}

	err := gracefulShipExit(rt)
	if err == nil {
		t.Fatal("expected gracefulShipExit to fail when initial status fetch fails")
	}
	if !strings.Contains(err.Error(), "Failed to retrieve ship information") {
		t.Fatalf("unexpected error for initial status failure: %v", err)
	}
}
