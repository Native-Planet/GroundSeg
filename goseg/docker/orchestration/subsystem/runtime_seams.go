package subsystem

import (
	"net/http"
	"time"

	"groundseg/broadcast"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker/orchestration"
	"groundseg/handler/system"
	"groundseg/structs"
)

type dockerRoutineRuntime struct {
	orchestration.Runtime
	broadcastOps dockerRoutineBroadcastOps
	wireguardOps dockerRoutineWireguardOps
	systemOps    dockerRoutineSystemOps
	httpOps      dockerRoutineHTTPOps
	timer        dockerRoutineTimer
	recovery     dockerRoutineRecoveryPolicy
}

type dockerRoutineBroadcastOps struct {
	getBroadcastStateFn  func() structs.AuthBroadcast
	updateBroadcastFn    func(structs.AuthBroadcast)
	broadcastClientsFn   func() error
	updateWgOnFn         func(bool) error
	setStartramRunningFn func(bool) error
}

type dockerRoutineWireguardOps struct {
	recoverWireguardFn func([]string, bool) error
}

type dockerRoutineSystemOps struct {
	barExitFn func(string) error
}

type dockerRoutineHTTPOps struct {
	getFn func(string) (*http.Response, error)
}

type dockerRoutineRecoveryPolicy struct {
	stopTransitionRestartFn     func(dockerRoutineRuntime, string, *structs.ContainerState) error
	restartDelay                time.Duration
	restartAfterDeathFn         func(dockerRoutineRuntime, string, string) error
	check502InitialDelay        time.Duration
	check502PollDelay           time.Duration
	check502ConsecutiveFailures int
	recoverWireguardAfter502Fn  func(dockerRoutineRuntime, config.Check502Settings) error
}

type dockerRoutineTimer struct {
	sleepFn func(time.Duration)
}

func newDockerRoutineRuntime() dockerRoutineRuntime {
	orch := orchestration.NewRuntime()
	return dockerRoutineRuntime{
		Runtime: orch,
		broadcastOps: dockerRoutineBroadcastOps{
			getBroadcastStateFn: func() structs.AuthBroadcast { return broadcast.DefaultBroadcastStateRuntime().GetState() },
			updateBroadcastFn:   func(state structs.AuthBroadcast) { broadcast.DefaultBroadcastStateRuntime().UpdateBroadcast(state) },
			broadcastClientsFn:  broadcast.BroadcastToClients,
			updateWgOnFn: func(enabled bool) error {
				return config.UpdateConfigTyped(config.WithWgOn(enabled))
			},
			setStartramRunningFn: broadcast.SetStartramRunning,
		},
		wireguardOps: dockerRoutineWireguardOps{
			recoverWireguardFn: system.RecoverWireguardFleet,
		},
		systemOps: dockerRoutineSystemOps{
			barExitFn: click.BarExit,
		},
		httpOps: dockerRoutineHTTPOps{
			getFn: http.Get,
		},
		timer: dockerRoutineTimer{
			sleepFn: time.Sleep,
		},
		recovery: dockerRoutineRecoveryPolicy{
			stopTransitionRestartFn:     defaultStopTransitionRestart,
			restartDelay:                2 * time.Second,
			restartAfterDeathFn:         defaultRestartAfterDeath,
			check502InitialDelay:        180 * time.Second,
			check502PollDelay:           120 * time.Second,
			check502ConsecutiveFailures: 2,
			recoverWireguardAfter502Fn:  defaultRecoverWireguardAfter502,
		},
	}
}

func newDockerRoutineRuntimeForTests() dockerRoutineRuntime {
	rt := newDockerRoutineRuntime()
	rt.GetContainerStateFn = func() map[string]structs.ContainerState { return map[string]structs.ContainerState{} }
	rt.UpdateContainerStateFn = func(string, structs.ContainerState) {}
	rt.StartContainerFn = func(string, string) (structs.ContainerState, error) {
		return structs.ContainerState{}, nil
	}
	rt.GetContainerNetworkFn = func(string) (string, error) { return "default", nil }
	rt.GetLusCodeFn = func(string) (string, error) { return "", nil }
	rt.GetShipStatusFn = func([]string) (map[string]string, error) { return map[string]string{}, nil }
	rt.LoadUrbitConfigFn = func(string) error { return nil }
	rt.UrbitConfFn = func(string) structs.UrbitDocker { return structs.UrbitDocker{} }
	rt.ClearLusCodeFn = func(string) {}
	rt.ShipSettingsSnapshotFn = func() config.ShipSettings { return config.ShipSettings{} }
	rt.Check502SettingsSnapshotFn = func() config.Check502Settings { return config.Check502Settings{} }
	rt.broadcastOps = dockerRoutineBroadcastOps{
		getBroadcastStateFn:  func() structs.AuthBroadcast { return structs.AuthBroadcast{} },
		updateBroadcastFn:    func(structs.AuthBroadcast) {},
		broadcastClientsFn:   func() error { return nil },
		updateWgOnFn:         func(bool) error { return nil },
		setStartramRunningFn: func(bool) error { return nil },
	}
	rt.wireguardOps = dockerRoutineWireguardOps{
		recoverWireguardFn: func([]string, bool) error { return nil },
	}
	rt.systemOps = dockerRoutineSystemOps{
		barExitFn: func(string) error { return nil },
	}
	rt.httpOps = dockerRoutineHTTPOps{
		getFn: func(string) (*http.Response, error) { return nil, nil },
	}
	rt.timer = dockerRoutineTimer{
		sleepFn: func(time.Duration) {},
	}
	return rt
}
