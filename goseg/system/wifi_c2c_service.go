package system

import (
	"encoding/json"
	"fmt"

	"groundseg/structs"
	wifiService "groundseg/system/wifi/service"
)

type c2cAction = wifiService.C2CAction

const c2cActionConnect = wifiService.ConnectAction

var supportedC2CActions = wifiService.SupportedC2CActions

type unsupportedC2CActionError = wifiService.UnsupportedC2CActionError

type c2cServiceDeps struct {
	connectToWiFi    func(string, string) error
	restartGroundSeg func() error
}

func processC2CMessageForAdapter(msg []byte) error {
	deps := defaultC2CServiceDeps()
	return processC2CMessageForAdapterWithDeps(msg, deps)
}

func processC2CMessageForAdapterWithDeps(msg []byte, deps c2cServiceDeps) error {
	var payload structs.WsC2cPayload
	if err := json.Unmarshal(msg, &payload); err != nil {
		return fmt.Errorf("unmarshal c2c payload: %w", err)
	}
	if payload.Type != "c2c" {
		return fmt.Errorf("unsupported c2c payload type: %q", payload.Type)
	}
	action, err := parseC2CAction(payload.Payload.Action)
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
		connectToWiFi:    ConnectToWifi,
		restartGroundSeg: restartGroundSegService,
	}
}

func c2cActionExecutorForAdapter(action c2cAction, ssid, password string) error {
	service := newC2CServiceForAdapterWithDeps(defaultC2CServiceDeps())
	return service.Execute(action, ssid, password)
}

func parseC2CAction(raw string) (c2cAction, error) {
	return wifiService.ParseC2CAction(raw)
}

func newC2CServiceForAdapter() wifiService.C2CService {
	return newC2CServiceForAdapterWithDeps(defaultC2CServiceDeps())
}

func newC2CServiceForAdapterWithDeps(deps c2cServiceDeps) wifiService.C2CService {
	if deps.connectToWiFi == nil {
		panic("newC2CServiceForAdapterWithDeps called without connectToWiFi")
	}
	if deps.restartGroundSeg == nil {
		panic("newC2CServiceForAdapterWithDeps called without restartGroundSeg")
	}
	return wifiService.NewC2CServiceForAdapter(deps.connectToWiFi, deps.restartGroundSeg)
}

func newC2CServiceFactory(deps c2cServiceDeps) func() wifiService.C2CService {
	return func() wifiService.C2CService {
		return newC2CServiceForAdapterWithDeps(deps)
	}
}

func restartGroundSegService() error {
	if _, err := runCommand("systemctl", "restart", "groundseg"); err != nil {
		return fmt.Errorf("restart groundseg after captive connect: %w", err)
	}
	return nil
}
