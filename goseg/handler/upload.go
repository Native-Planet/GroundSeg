package handler

import (
	"encoding/json"
	"fmt"
	"groundseg/importer"
	"groundseg/logger"
	"groundseg/structs"
)

func UploadHandler(msg []byte) error {
	logger.Logger.Info("Upload")
	var uploadPayload structs.WsUploadPayload
	err := json.Unmarshal(msg, &uploadPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal upload payload: %v", err)
	}
	switch uploadPayload.Payload.Action {
	case "open-endpoint":
		if err := importer.SetUploadSession(uploadPayload); err != nil {
			return fmt.Errorf("Failed to open upload endpoint: %v", uploadPayload.Payload.Endpoint)
		}
	case "reset":
		if err := importer.Reset(); err != nil {
			return fmt.Errorf("Failed to reset importer: %v", err)
		}
	default:
		return fmt.Errorf("Unrecognized upload action: %v", uploadPayload.Payload.Action)
	}
	return nil
}
