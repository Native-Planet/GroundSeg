package conformance

import (
	"testing"

	"groundseg/protocol/contracts"
)

func TestActionContractFixturesReturnSnapshots(t *testing.T) {
	first := ActionContractFixtures()
	if len(first) == 0 {
		t.Fatal("expected action fixtures")
	}

	first[0].Contract = contracts.ContractID("mutated")
	second := ActionContractFixtures()
	if len(second) == 0 {
		t.Fatal("expected action fixtures on second read")
	}
	if second[0].Contract == contracts.ContractID("mutated") {
		t.Fatal("expected ActionContractFixtures to return a fresh snapshot")
	}
}

func TestValidateFixturesForNamespaceRejectsMissingNamespace(t *testing.T) {
	if _, err := ValidateFixturesForNamespace(contracts.ActionNamespace("missing")); err == nil {
		t.Fatal("expected missing namespace validation to fail")
	}
}
