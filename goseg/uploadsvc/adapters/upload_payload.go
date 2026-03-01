package adapters

import (
	"groundseg/structs"
	"groundseg/uploadsvc"
)

func CommandFromWsPayload(payload structs.WsUploadPayload) uploadsvc.Command {
	return uploadsvc.Command{
		Action:              uploadsvc.Action(payload.Payload.Action),
		OpenEndpointRequest: OpenEndpointRequestFromWsPayload(payload),
	}
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
