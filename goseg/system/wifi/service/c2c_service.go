package service

import (
	"fmt"

	"groundseg/protocol/actions"
)

// C2CService is responsible for performing captured C2C actions.
type C2CService interface {
	ConnectToWiFi(ssid, password string) error
	RestartGroundSeg() error
	Execute(action actions.Action, ssid, password string) error
}

// C2CCommand is a typed, transport-agnostic command payload.
type C2CCommand struct {
	Action   actions.Action
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
	dispatcher       actions.ActionDispatcher[actions.Action, C2CService, c2cCommand]
}

func NewC2CServiceForAdapter(
	connectToWiFi func(string, string) error,
	restartGroundSeg func() error,
) (C2CService, error) {
	if connectToWiFi == nil {
		return nil, fmt.Errorf("connectToWiFi callback is required")
	}
	if restartGroundSeg == nil {
		return nil, fmt.Errorf("restartGroundSeg callback is required")
	}
	return &C2CServiceExecutor{
		connectToWiFi:    connectToWiFi,
		restartGroundSeg: restartGroundSeg,
		dispatcher:       NewC2CDispatcher(),
	}, nil
}

func NewC2CDispatcher() actions.ActionDispatcher[actions.Action, C2CService, c2cCommand] {
	return actions.NewActionDispatcher(actions.NamespaceC2C, []actions.ActionBinding[actions.Action, C2CService, c2cCommand]{
		{
			Action: actions.ActionC2CConnect,
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
}

func (s C2CServiceExecutor) ConnectToWiFi(ssid, password string) error {
	if s.connectToWiFi == nil {
		return fmt.Errorf("connectToWiFi callback is not configured")
	}
	return s.connectToWiFi(ssid, password)
}

func (s C2CServiceExecutor) RestartGroundSeg() error {
	if s.restartGroundSeg == nil {
		return fmt.Errorf("restartGroundSeg callback is not configured")
	}
	return s.restartGroundSeg()
}

func (s C2CServiceExecutor) Execute(action actions.Action, ssid, password string) error {
	_, err := actions.ParseC2CAction(string(action))
	if err != nil {
		return err
	}
	if len(s.dispatcher.Supported()) == 0 {
		return fmt.Errorf("c2c action dispatcher is not configured")
	}
	return s.dispatcher.Execute(action, s, c2cCommand{ssid: ssid, password: password})
}

func ProcessC2CMessage(cmd C2CCommand, serviceFactory func() C2CService) error {
	if serviceFactory == nil {
		return fmt.Errorf("service factory is required")
	}
	action, err := actions.ParseC2CAction(string(cmd.Action))
	if err != nil {
		return err
	}
	service := serviceFactory()
	if service == nil {
		return fmt.Errorf("c2c service is required")
	}
	return service.Execute(action, cmd.SSID, cmd.Password)
}
