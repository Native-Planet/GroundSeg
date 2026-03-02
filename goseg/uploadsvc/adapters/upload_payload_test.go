package adapters

import (
	"errors"
	"testing"

	"groundseg/protocol/actions"
	"groundseg/structs"
	"groundseg/uploadsvc"
)

func sampleToken() structs.WsTokenStruct {
	return structs.WsTokenStruct{
		ID:    "tok-id",
		Token: "tok-val",
	}
}

func payloadForContract(contract actions.UploadActionContract) structs.WsUploadAction {
	payload := structs.WsUploadAction{
		Action: string(contract.Action),
	}
	if contract.RequiredPayloads.Has(actions.UploadPayloadOpenEndpoint) {
		payload.Endpoint = "session-1"
		payload.Remote = true
		payload.Fix = true
		payload.SelectedDrive = "/dev/sda"
	}
	return payload
}

func forbiddenPayload(contract actions.UploadActionContract) *structs.WsUploadAction {
	if !contract.ForbiddenPayloads.Has(actions.UploadPayloadOpenEndpoint) {
		return nil
	}
	payload := structs.WsUploadAction{Action: string(contract.Action), Endpoint: "session-x", Remote: true, Fix: true, SelectedDrive: "/dev/extra"}
	return &payload
}

func TestCommandFromWsPayloadMapsProtocolContracts(t *testing.T) {
	for _, contract := range actions.UploadActionContracts() {
		t.Run(string(contract.Action), func(t *testing.T) {
			payload := structs.WsUploadPayload{
				Token:   sampleToken(),
				Payload: payloadForContract(contract),
			}
			cmd, err := CommandFromWsPayload(payload)
			if err != nil {
				t.Fatalf("expected action %q to map, got error %v", contract.Action, err)
			}
			if cmd.Action != contract.Action {
				t.Fatalf("expected action %q, got %q", contract.Action, cmd.Action)
			}
			if contract.RequiredPayloads.Has(actions.UploadPayloadOpenEndpoint) && cmd.OpenEndpointRequest == nil {
				t.Fatalf("expected open-endpoint request for action %q", contract.Action)
			}
			if contract.RequiredPayloads.Has(actions.UploadPayloadReset) && cmd.ResetRequest == nil {
				t.Fatalf("expected reset request for action %q", contract.Action)
			}
			if contract.RequiredPayloads.Has(actions.UploadPayloadOpenEndpoint) {
				if cmd.OpenEndpointRequest.Endpoint != payloadForContract(contract).Endpoint {
					t.Fatalf("unexpected endpoint for action %q: got=%q want=%q", contract.Action, cmd.OpenEndpointRequest.Endpoint, payloadForContract(contract).Endpoint)
				}
				if cmd.OpenEndpointRequest.TokenID != sampleToken().ID || cmd.OpenEndpointRequest.TokenValue != sampleToken().Token {
					t.Fatalf("expected mapped token values for action %q", contract.Action)
				}
			}
		})
	}
}

func TestCommandFromWsPayloadRejectsForbiddenPayloads(t *testing.T) {
	for _, contract := range actions.UploadActionContracts() {
		forbidden := forbiddenPayload(contract)
		if forbidden == nil {
			continue
		}
		payload := structs.WsUploadPayload{
			Token:   sampleToken(),
			Payload: *forbidden,
		}
		_, err := CommandFromWsPayload(payload)
		if err == nil {
			t.Fatalf("expected forbidden payload for action %q to error", contract.Action)
		}
		var validation uploadsvc.CommandValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("expected CommandValidationError for %q, got %T: %v", contract.Action, err, err)
		}
		if validation.Action != contract.Action {
			t.Fatalf("expected validation error for %q, got %q", contract.Action, validation.Action)
		}
	}
}

func TestCommandFromWsPayloadRejectsOtherNamespaceActions(t *testing.T) {
	for _, action := range actions.ActionMetas(actions.NamespaceC2C) {
		payload := structs.WsUploadPayload{
			Payload: structs.WsUploadAction{
				Action: string(action.Action),
			},
		}
		if _, err := CommandFromWsPayload(payload); err == nil {
			t.Fatalf("expected namespace mismatch for action %q", action.Action)
		}
	}
}

func TestCommandFromWsPayloadUsesParsedActionOutput(t *testing.T) {
	contract := actions.UploadActionContracts()[0]
	payload := structs.WsUploadPayload{
		Token:   sampleToken(),
		Payload: payloadForContract(contract),
	}

	parsed, err := actions.ParseUploadAction(payload.Payload.Action)
	if err != nil {
		t.Fatalf("ParseAction rejected supported action: %v", err)
	}

	cmd, err := CommandFromWsPayload(payload)
	if err != nil {
		t.Fatalf("CommandFromWsPayload returned error: %v", err)
	}
	if cmd.Action != parsed {
		t.Fatalf("command action did not use parser output: got=%q want=%q", cmd.Action, parsed)
	}
}

func TestCommandFromWsPayloadRejectsUnsupportedAction(t *testing.T) {
	payload := structs.WsUploadPayload{
		Payload: structs.WsUploadAction{
			Action: "invalid",
		},
	}
	if _, err := CommandFromWsPayload(payload); err == nil {
		t.Fatal("expected unsupported action to error")
	}
}
