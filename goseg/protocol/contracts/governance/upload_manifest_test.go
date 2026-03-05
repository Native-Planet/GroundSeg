package governance

import (
	"testing"

	"groundseg/protocol/contracts/catalog/common"
)

func TestUploadActionDeclarationsExposeOpenAndResetContracts(t *testing.T) {
	declarations := uploadActionDeclarations()
	if len(declarations) != 2 {
		t.Fatalf("expected two upload action declarations, got %d", len(declarations))
	}

	openDecl := declarations[0]
	if openDecl.Action != ActionUploadOpenEndpoint {
		t.Fatalf("unexpected open endpoint action token: %s", openDecl.Action)
	}
	if openDecl.ContractID != UploadOpenEndpointContractID {
		t.Fatalf("unexpected open endpoint contract id: %s", openDecl.ContractID)
	}
	if openDecl.Owner != string(common.OwnerUploadService) {
		t.Fatalf("unexpected open endpoint owner: %s", openDecl.Owner)
	}
	if !openDecl.RequiredPayloads.Has(UploadPayloadRuleOpenEndpoint) {
		t.Fatal("expected open endpoint declaration to require open endpoint payload")
	}
	if !openDecl.ForbiddenPayloads.Has(UploadPayloadRuleReset) {
		t.Fatal("expected open endpoint declaration to forbid reset payload")
	}

	resetDecl := declarations[1]
	if resetDecl.Action != ActionUploadReset {
		t.Fatalf("unexpected reset action token: %s", resetDecl.Action)
	}
	if resetDecl.ContractID != UploadResetContractID {
		t.Fatalf("unexpected reset contract id: %s", resetDecl.ContractID)
	}
	if !resetDecl.RequiredPayloads.Has(UploadPayloadRuleReset) {
		t.Fatal("expected reset declaration to require reset payload")
	}
	if !resetDecl.ForbiddenPayloads.Has(UploadPayloadRuleOpenEndpoint) {
		t.Fatal("expected reset declaration to forbid open endpoint payload")
	}
}
