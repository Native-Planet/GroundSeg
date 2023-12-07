package handler

import (
	"encoding/json"
	"fmt"
	"goseg/leakchannel"
	"goseg/logger"
	"goseg/structs"
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
		logger.Logger.Error(fmt.Sprintf("Couldn't unmarshal urbit payload from %s's gallseg: %v", action.Patp, err))
		return
	}
	if urbitPayload.Payload.Type != "urbit" {
		return
	}
	if urbitPayload.Payload.Patp != action.Patp {
		return
	}
	if err := UrbitHandler(action.Content); err != nil {
		logger.Logger.Error(fmt.Sprintf("%+v", err))
		return
	}
}

func gallsegAuthedHandler(action leakchannel.ActionChannel) {
	switch action.Type {
	case "urbit":
		if err := UrbitHandler(action.Content); err != nil {
			logger.Logger.Error(fmt.Sprintf("%+v", err))
			return
		}
	case "penpai":
		if err := PenpaiHandler(action.Content); err != nil {
			logger.Logger.Error(fmt.Sprintf("%v", err))
			return
		}
	case "new_ship":
		if err := NewShipHandler(action.Content); err != nil {
			logger.Logger.Error(fmt.Sprintf("%v", err))
			return
		}
	case "system":
		if err := SystemHandler(action.Content); err != nil {
			logger.Logger.Error(fmt.Sprintf("%v", err))
			return
		}
	case "startram":
		if err := StartramHandler(action.Content); err != nil {
			logger.Logger.Error(fmt.Sprintf("%v", err))
			return
		}
	case "support":
		if err := SupportHandler(action.Content); err != nil {
			logger.Logger.Error(fmt.Sprintf("Error creating bug report: %v", err))
			return
		}
		/*
			case "password":
				if err := PwHandler(action.Content); err != nil {
					logger.Logger.Error(fmt.Sprintf("%v", err))
					return
				} else {
					resp, err := handler.UnauthHandler()
					if err != nil {
						logger.Logger.Warn(fmt.Sprintf("Unable to generate deauth payload: %v", err))
					}
					MuCon.Write(resp)
				}
		*/
	}
}
