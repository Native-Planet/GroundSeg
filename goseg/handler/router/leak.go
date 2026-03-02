package router

import (
	"encoding/json"
	"fmt"
	"groundseg/handler/authsvc"
	"groundseg/handler/ship"
	"groundseg/handler/system"
	"groundseg/leakchannel"
	"groundseg/shipworkflow"
	"groundseg/structs"

	"go.uber.org/zap"
)

var (
	urbitHandlerForLeak    = ship.UrbitHandler
	penpaiHandlerForLeak   = system.PenpaiHandler
	newShipHandlerForLeak  = handleNewShipForLeak
	systemHandlerForLeak   = system.SystemHandler
	startramHandlerForLeak = system.StartramHandler
	supportHandlerForLeak  = system.SupportHandler
	pwHandlerForLeak       = authsvc.PwHandler
)

func handleNewShipForLeak(payload []byte) error {
	return shipworkflow.HandleNewShip(payload, shipworkflow.HandleNewShipBoot, shipworkflow.CancelNewShip, shipworkflow.ResetNewShip)
}

func HandleLeakAction() {
	// no:
	// pier upload
	// password/login stuff
	// logs
	for {
		action := <-leakchannel.LeakAction
		safelyHandleLeakAction(action)
	}
}

func safelyHandleLeakAction(action leakchannel.ActionChannel) {
	if action.Auth {
		safely(func() {
			gallsegAuthedHandler(action)
		}, "gallsegAuthedHandler", action)
		return
	}
	safely(func() {
		gallsegUnauthHandler(action)
	}, "gallsegUnauthHandler", action)
}

func safely(fn func(), handlerName string, action leakchannel.ActionChannel) {
	defer func() {
		if recovered := recover(); recovered != nil {
			zap.L().Error(fmt.Sprintf("Recovered panic in %s for action=%s patp=%s: %v", handlerName, action.Type, action.Patp, recovered))
		}
	}()
	fn()
}

func Initialize() {
	go HandleLeakAction()
}

func gallsegUnauthHandler(action leakchannel.ActionChannel) {
	var urbitPayload structs.WsUrbitPayload
	err := json.Unmarshal(action.Content, &urbitPayload)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't unmarshal urbit payload from %s's gallseg: %v", action.Patp, err))
		return
	}
	if urbitPayload.Payload.Type != "urbit" {
		return
	}
	if urbitPayload.Payload.Patp != action.Patp {
		return
	}
	safely(func() {
		if err := urbitHandlerForLeak(action.Content); err != nil {
			zap.L().Error(fmt.Sprintf("%+v", err))
		}
	}, "urbitHandlerForLeak", action)
}

func gallsegAuthedHandler(action leakchannel.ActionChannel) {
	switch action.Type {
	case "urbit":
		safely(func() {
			if err := urbitHandlerForLeak(action.Content); err != nil {
				zap.L().Error(fmt.Sprintf("%+v", err))
			}
		}, "urbitHandlerForLeak", action)
	case "penpai":
		safely(func() {
			if err := penpaiHandlerForLeak(action.Content); err != nil {
				zap.L().Error(fmt.Sprintf("%v", err))
			}
		}, "penpaiHandlerForLeak", action)
	case "new_ship":
		safely(func() {
			if err := newShipHandlerForLeak(action.Content); err != nil {
				zap.L().Error(fmt.Sprintf("%v", err))
			}
		}, "newShipHandlerForLeak", action)
	case "system":
		safely(func() {
			if err := systemHandlerForLeak(action.Content); err != nil {
				zap.L().Error(fmt.Sprintf("%v", err))
			}
		}, "systemHandlerForLeak", action)
	case "startram":
		safely(func() {
			if err := startramHandlerForLeak(action.Content); err != nil {
				zap.L().Error(fmt.Sprintf("%v", err))
			}
		}, "startramHandlerForLeak", action)
	case "support":
		safely(func() {
			if err := supportHandlerForLeak(action.Content); err != nil {
				zap.L().Error(fmt.Sprintf("Error creating bug report: %v", err))
			}
		}, "supportHandlerForLeak", action)
	case "password":
		safely(func() {
			if err := pwHandlerForLeak(action.Content, true); err != nil {
				zap.L().Error(fmt.Sprintf("%v", err))
			}
		}, "pwHandlerForLeak", action)
	default:
		zap.L().Error(fmt.Sprintf("Invalid gallseg action: %v", action.Type))
	}
}
