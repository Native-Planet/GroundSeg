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
	runtimeOps   dockerRuntimeOps
	broadcastOps dockerRoutineBroadcastOps
	wireguardOps dockerRoutineWireguardOps
	systemOps    dockerRoutineSystemOps
	httpOps      dockerRoutineHTTPOps
	timer        dockerRoutineTimer
	recovery     dockerRoutineRecoveryPolicy
}

type dockerRuntimeOps struct {
	LoadUrbitConfigFn          func(string) error
	UrbitConfFn                func(string) structs.UrbitDocker
	ClearLusCodeFn             func(string)
	StartContainerFn           func(string, string) (structs.ContainerState, error)
	GetContainerStateFn        func() map[string]structs.ContainerState
	UpdateContainerStateFn     func(string, structs.ContainerState)
	Check502SettingsSnapshotFn func() dockerCheck502Settings
	GetShipStatusFn            func([]string) (map[string]string, error)
	GetContainerNetworkFn      func(string) (string, error)
	GetLusCodeFn               func(string) (string, error)
	ShipSettingsSnapshotFn     func() dockerShipSettings
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
	recoverWireguardAfter502Fn  func(dockerRoutineRuntime, dockerCheck502Settings) error
}

type dockerRoutineTimer struct {
	sleepFn func(time.Duration)
}

type dockerCheck502Settings struct {
	Piers      []string
	WgOn       bool
	Disable502 bool
}

type dockerShipSettings struct {
	Piers []string
}

func newDockerRoutineRuntime() dockerRoutineRuntime {
	orch := orchestration.NewRuntime()
	return dockerRoutineRuntime{
		runtimeOps: dockerRuntimeOps{
			LoadUrbitConfigFn:      orch.LoadUrbitConfigFn,
			UrbitConfFn:            orch.UrbitConfFn,
			ClearLusCodeFn:         orch.ClearLusCodeFn,
			StartContainerFn:       orch.StartContainerFn,
			GetContainerStateFn:    orch.GetContainerStateFn,
			UpdateContainerStateFn: orch.UpdateContainerStateFn,
			Check502SettingsSnapshotFn: func() dockerCheck502Settings {
				settings := config.Check502SettingsSnapshot()
				return dockerCheck502Settings{
					Piers:      append([]string(nil), settings.Piers...),
					WgOn:       settings.WgOn,
					Disable502: settings.Disable502,
				}
			},
			GetShipStatusFn:       orch.GetShipStatusFn,
			GetContainerNetworkFn: orch.GetContainerNetworkFn,
			GetLusCodeFn:          orch.GetLusCodeFn,
			ShipSettingsSnapshotFn: func() dockerShipSettings {
				settings := config.ShipSettingsSnapshot()
				return dockerShipSettings{
					Piers: append([]string(nil), settings.Piers...),
				}
			},
		},
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
