package routines

import (
	"groundseg/broadcast"
	"groundseg/chopsvc"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/handler/ship"
	"groundseg/handler/system"
	"groundseg/internal/seams"
	"groundseg/structs"
	"net/http"
	"time"
)

type runtimeBase struct {
	seams.RuntimeBase
}

func newRuntimeBase() runtimeBase { return runtimeBase{RuntimeBase: seams.NewRuntimeBase()} }

type dockerRoutineRuntime struct {
	runtimeBase

	getContainerState    func() map[string]structs.ContainerState
	updateContainerState func(string, structs.ContainerState)
	getState             func() structs.AuthBroadcast
	updateBroadcast      func(structs.AuthBroadcast)
	broadcastClients     func() error
	updateWgOn           func(bool) error
	getShipStatus        func([]string) (map[string]string, error)
	getShipSettings      func() config.ShipSettings
	getCheck502Settings  func() config.Check502Settings
	startContainer       func(string, string) (structs.ContainerState, error)
	loadUrbitConfig      func(string) error
	urbitConf            func(string) structs.UrbitDocker
	clearLusCode         func(string)
	getContainerNetwork  func(string) (string, error)
	getLusCode           func(string) (string, error)
	httpGet              func(string) (*http.Response, error)
	recoverWireguard     func([]string, bool) error
	barExit              func(string) error
	sleep                func(time.Duration)
}

func newDockerRoutineRuntime() dockerRoutineRuntime {
	base := newRuntimeBase()
	return dockerRoutineRuntime{
		runtimeBase:          base,
		getContainerState:    config.GetContainerState,
		updateContainerState: config.UpdateContainerState,
		getState:             broadcast.GetState,
		updateBroadcast:      broadcast.UpdateBroadcast,
		broadcastClients:     broadcast.BroadcastToClients,
		updateWgOn: func(wgOn bool) error {
			return config.UpdateConfTyped(config.WithWgOn(wgOn))
		},
		getShipStatus:       docker.GetShipStatus,
		getShipSettings:     config.ShipSettingsSnapshot,
		getCheck502Settings: config.Check502SettingsSnapshot,
		startContainer:      docker.StartContainer,
		loadUrbitConfig:     config.LoadUrbitConfig,
		urbitConf:           config.UrbitConf,
		clearLusCode:        click.ClearLusCode,
		getContainerNetwork: docker.GetContainerNetwork,
		getLusCode:          click.GetLusCode,
		httpGet:             http.Get,
		recoverWireguard:    system.RecoverWireguardFleet,
		barExit:             click.BarExit,
		sleep:               time.Sleep,
	}
}

type versionRuntime struct {
	runtimeBase

	syncVersionInfo   func() (structs.Channel, bool)
	getVersionChannel func() structs.Channel
	setVersionChannel func(structs.Channel)
	updateDocker      func(string, structs.Channel, structs.Channel)
	updateBinary      func(versionRuntime, string, structs.Channel)
	getSha256         func(string) (string, error)
	getConf           func() structs.SysConfig
	getShipStatus     func([]string) (map[string]string, error)
	startContainer    func(string, string) (structs.ContainerState, error)
	stopContainer     func(string) error
	loadUrbitConfig   func(string) error
	urbitConf         func(string) structs.UrbitDocker
	waitComplete      func(string) error
	chopPier          func(string) error
	updateUrbit       func(string, func(*structs.UrbitDocker) error) error
}

func newVersionRuntime() versionRuntime {
	base := newRuntimeBase()
	return versionRuntime{
		runtimeBase:       base,
		syncVersionInfo:   config.SyncVersionInfo,
		getVersionChannel: config.GetVersionChannel,
		setVersionChannel: config.SetVersionChannel,
		updateDocker:      updateDocker,
		updateBinary:      updateBinary,
		getSha256:         getSha256,
		getConf:           config.Conf,
		getShipStatus:     docker.GetShipStatus,
		startContainer:    docker.StartContainer,
		stopContainer:     docker.StopContainerByName,
		loadUrbitConfig:   config.LoadUrbitConfig,
		urbitConf:         config.UrbitConf,
		waitComplete:      ship.WaitComplete,
		chopPier:          chopsvc.ChopPier,
		updateUrbit:       config.UpdateUrbit,
	}
}
