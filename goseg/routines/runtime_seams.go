package routines

import (
	"groundseg/broadcast"
	"groundseg/chopsvc"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/handler"
	"groundseg/structs"
	"net/http"
	"time"
)

type routineRuntime struct {
	getContainerState    func() map[string]structs.ContainerState
	updateContainerState func(string, structs.ContainerState)
	getState             func() structs.AuthBroadcast
	updateBroadcast      func(structs.AuthBroadcast)
	broadcastClients     func() error
	updateWgOn           func(bool) error
	syncVersionInfo      func() (structs.Channel, bool)
	getVersionChannel    func() structs.Channel
	setVersionChannel    func(structs.Channel)
	updateDocker         func(string, structs.Channel, structs.Channel)
	updateBinary         func(versionRuntime, string, structs.Channel)
	getSha256            func(string) (string, error)
	getConf              func() structs.SysConfig
	getShipStatus        func([]string) (map[string]string, error)
	getShipSettings      func() config.ShipSettings
	getCheck502Settings  func() config.Check502Settings
	startContainer       func(string, string) (structs.ContainerState, error)
	stopContainer        func(string) error
	loadUrbitConfig      func(string) error
	urbitConf            func(string) structs.UrbitDocker
	waitComplete         func(string) error
	chopPier             func(string, structs.UrbitDocker) error
	updateUrbit          func(string, func(*structs.UrbitDocker) error) error
	barExit              func(string) error
	sleep                func(time.Duration)
	basePath             func() string
	architecture         func() string
	debugMode            func() bool
	clearLusCode         func(string)
	getContainerNetwork  func(string) (string, error)
	getLusCode           func(string) (string, error)
	httpGet              func(string) (*http.Response, error)
	recoverWireguard     func([]string, bool) error
}

func newRoutineRuntime() routineRuntime {
	return routineRuntime{
		getContainerState:    config.GetContainerState,
		updateContainerState: config.UpdateContainerState,
		getState:             broadcast.GetState,
		updateBroadcast:      broadcast.UpdateBroadcast,
		broadcastClients:     broadcast.BroadcastToClients,
		updateWgOn: func(wgOn bool) error {
			return config.UpdateConfTyped(config.WithWgOn(wgOn))
		},
		syncVersionInfo:     config.SyncVersionInfo,
		getVersionChannel:   config.GetVersionChannel,
		setVersionChannel:   config.SetVersionChannel,
		updateDocker:        updateDocker,
		updateBinary:        updateBinary,
		getSha256:           getSha256,
		getConf:             config.Conf,
		getShipStatus:       docker.GetShipStatus,
		getShipSettings:     config.ShipSettingsSnapshot,
		getCheck502Settings: config.Check502SettingsSnapshot,
		startContainer:      docker.StartContainer,
		stopContainer:       docker.StopContainerByName,
		loadUrbitConfig:     config.LoadUrbitConfig,
		urbitConf:           config.UrbitConf,
		waitComplete:        handler.WaitComplete,
		chopPier:            chopsvc.ChopPier,
		updateUrbit:         config.UpdateUrbit,
		barExit:             click.BarExit,
		sleep:               time.Sleep,
		basePath:            func() string { return config.BasePath },
		architecture:        func() string { return config.Architecture },
		debugMode:           func() bool { return config.DebugMode },
		clearLusCode:        click.ClearLusCode,
		getContainerNetwork: docker.GetContainerNetwork,
		getLusCode:          click.GetLusCode,
		httpGet:             http.Get,
		recoverWireguard:    handler.RecoverWireguardFleet,
	}
}
