package adapters

import (
	"groundseg/protocol/actions"
	"groundseg/structs"
	"groundseg/uploadsvc"
)

func CommandFromWsPayload(payload structs.WsUploadPayload) (uploadsvc.Command, error) {
	action, err := actions.ParseUploadAction(payload.Payload.Action)
	if err != nil {
		return uploadsvc.Command{}, err
	}

	openEndpointRequest := OpenEndpointRequestFromWsPayload(payload)
	var resetRequest *uploadsvc.ResetRequest

	switch action {
	case actions.ActionUploadReset:
		resetRequest = &uploadsvc.ResetRequest{}
	}
	return uploadsvc.CommandFromUploadInputs(action, openEndpointRequest, resetRequest)
}

func OpenEndpointRequestFromWsPayload(payload structs.WsUploadPayload) uploadsvc.OpenEndpointRequest {
	return uploadsvc.OpenEndpointRequest{
		Endpoint:      payload.Payload.Endpoint,
		TokenID:       payload.Token.ID,
		TokenValue:    payload.Token.Token,
		Remote:        payload.Payload.Remote,
		Fix:           payload.Payload.Fix,
		SelectedDrive: payload.Payload.SelectedDrive,
	}
}
