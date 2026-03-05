package conformance

import (
	"testing"

	"groundseg/protocol/contracts"
	"groundseg/protocol/contracts/familycatalog"
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

func TestUploadActionBindingSpecsExposePayloadRules(t *testing.T) {
	specs := contracts.UploadActionBindingSpecs()
	if len(specs) == 0 {
		t.Fatal("expected upload action binding specs")
	}
	seen := make(map[contracts.ActionVerb]struct{}, len(specs))
	for _, spec := range specs {
		if spec.RequiredPayloads.Has(spec.ForbiddenPayloads) {
			t.Fatalf("upload payload rules overlap for %s", spec.Action)
		}
		if spec.RequiredPayloads.IsEmpty() {
			t.Fatalf("upload payload rules missing required payload for %s", spec.Action)
		}
		seen[spec.Action] = struct{}{}
	}
	if _, ok := seen[contracts.ActionUploadOpenEndpoint]; !ok {
		t.Fatalf("missing payload-rule spec for action %s", contracts.ActionUploadOpenEndpoint)
	}
	if _, ok := seen[contracts.ActionUploadReset]; !ok {
		t.Fatalf("missing payload-rule spec for action %s", contracts.ActionUploadReset)
	}
}

func TestStartramContractSpecsMatchCanonicalCatalog(t *testing.T) {
	specs := familycatalog.StartramContractSpecs()
	if len(specs) == 0 {
		t.Fatal("expected startram contract specs")
	}

	type expectedStartramSpec struct {
		name        string
		description string
		message     string
		owner       string
	}

	expectedByID := make(map[contracts.ContractID]expectedStartramSpec, len(specs))
	for _, spec := range specs {
		id := contracts.ContractID(spec.ID)
		if _, exists := expectedByID[id]; exists {
			t.Fatalf("duplicate startram spec id %s", id)
		}
		expectedByID[id] = expectedStartramSpec{
			name:        spec.Name,
			description: spec.Description,
			message:     spec.Message,
			owner:       spec.Owner,
		}
	}

	entries := contracts.StartramContractCatalogEntries()
	if len(entries) == 0 {
		t.Fatal("expected startram entries in canonical contract catalog")
	}
	seen := 0
	for _, entry := range entries {
		expected, ok := expectedByID[entry.ID]
		if !ok {
			t.Fatalf("unexpected startram catalog entry %s", entry.ID)
		}
		if entry.Descriptor.Name != expected.name {
			t.Fatalf("startram descriptor name drift for %s: got %q want %q", entry.ID, entry.Descriptor.Name, expected.name)
		}
		if entry.Descriptor.Description != expected.description {
			t.Fatalf("startram descriptor description drift for %s: got %q want %q", entry.ID, entry.Descriptor.Description, expected.description)
		}
		if entry.Descriptor.Message != expected.message {
			t.Fatalf("startram descriptor message drift for %s: got %q want %q", entry.ID, entry.Descriptor.Message, expected.message)
		}
		if entry.Governance.OwnerModule != expected.owner {
			t.Fatalf("startram governance owner drift for %s: got %q want %q", entry.ID, entry.Governance.OwnerModule, expected.owner)
		}
		delete(expectedByID, entry.ID)
		seen++
	}
	if seen == 0 {
		t.Fatal("expected startram entries in canonical contract catalog")
	}
	for missingID := range expectedByID {
		t.Fatalf("missing startram contract entry %s in canonical contract catalog", missingID)
	}
}
