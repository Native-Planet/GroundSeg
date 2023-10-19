package handler

import (
	"encoding/json"
	"fmt"
	"goseg/logger"
	"goseg/structs"
)

func PenpaiHandler(msg []byte) error {
	logger.Logger.Info("Penpai")
	var penpaiPayload structs.WsPenpaiPayload
	err := json.Unmarshal(msg, &penpaiPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal penpai payload: %v", err)
	}
	switch penpaiPayload.Payload.Action {
	case "set-model":
		model := penpaiPayload.Payload.Model
		logger.Logger.Debug(fmt.Sprintf("Todo: set penpai mdel to %v", model))
		// start container
		// update config
	}
	return nil
}
