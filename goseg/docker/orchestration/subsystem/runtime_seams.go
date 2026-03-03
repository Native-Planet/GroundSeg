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
			getBroadcastStateFn: broadcast.GetState,
			updateBroadcastFn:   broadcast.UpdateBroadcast,
			broadcastClientsFn:  broadcast.BroadcastToClients,
			updateWgOnFn: func(enabled bool) error {
				return config.UpdateConfTyped(config.WithWgOn(enabled))
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
