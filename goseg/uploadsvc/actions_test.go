package uploadsvc

import (
	"testing"

	"groundseg/protocol/contracts"
)

func TestParseUploadActionAcceptsKnownActions(t *testing.T) {
	t.Parallel()

	gotOpen, err := ParseUploadAction(string(ActionUploadOpenEndpoint))
	if err != nil {
		t.Fatalf("ParseUploadAction(open-endpoint) returned error: %v", err)
	}
	if gotOpen != ActionUploadOpenEndpoint {
		t.Fatalf("unexpected parsed open-endpoint action: %q", gotOpen)
	}

	gotReset, err := ParseUploadAction(string(ActionUploadReset))
	if err != nil {
		t.Fatalf("ParseUploadAction(reset) returned error: %v", err)
	}
	if gotReset != ActionUploadReset {
		t.Fatalf("unexpected parsed reset action: %q", gotReset)
	}
}

func TestParseUploadActionRejectsUnknownAction(t *testing.T) {
	t.Parallel()

	if _, err := ParseUploadAction("not-an-upload-action"); err == nil {
		t.Fatal("expected unknown upload action to fail parsing")
	}
}

func TestUploadParserActionsMatchUploadContractRegistry(t *testing.T) {
	t.Parallel()

	expectedByAction := make(map[Action]contracts.UploadActionBindingSpec)
	for _, spec := range contracts.UploadActionBindingSpecs() {
		expectedByAction[Action(spec.Action)] = spec
	}
	supported, err := SupportedUploadActions()
	if err != nil {
		t.Fatalf("supported upload actions: %v", err)
	}
	if len(supported) != len(expectedByAction) {
		t.Fatalf("supported action count mismatch: got %d want %d", len(supported), len(expectedByAction))
	}
	contractsByAction, err := UploadActionContractByAction()
	if err != nil {
		t.Fatalf("upload action contracts: %v", err)
	}
	for _, action := range supported {
		parsed, err := ParseUploadAction(string(action))
		if err != nil {
			t.Fatalf("parse supported upload action %q: %v", action, err)
		}
		uploadContract, ok := contractsByAction[parsed]
		if !ok {
			t.Fatalf("missing upload action contract for %q", parsed)
		}
		expected, ok := expectedByAction[parsed]
		if !ok {
			t.Fatalf("missing upload contract registry entry for action %q", parsed)
		}
		if uploadContract.ID != expected.Contract {
			t.Fatalf("contract id mismatch for action %q: got %s want %s", parsed, uploadContract.ID, expected.Contract)
		}
		if uploadContract.Contract.Name != expected.Name || uploadContract.Contract.Description != expected.Description {
			t.Fatalf("descriptor metadata mismatch for action %q", parsed)
		}
		if uploadContract.RequiredPayloads != expected.RequiredPayloads || uploadContract.ForbiddenPayloads != expected.ForbiddenPayloads {
			t.Fatalf("payload rule mismatch for action %q", parsed)
		}
	}
}
