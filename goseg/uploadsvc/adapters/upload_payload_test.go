package adapters

import (
	"errors"
	"fmt"
	"sort"
	"testing"

	"groundseg/protocol/actions"
	"groundseg/structs"
	"groundseg/uploadsvc"
)

func sortedUploadActions(contracts map[actions.Action]uploadsvc.UploadActionContract) []actions.Action {
	actionsSlice := make([]actions.Action, 0, len(contracts))
	for action := range contracts {
		actionsSlice = append(actionsSlice, action)
	}
	sort.Slice(actionsSlice, func(i, j int) bool {
		return string(actionsSlice[i]) < string(actionsSlice[j])
	})
	return actionsSlice
}

func mustUploadContracts(t *testing.T) map[actions.Action]uploadsvc.UploadActionContract {
	t.Helper()
	contractsByAction, err := uploadsvc.UploadActionContractByAction()
	if err != nil {
		t.Fatalf("upload action contracts: %v", err)
	}
	return contractsByAction
}

func sampleToken() structs.WsTokenStruct {
	return structs.WsTokenStruct{
		ID:    "tok-id",
		Token: "tok-val",
	}
}

func payloadForContract(contract uploadsvc.UploadActionContract) structs.WsUploadAction {
	payload := structs.WsUploadAction{
		Action: string(contract.Action),
	}
	if contract.RequiredPayloads.Has(uploadsvc.UploadPayloadOpenEndpoint) {
		payload.Endpoint = "session-1"
		payload.Remote = true
		payload.Fix = true
		payload.SelectedDrive = "/dev/sda"
	}
	return payload
}

func forbiddenPayload(contract uploadsvc.UploadActionContract) *structs.WsUploadAction {
	if !contract.ForbiddenPayloads.Has(uploadsvc.UploadPayloadOpenEndpoint) {
		return nil
	}
	payload := structs.WsUploadAction{Action: string(contract.Action), Endpoint: "session-x", Remote: true, Fix: true, SelectedDrive: "/dev/extra"}
	return &payload
}

func TestCommandFromWsPayloadMapsProtocolContracts(t *testing.T) {
	contracts := mustUploadContracts(t)
	for _, action := range sortedUploadActions(contracts) {
		contract := contracts[action]
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
			if contract.RequiredPayloads.Has(uploadsvc.UploadPayloadOpenEndpoint) && cmd.OpenEndpointRequest == nil {
				t.Fatalf("expected open-endpoint request for action %q", contract.Action)
			}
			if contract.RequiredPayloads.Has(uploadsvc.UploadPayloadReset) && cmd.ResetRequest == nil {
				t.Fatalf("expected reset request for action %q", contract.Action)
			}
			if contract.RequiredPayloads.Has(uploadsvc.UploadPayloadOpenEndpoint) {
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
	contracts := mustUploadContracts(t)
	for _, action := range sortedUploadActions(contracts) {
		contract := contracts[action]
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
	payload := structs.WsUploadPayload{
		Payload: structs.WsUploadAction{Action: string(actions.ActionC2CConnect)},
	}
	if _, err := CommandFromWsPayload(payload); err == nil {
		t.Fatalf("expected namespace mismatch for action %q", actions.ActionC2CConnect)
	}
}

func TestCommandFromWsPayloadUsesParsedActionOutput(t *testing.T) {
	contractMap := mustUploadContracts(t)
	contract, ok := contractMap[uploadsvc.ActionUploadOpenEndpoint]
	if !ok {
		t.Fatal("expected upload open-endpoint contract")
	}
	payload := structs.WsUploadPayload{
		Token:   sampleToken(),
		Payload: payloadForContract(contract),
	}

	parsed, err := uploadsvc.ParseUploadAction(payload.Payload.Action)
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
	invalidAction := fmt.Sprintf("%s-unsupported", uploadsvc.ActionUploadReset)
	payload := structs.WsUploadPayload{
		Payload: structs.WsUploadAction{
			Action: invalidAction,
		},
	}
	_, err := CommandFromWsPayload(payload)
	if err == nil {
		t.Fatal("expected unsupported action to error")
	}
	var unsupported actions.UnsupportedActionError
	if !errors.As(err, &unsupported) {
		t.Fatalf("expected protocol unsupported action error for %q, got %T: %v", invalidAction, err, err)
	}
}

func TestCommandFromWsPayloadSupportsAllProtocolUploadActions(t *testing.T) {
	contracts := mustUploadContracts(t)
	actionsSlice := sortedUploadActions(contracts)
	if len(actionsSlice) == 0 {
		t.Fatal("expected upload action contracts")
	}
	for _, action := range actionsSlice {
		parsed, err := uploadsvc.ParseUploadAction(string(action))
		if err != nil {
			t.Fatalf("expected protocol upload action %q to parse, got %v", action, err)
		}
		contract, exists := contracts[parsed]
		if !exists {
			t.Fatalf("expected upload action contract for %q", parsed)
		}

		payload := structs.WsUploadPayload{
			Token:   sampleToken(),
			Payload: payloadForContract(contract),
		}
		cmd, err := CommandFromWsPayload(payload)
		if err != nil {
			t.Fatalf("expected protocol action %q to map, got error %v", parsed, err)
		}
		if cmd.Action != parsed {
			t.Fatalf("expected parser action %q to match command action %q", parsed, cmd.Action)
		}
	}
}
