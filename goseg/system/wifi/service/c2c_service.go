package service

import "fmt"

import "groundseg/protocol/actions"

type C2CAction = actions.Action

const (
	ConnectAction C2CAction = actions.ActionC2CConnect
)

type UnsupportedC2CActionError = actions.UnsupportedActionError

// C2CService is responsible for performing captured C2C actions.
type C2CService interface {
	ConnectToWiFi(ssid, password string) error
	RestartGroundSeg() error
	Execute(action C2CAction, ssid, password string) error
}

// C2CCommand is a typed, transport-agnostic command payload.
type C2CCommand struct {
	Action   C2CAction
	SSID     string
	Password string
}

type c2cCommand struct {
	ssid     string
	password string
}

type C2CServiceExecutor struct {
	connectToWiFi    func(string, string) error
	restartGroundSeg func() error
}

func NewC2CServiceForAdapter(connectToWiFi func(string, string) error, restartGroundSeg func() error) C2CServiceExecutor {
	return C2CServiceExecutor{
		connectToWiFi:    connectToWiFi,
		restartGroundSeg: restartGroundSeg,
	}
}

var c2cActionDispatcher = actions.NewActionDispatcher(actions.NamespaceC2C, []actions.ActionBinding[C2CAction, C2CService, c2cCommand]{
	{
		Action: ConnectAction,
		Execute: func(service C2CService, cmd c2cCommand) error {
			if err := service.ConnectToWiFi(cmd.ssid, cmd.password); err != nil {
				return fmt.Errorf("connect to wifi %s: %w", cmd.ssid, err)
			}
			if err := service.RestartGroundSeg(); err != nil {
				return fmt.Errorf("restart groundseg after captive connect: %w", err)
			}
			return nil
		},
	},
})

func SupportedC2CActions() []C2CAction {
	return actions.SupportedC2CActions()
}

func (s C2CServiceExecutor) ConnectToWiFi(ssid, password string) error {
	return s.connectToWiFi(ssid, password)
}

func (s C2CServiceExecutor) RestartGroundSeg() error {
	return s.restartGroundSeg()
}

func (s C2CServiceExecutor) Execute(action C2CAction, ssid, password string) error {
	return c2cActionDispatcher.Execute(action, s, c2cCommand{ssid: ssid, password: password})
}

func ParseC2CAction(raw string) (C2CAction, error) {
	return actions.ParseC2CAction(raw)
}

func ProcessC2CMessage(cmd C2CCommand, serviceFactory func() C2CService) error {
	if serviceFactory == nil {
		return fmt.Errorf("service factory is required")
	}
	service := serviceFactory()
	if service == nil {
		return fmt.Errorf("c2c service is required")
	}
	return service.Execute(cmd.Action, cmd.SSID, cmd.Password)
}
