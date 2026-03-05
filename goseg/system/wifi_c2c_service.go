package system

import (
	"encoding/json"
	"fmt"

	"groundseg/protocol/actions"
	"groundseg/structs"
	wifiService "groundseg/system/wifi/service"
)

var c2cActionConnect = actions.ActionC2CConnect

var supportedC2CActions = func() []actions.Action {
	supported, err := actions.SupportedActions(actions.NamespaceC2C)
	if err != nil {
		return nil
	}
	return supported
}

type c2cServiceDeps struct {
	connectToWiFi    func(string, string) error
	restartGroundSeg func() error
}

func processC2CMessageForAdapter(msg []byte) error {
	deps := defaultC2CServiceDeps()
	return processC2CMessageForAdapterWithDeps(msg, deps)
}

func processC2CMessageForAdapterWithDeps(msg []byte, deps c2cServiceDeps) error {
	var payload structs.WsC2CPayload
	if err := json.Unmarshal(msg, &payload); err != nil {
		return fmt.Errorf("unmarshal c2c payload: %w", err)
	}
	if payload.Type != "c2c" {
		return fmt.Errorf("unsupported c2c payload type: %q", payload.Type)
	}
	action, err := actions.ParseAction(actions.NamespaceC2C, payload.Payload.Action)
	if err != nil {
		return err
	}
	cmd := wifiService.C2CCommand{
		Action:   action,
		SSID:     payload.Payload.SSID,
		Password: payload.Payload.Password,
	}
	return wifiService.ProcessC2CMessage(cmd, newC2CServiceFactory(deps))
}

func defaultC2CServiceDeps() c2cServiceDeps {
	return c2cServiceDeps{
		connectToWiFi:    func(ssid, password string) error { return NewWiFiRuntimeService().ConnectToWiFi(ssid, password) },
		restartGroundSeg: restartGroundSegService,
	}
}

func c2cActionExecutorForAdapter(action actions.Action, ssid, password string) error {
	service, err := newC2CServiceForAdapterWithDeps(defaultC2CServiceDeps())
	if err != nil {
		return err
	}
	return service.Execute(action, ssid, password)
}

func newC2CServiceForAdapter() (wifiService.C2CService, error) {
	return newC2CServiceForAdapterWithDeps(defaultC2CServiceDeps())
}

func newC2CServiceForAdapterWithDeps(deps c2cServiceDeps) (wifiService.C2CService, error) {
	if deps.connectToWiFi == nil {
		return nil, fmt.Errorf("newC2CServiceForAdapterWithDeps called without connectToWiFi")
	}
	if deps.restartGroundSeg == nil {
		return nil, fmt.Errorf("newC2CServiceForAdapterWithDeps called without restartGroundSeg")
	}
	return wifiService.NewC2CServiceForAdapter(deps.connectToWiFi, deps.restartGroundSeg)
}

func newC2CServiceFactory(deps c2cServiceDeps) func() wifiService.C2CService {
	return func() wifiService.C2CService {
		service, err := newC2CServiceForAdapterWithDeps(deps)
		if err != nil {
			return nil
		}
		return service
	}
}

func restartGroundSegService() error {
	if _, err := runCommand("systemctl", "restart", "groundseg"); err != nil {
		return fmt.Errorf("restart groundseg after captive connect: %w", err)
	}
	return nil
}
