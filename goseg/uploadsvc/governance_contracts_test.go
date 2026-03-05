package uploadsvc

import (
	"errors"
	"sync"
	"testing"

	"groundseg/protocol/actions"
	"groundseg/protocol/contracts/governance"
)

func resetUploadGovernanceCacheForTest(t *testing.T) {
	t.Helper()
	uploadGovernanceContractsByAction = nil
	uploadGovernanceContractsByActionErr = nil
	uploadGovernanceContractsByActionInit = sync.Once{}
}

func TestBuildUploadGovernanceContractsByActionMapsPolicies(t *testing.T) {
	contracts, err := buildUploadGovernanceContractsByAction()
	if err != nil {
		t.Fatalf("expected governance contract map build to succeed: %v", err)
	}

	openContract, ok := contracts[ActionUploadOpenEndpoint]
	if !ok {
		t.Fatal("expected open-endpoint action contract in map")
	}
	if openContract.ContractID != governance.UploadOpenEndpointContractID {
		t.Fatalf("unexpected open-endpoint contract id: %s", openContract.ContractID)
	}
	if !openContract.RequiredPayloads.Has(actions.UploadPayloadOpenEndpoint) {
		t.Fatal("expected open-endpoint contract to require open-endpoint payload")
	}
	if !openContract.ForbiddenPayloads.Has(actions.UploadPayloadReset) {
		t.Fatal("expected open-endpoint contract to forbid reset payload")
	}

	resetContract, ok := contracts[ActionUploadReset]
	if !ok {
		t.Fatal("expected reset action contract in map")
	}
	if resetContract.ContractID != governance.UploadResetContractID {
		t.Fatalf("unexpected reset contract id: %s", resetContract.ContractID)
	}
	if !resetContract.RequiredPayloads.Has(actions.UploadPayloadReset) {
		t.Fatal("expected reset contract to require reset payload")
	}
	if !resetContract.ForbiddenPayloads.Has(actions.UploadPayloadOpenEndpoint) {
		t.Fatal("expected reset contract to forbid open-endpoint payload")
	}
}

func TestUploadGovernanceContractForActionSupportsKnownAndUnknownActions(t *testing.T) {
	resetUploadGovernanceCacheForTest(t)
	t.Cleanup(func() {
		resetUploadGovernanceCacheForTest(t)
	})

	openContract, err := uploadGovernanceContractForAction(ActionUploadOpenEndpoint)
	if err != nil {
		t.Fatalf("expected known action contract lookup to succeed: %v", err)
	}
	if openContract.Action != ActionUploadOpenEndpoint {
		t.Fatalf("unexpected action contract binding: %s", openContract.Action)
	}

	_, err = uploadGovernanceContractForAction(Action("unsupported-action"))
	if err == nil {
		t.Fatal("expected unsupported action lookup to fail")
	}
	var unsupported actions.UnsupportedActionError
	if !errors.As(err, &unsupported) {
		t.Fatalf("expected unsupported action error type, got %T: %v", err, err)
	}
}
