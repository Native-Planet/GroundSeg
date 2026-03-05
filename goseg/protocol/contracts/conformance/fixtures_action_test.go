package conformance

import (
	"testing"

	"groundseg/protocol/contracts"
	"groundseg/protocol/contracts/familycatalog"
	"groundseg/protocol/contracts/governance"
)

func TestActionContractFixturesCoverCanonicalBindings(t *testing.T) {
	fixtures := ActionContractFixtures()
	if len(fixtures) == 0 {
		t.Fatal("expected at least one action contract fixture")
	}

	foundC2C := false
	for _, fixture := range fixtures {
		if err := ValidateActionContractFixture(fixture); err != nil {
			t.Fatalf("fixture validation failed for %s:%s: %v", fixture.Namespace, fixture.Action, err)
		}
		if fixture.Namespace == contracts.ActionNamespace(familycatalog.NamespaceC2C) &&
			fixture.Action == contracts.ActionVerb(familycatalog.ActionC2CConnect) {
			foundC2C = true
		}
	}
	if !foundC2C {
		t.Fatal("expected C2C connect fixture to be present")
	}
}

func TestActionContractFixturesMatchCanonicalBindings(t *testing.T) {
	fixtures := ActionContractFixtures()
	bindings := contracts.ActionContractBindings()
	if len(fixtures) != len(bindings) {
		t.Fatalf("fixture count mismatch: got %d want %d", len(fixtures), len(bindings))
	}

	type key struct {
		namespace contracts.ActionNamespace
		action    contracts.ActionVerb
	}
	fixtureContracts := make(map[key]contracts.ContractID, len(fixtures))
	for _, fixture := range fixtures {
		fixtureContracts[key{namespace: fixture.Namespace, action: fixture.Action}] = fixture.Contract
	}

	for _, binding := range bindings {
		k := key{
			namespace: binding.Namespace,
			action:    binding.Action,
		}
		contractID, ok := fixtureContracts[k]
		if !ok {
			t.Fatalf("fixture missing canonical binding %s:%s", binding.Namespace, binding.Action)
		}
		if contractID != binding.Contract {
			t.Fatalf("fixture contract mismatch for %s:%s: got %s want %s", binding.Namespace, binding.Action, contractID, binding.Contract)
		}
		delete(fixtureContracts, k)
	}

	for unexpected := range fixtureContracts {
		t.Fatalf("fixture has unexpected binding %s:%s", unexpected.namespace, unexpected.action)
	}
}

func TestValidateActionContractFixtureRejectsUnknownAction(t *testing.T) {
	err := ValidateActionContractFixture(ActionContractFixture{
		Namespace: contracts.ActionNamespace(familycatalog.NamespaceC2C),
		Action:    contracts.ActionVerb("does-not-exist"),
		Contract:  "protocol.actions.c2c.does-not-exist",
	})
	if err == nil {
		t.Fatal("expected validation failure for unknown action fixture")
	}
}

func TestValidateFixturesForNamespace(t *testing.T) {
	count, err := ValidateFixturesForNamespace(contracts.ActionNamespace(familycatalog.NamespaceUpload))
	if err != nil {
		t.Fatalf("validate upload fixtures: %v", err)
	}
	if count == 0 {
		t.Fatal("expected upload fixtures")
	}
}

func TestStartramGovernanceDeclarationsMapToCatalogEntries(t *testing.T) {
	declarations := governance.StartramContractDeclarations()
	if len(declarations) == 0 {
		t.Fatal("expected startram governance declarations")
	}
	entries := contracts.StartramContractCatalogEntries()
	byID := make(map[contracts.ContractID]struct{}, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = struct{}{}
	}
	for _, declaration := range declarations {
		if _, ok := byID[contracts.ContractID(declaration.ID)]; !ok {
			t.Fatalf("governance startram declaration missing catalog entry: %s", declaration.ID)
		}
	}
}
