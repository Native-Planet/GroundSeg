package adapters

import (
	"testing"

	"groundseg/structs"
	"groundseg/uploadsvc"
)

func TestCommandFromWsPayloadMapsActionAndRequest(t *testing.T) {
	payload := structs.WsUploadPayload{
		Token: structs.WsTokenStruct{
			ID:    "tok-id",
			Token: "tok-val",
		},
		Payload: structs.WsUploadAction{
			Action:        string(uploadsvc.ActionOpenEndpoint),
			Endpoint:      "session-1",
			Remote:        true,
			Fix:           true,
			SelectedDrive: "/dev/sda",
		},
	}

	cmd := CommandFromWsPayload(payload)

	if cmd.Action != uploadsvc.ActionOpenEndpoint {
		t.Fatalf("unexpected action: %v", cmd.Action)
	}
	req := cmd.OpenEndpointRequest
	if req.Endpoint != "session-1" ||
		req.TokenID != "tok-id" ||
		req.TokenValue != "tok-val" ||
		!req.Remote ||
		!req.Fix ||
		req.SelectedDrive != "/dev/sda" {
		t.Fatalf("unexpected open endpoint request mapping: %+v", req)
	}
}
