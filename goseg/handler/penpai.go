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
	case "toggle":
		// start container
		// stop container
		logger.Logger.Debug(fmt.Sprintf("Todo: toggle penpai"))
	case "set-model":
		model := penpaiPayload.Payload.Model
		// update config
		// if running, restart container
		logger.Logger.Debug(fmt.Sprintf("Todo: set penpai model to %v", model))
	case "set-cores":
		cores := penpaiPayload.Payload.Cores
		// check if core count is valid
		// update config
		// if running, restart container
		logger.Logger.Debug(fmt.Sprintf("Todo: set penpai cores to %v", cores))
	case "remove":
		// check if container exists
		// remove container, delete volume
		logger.Logger.Debug(fmt.Sprintf("Todo: remove penpai"))
	}
	return nil
}
