package contractspec_test

import (
	"testing"

	"groundseg/protocol/contracts"
	"groundseg/uploadsvc"
	"groundseg/uploadsvc/contractspec"
)

func TestSupportedUploadActionsAndContractsStayAligned(t *testing.T) {
	availableActions, err := uploadsvc.SupportedUploadActions()
	if err != nil {
		t.Fatalf("supported upload actions: %v", err)
	}
	expected := contracts.UploadActionBindingSpecs()
	if len(availableActions) != len(expected) {
		t.Fatalf("supported upload action count mismatch: got %d want %d", len(availableActions), len(expected))
	}
	specs := contractspec.UploadActionSpecs()
	if len(specs) != len(availableActions) {
		t.Fatalf("upload contract specs count mismatch: got %d want %d", len(specs), len(availableActions))
	}

	contractByAction, err := uploadsvc.UploadActionContractByAction()
	if err != nil {
		t.Fatalf("upload action contracts: %v", err)
	}
	for _, action := range availableActions {
		if _, ok := contractByAction[action]; !ok {
			t.Fatalf("missing contract metadata for supported action %q", action)
		}
	}
	for _, expectedSpec := range expected {
		action := uploadsvc.Action(expectedSpec.Action)
		if _, ok := contractByAction[action]; !ok {
			t.Fatalf("missing contract metadata for expected action %q", action)
		}
	}

}
