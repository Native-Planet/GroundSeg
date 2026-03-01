package adapters

import (
	"groundseg/structs"
	"groundseg/uploadsvc"
)

func CommandFromWsPayload(payload structs.WsUploadPayload) (uploadsvc.Command, error) {
	action, err := uploadsvc.ParseAction(payload.Payload.Action)
	if err != nil {
		return uploadsvc.Command{}, err
	}
	return uploadsvc.Command{
		Action:              action,
		OpenEndpointRequest: OpenEndpointRequestFromWsPayload(payload),
	}, nil
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
