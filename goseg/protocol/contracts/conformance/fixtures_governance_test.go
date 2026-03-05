package conformance

import (
	"testing"

	"groundseg/protocol/contracts"
	"groundseg/protocol/contracts/governance"
)

func TestGovernanceManifestMatchesActionBindings(t *testing.T) {
	type bindingKey struct {
		namespace contracts.ActionNamespace
		action    contracts.ActionVerb
	}
	byBinding := make(map[bindingKey]contracts.ContractID)
	for _, binding := range contracts.ActionContractBindings() {
		byBinding[bindingKey{namespace: binding.Namespace, action: binding.Action}] = binding.Contract
	}
	declarations := governance.ActionDeclarations()
	if len(declarations) == 0 {
		t.Fatal("expected governance action declarations")
	}
	if len(declarations) != len(byBinding) {
		t.Fatalf("governance declaration count mismatch: got %d want %d", len(declarations), len(byBinding))
	}
	for _, declaration := range declarations {
		key := bindingKey{
			namespace: contracts.ActionNamespace(declaration.Namespace),
			action:    contracts.ActionVerb(declaration.Action),
		}
		contractID, ok := byBinding[key]
		if !ok {
			t.Fatalf("governance declaration missing action binding %s:%s", declaration.Namespace, declaration.Action)
		}
		if contractID != contracts.ContractID(declaration.ContractID) {
			t.Fatalf(
				"governance declaration contract drift for %s:%s: got %s want %s",
				declaration.Namespace,
				declaration.Action,
				contractID,
				declaration.ContractID,
			)
		}
	}
}

func TestGovernanceManifestMatchesStartramCatalog(t *testing.T) {
	declarations := governance.StartramContractDeclarations()
	if len(declarations) == 0 {
		t.Fatal("expected governance startram declarations")
	}
	entries := contracts.StartramContractCatalogEntries()
	if len(entries) != len(declarations) {
		t.Fatalf("governance startram declaration count mismatch: got %d want %d", len(declarations), len(entries))
	}
	byID := make(map[contracts.ContractID]contracts.ContractCatalogEntry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}
	for _, declaration := range declarations {
		entry, ok := byID[contracts.ContractID(declaration.ID)]
		if !ok {
			t.Fatalf("governance startram declaration missing from catalog: %s", declaration.ID)
		}
		if entry.Descriptor.Name != declaration.Name {
			t.Fatalf("startram declaration name drift for %s", declaration.ID)
		}
		if entry.Descriptor.Description != declaration.Description {
			t.Fatalf("startram declaration description drift for %s", declaration.ID)
		}
		if entry.Descriptor.Message != declaration.Message {
			t.Fatalf("startram declaration message drift for %s", declaration.ID)
		}
		if entry.Governance.OwnerModule != declaration.Owner {
			t.Fatalf("startram declaration owner drift for %s", declaration.ID)
		}
	}
}
