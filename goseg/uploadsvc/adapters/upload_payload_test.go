package adapters

import (
	"errors"
	"testing"

	"groundseg/structs"
	"groundseg/uploadsvc"
)

func TestCommandFromWsPayloadMapsActionAndRequest(t *testing.T) {
	cases := []struct {
		name            string
		action          uploadsvc.Action
		payload         structs.WsUploadAction
		expectReq       uploadsvc.OpenEndpointRequest
		expectErrorType error
	}{
		{
			name:   "open-endpoint",
			action: uploadsvc.ActionOpenEndpoint,
			payload: structs.WsUploadAction{
				Action:        string(uploadsvc.ActionOpenEndpoint),
				Endpoint:      "session-1",
				Remote:        true,
				Fix:           true,
				SelectedDrive: "/dev/sda",
			},
			expectReq: uploadsvc.OpenEndpointRequest{
				Endpoint:      "session-1",
				TokenID:       "tok-id",
				TokenValue:    "tok-val",
				Remote:        true,
				Fix:           true,
				SelectedDrive: "/dev/sda",
			},
		},
		{
			name:   "reset",
			action: uploadsvc.ActionReset,
			payload: structs.WsUploadAction{
				Action: string(uploadsvc.ActionReset),
			},
		},
		{
			name:   "unsupported",
			action: uploadsvc.Action("invalid"),
			payload: structs.WsUploadAction{
				Action: "invalid",
			},
			expectErrorType: uploadsvc.UnsupportedActionError{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := structs.WsUploadPayload{
				Token: structs.WsTokenStruct{
					ID:    "tok-id",
					Token: "tok-val",
				},
				Payload: tc.payload,
			}

			cmd, err := CommandFromWsPayload(payload)
			if tc.expectErrorType != nil {
				if err == nil {
					t.Fatalf("expected unsupported action to return error")
				}
				var expected uploadsvc.UnsupportedActionError
				if !errors.As(err, &expected) {
					t.Fatalf("expected UnsupportedActionError, got %T: %v", err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("CommandFromWsPayload returned error: %v", err)
			}
			if cmd.Action != tc.action {
				t.Fatalf("unexpected action: %v", cmd.Action)
			}
			if tc.action == uploadsvc.ActionOpenEndpoint {
				req := cmd.OpenEndpointRequest
				if req != tc.expectReq {
					t.Fatalf("unexpected open endpoint request mapping: got=%+v want=%+v", req, tc.expectReq)
				}
			}
		})
	}
}
