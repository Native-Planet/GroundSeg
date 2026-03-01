package system

import (
	"encoding/json"
	"fmt"
	"groundseg/shipworkflow"
	"groundseg/structs"
	"groundseg/transition"

	"go.uber.org/zap"
)

var (
	startramServicesActionHandler = shipworkflow.HandleStartramServices
	startramRegionsActionHandler  = shipworkflow.HandleStartramRegions
	startramRegisterActionHandler = shipworkflow.HandleStartramRegister
	startramToggleActionHandler   = shipworkflow.HandleStartramToggle
	startramRestartActionHandler  = shipworkflow.HandleStartramRestart
	startramCancelActionHandler   = shipworkflow.HandleStartramCancel
	startramEndpointActionHandler = shipworkflow.HandleStartramEndpoint
	startramReminderActionHandler = shipworkflow.HandleStartramReminder
	startramSetBackupPWHandler    = shipworkflow.HandleStartramSetBackupPassword
)

func runStartramAsync(action string, fn func() error) {
	go func() {
		if err := fn(); err != nil {
			zap.L().Error(fmt.Sprintf("startram action %s failed: %v", action, err))
		}
	}()
}

func StartramHandler(msg []byte) error {
	var startramPayload structs.WsStartramPayload
	err := json.Unmarshal(msg, &startramPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal startram payload: %v", err)
	}
	action, ok := transition.ParseStartramHandlerAction(startramPayload.Payload.Action)
	if !ok {
		return fmt.Errorf("Unrecognized startram action: %v", startramPayload.Payload.Action)
	}
	switch action {
	case transition.StartramHandlerActionServices:
		runStartramAsync(string(transition.StartramHandlerActionServices), startramServicesActionHandler)
	case transition.StartramHandlerActionRegions:
		runStartramAsync(string(transition.StartramHandlerActionRegions), startramRegionsActionHandler)
	case transition.StartramHandlerActionRegister:
		regCode := startramPayload.Payload.Key
		region := startramPayload.Payload.Region
		runStartramAsync(string(transition.StartramHandlerActionRegister), func() error {
			return startramRegisterActionHandler(regCode, region)
		})
	case transition.StartramHandlerActionToggle:
		runStartramAsync(string(transition.StartramHandlerActionToggle), startramToggleActionHandler)
	case transition.StartramHandlerActionRestart:
		runStartramAsync(string(transition.StartramHandlerActionRestart), startramRestartActionHandler)
	case transition.StartramHandlerActionCancel:
		key := startramPayload.Payload.Key
		reset := startramPayload.Payload.Reset
		runStartramAsync(string(transition.StartramHandlerActionCancel), func() error {
			return startramCancelActionHandler(key, reset)
		})
	case transition.StartramHandlerActionEndpoint:
		endpoint := startramPayload.Payload.Endpoint
		runStartramAsync(string(transition.StartramHandlerActionEndpoint), func() error {
			return startramEndpointActionHandler(endpoint)
		})
	case transition.StartramHandlerActionReminder:
		runStartramAsync(string(transition.StartramHandlerActionReminder), func() error {
			return startramReminderActionHandler(startramPayload.Payload.Remind)
		})
		/*
			case "restore-backup":
				go handleStartramRestoreBackup(startramPayload.Payload.Target, startramPayload.Payload.Patp, startramPayload.Payload.Backup, startramPayload.Payload.Key)
			case "upload-backup":
				go handleStartramUploadBackup(startramPayload.Payload.Patp)
		*/
	case transition.StartramHandlerActionSetBackupPW:
		runStartramAsync(string(transition.StartramHandlerActionSetBackupPW), func() error {
			return startramSetBackupPWHandler(startramPayload.Payload.Password)
		})
	}
	return nil
}

func appendOrchestrationError(stepErrors *[]error, context string, err error) {
	if err == nil {
		return
	}
	*stepErrors = append(*stepErrors, fmt.Errorf("%s: %w", context, err))
	zap.L().Error(fmt.Sprintf("%s: %v", context, err))
}
