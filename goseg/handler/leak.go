package handler

import (
	"encoding/json"
	"fmt"
	"groundseg/leakchannel"
	"groundseg/structs"

	"go.uber.org/zap"
)

var (
	urbitHandlerForLeak    = UrbitHandler
	penpaiHandlerForLeak   = PenpaiHandler
	newShipHandlerForLeak  = NewShipHandler
	systemHandlerForLeak   = SystemHandler
	startramHandlerForLeak = StartramHandler
	supportHandlerForLeak  = SupportHandler
	pwHandlerForLeak       = PwHandler
)

func HandleLeakAction() {
	// no:
	// pier upload
	// password/login stuff
	// logs
	for {
		action := <-leakchannel.LeakAction
		if action.Auth {
			gallsegAuthedHandler(action)
			continue
		}
		gallsegUnauthHandler(action)
	}
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
	if err := urbitHandlerForLeak(action.Content); err != nil {
		zap.L().Error(fmt.Sprintf("%+v", err))
		return
	}
}

func gallsegAuthedHandler(action leakchannel.ActionChannel) {
	switch action.Type {
	case "urbit":
		if err := urbitHandlerForLeak(action.Content); err != nil {
			zap.L().Error(fmt.Sprintf("%+v", err))
		}
	case "penpai":
		if err := penpaiHandlerForLeak(action.Content); err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		}
	case "new_ship":
		if err := newShipHandlerForLeak(action.Content); err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		}
	case "system":
		if err := systemHandlerForLeak(action.Content); err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		}
	case "startram":
		if err := startramHandlerForLeak(action.Content); err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		}
	case "support":
		if err := supportHandlerForLeak(action.Content); err != nil {
			zap.L().Error(fmt.Sprintf("Error creating bug report: %v", err))
		}
	case "password":
		if err := pwHandlerForLeak(action.Content, true); err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		}
	default:
		zap.L().Error(fmt.Sprintf("Invalid gallseg action: %v", action.Type))
	}
}
