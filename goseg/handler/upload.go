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

func TransloadHandler(msg []byte) error {
	logger.Logger.Info("Transload")
	var transloadPayload structs.WsTransloadPayload
	var customDrive string
	err := json.Unmarshal(msg, &transloadPayload)
	if err != nil {
		return fmt.Errorf("couldn't unmarshal transload payload: %v", err)
	}
	/*
		if err = checkPath(transloadPayload.Payload.Path); err != nil {
			return fmt.Errorf("invalid pier path: %v", err)
		}
	*/
	//filename := filepath.Base(transloadPayload.Payload.Path)
	if transloadPayload.Payload.SelectedDrive != "system-drive" {
		customDrive = transloadPayload.Payload.SelectedDrive
	}
	logger.Logger.Warn(fmt.Sprintf("%+v", transloadPayload))
	_ = customDrive
	//go importer.TransloadPier(filename, transloadPayload.Payload.Patp, transloadPayload.Payload.Remote, transloadPayload.Payload.Fix, transloadPayload.Payload.Path, customDrive)
	return nil
}

/*
func checkPath(filePath string) error {
	validPathRegex := regexp.MustCompile(`^/(?:[^/]+/)*[^/]+(?:\.tar|\.tar\.gz|\.tgz|\.zip)$`)
	if !validPathRegex.MatchString(filePath) {
		return fmt.Errorf("invalid file path or extension")
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist")
	} else if err != nil {
		return fmt.Errorf("error checking file: %v", err)
	}
	return nil
}
*/
