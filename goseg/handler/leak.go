package handler

import (
	"fmt"
	"goseg/leakchannel"
	"goseg/logger"
)

func HandleLeakAction() {
	// no:
	// pier upload
	// password/login stuff
	// logs
	for {
		action := <-leakchannel.LeakAction
		switch action.Type {
		case "urbit":
			if err := UrbitHandler(action.Content); err != nil {
				logger.Logger.Error(fmt.Sprintf("%+v", err))
				continue
			}
		case "penpai":
			if err := PenpaiHandler(action.Content); err != nil {
				logger.Logger.Error(fmt.Sprintf("%v", err))
				continue
			}
		case "new_ship":
			if err := NewShipHandler(action.Content); err != nil {
				logger.Logger.Error(fmt.Sprintf("%v", err))
				continue
			}
			/*
				case "pier_upload":
					if err := UploadHandler(action.Content); err != nil {
						logger.Logger.Error(fmt.Sprintf("%v", err))
						continue
					}
				case "password":
					if err := PwHandler(action.Content); err != nil {
						logger.Logger.Error(fmt.Sprintf("%v", err))
						continue
					} else {
						resp, err := handler.UnauthHandler()
						if err != nil {
							logger.Logger.Warn(fmt.Sprintf("Unable to generate deauth payload: %v", err))
						}
						MuCon.Write(resp)
					}
			*/
		case "system":
			if err := SystemHandler(action.Content); err != nil {
				logger.Logger.Error(fmt.Sprintf("%v", err))
				continue
			}
		case "startram":
			if err := StartramHandler(action.Content); err != nil {
				logger.Logger.Error(fmt.Sprintf("%v", err))
				continue
			}
		case "support":
			if err := SupportHandler(action.Content); err != nil {
				logger.Logger.Error(fmt.Sprintf("Error creating bug report: %v", err))
				continue
			}
		}
	}
}
