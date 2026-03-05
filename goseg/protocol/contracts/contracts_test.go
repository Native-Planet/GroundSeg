package contracts

import (
	"testing"
	"time"
)

func mustLoadCatalog(t *testing.T) *ContractCatalog {
	t.Helper()
	catalog, err := LoadRegistry()
	if err != nil {
		t.Fatalf("failed to load contract catalog: %v", err)
	}
	return catalog
}

func TestContractCatalogHasLifecycleMetadata(t *testing.T) {
	catalog := mustLoadCatalog(t)
	for id, descriptor := range catalog.ContractDescriptorsForDebug() {
		if descriptor.Name == "" {
			t.Fatalf("contract %s has empty name", id)
		}
		if descriptor.Description == "" {
			t.Fatalf("contract %s has empty description", id)
		}
		if !isKnownCompatibility(descriptor.Compatibility) {
			t.Fatalf("contract %s has unknown compatibility %q", descriptor.Name, descriptor.Compatibility)
		}
		if descriptor.Compatibility == "" {
			t.Fatalf("contract %s has empty compatibility", descriptor.Name)
		}
		if descriptor.IntroducedIn == "" {
			t.Fatalf("contract %s has empty introduction version", descriptor.Name)
		}
		if _, err := time.Parse(contractVersionLayout, descriptor.IntroducedIn); err != nil {
			t.Fatalf("contract %s has invalid introduced version %q: %v", descriptor.Name, descriptor.IntroducedIn, err)
		}
		if descriptor.DeprecatedIn != "" {
			if _, err := time.Parse(contractVersionLayout, descriptor.DeprecatedIn); err != nil {
				t.Fatalf("contract %s has invalid deprecated version %q: %v", descriptor.Name, descriptor.DeprecatedIn, err)
			}
		}
		if descriptor.RemovedIn != "" {
			if _, err := time.Parse(contractVersionLayout, descriptor.RemovedIn); err != nil {
				t.Fatalf("contract %s has invalid removed version %q: %v", descriptor.Name, descriptor.RemovedIn, err)
			}
		}
		if descriptor.RemovedIn != "" && descriptor.DeprecatedIn != "" && !IsVersionAtLeastOrEqual(descriptor.RemovedIn, descriptor.DeprecatedIn) {
			t.Fatalf("contract %s has removed-before-deprecated lifecycle windows", descriptor.Name)
		}
		if !descriptor.IsActive(CurrentContractVersion) && descriptor.RemovedIn == "" {
			t.Fatalf("contract %s is currently inactive without removal date", descriptor.Name)
		}
	}
}

func TestContractLifecyclePolicyDeclaredPerContract(t *testing.T) {
	catalog := mustLoadCatalog(t)
	descriptors := catalog.ContractDescriptorsForDebug()
	entries := ContractCatalogEntries()
	expected := make(map[ContractID]ContractMetadata, len(entries))
	for _, entry := range entries {
		expected[entry.ID] = entry.Descriptor.ContractMetadata
	}

	if len(descriptors) != len(expected) {
		t.Fatalf("contract catalog changed: expected %d contracts, got %d", len(expected), len(descriptors))
	}

	for id, expectedMetadata := range expected {
		got, ok := descriptors[id]
		if !ok {
			t.Fatalf("missing contract descriptor for %s", id)
		}
		if got.IntroducedIn != expectedMetadata.IntroducedIn {
			t.Fatalf("contract %s introduced version changed without explicit update: got %s", id, got.IntroducedIn)
		}
		if got.Compatibility != expectedMetadata.Compatibility {
			t.Fatalf("contract %s compatibility changed without explicit update: got %s", id, got.Compatibility)
		}
		if got.DeprecatedIn != expectedMetadata.DeprecatedIn {
			t.Fatalf("contract %s deprecated version changed without explicit update: got %s", id, got.DeprecatedIn)
		}
		if got.RemovedIn != expectedMetadata.RemovedIn {
			t.Fatalf("contract %s removed version changed without explicit update: got %s", id, got.RemovedIn)
		}
	}
}

func TestContractIntroductionsHaveExplicitAnchors(t *testing.T) {
	entries := ContractCatalogEntries()
	for _, entry := range entries {
		introduced := entry.Descriptor.IntroducedIn
		id := entry.ID

		if introduced == "" {
			t.Fatalf("contract %s missing introduced anchor", id)
		}
		if _, err := time.Parse(contractVersionLayout, introduced); err != nil {
			t.Fatalf("contract %s has invalid introduced anchor %q: %v", id, introduced, err)
		}
	}
}

func TestContractDescriptorForUnknownIDReturnsMissing(t *testing.T) {
	if descriptor, ok := ContractDescriptorFor("protocol.contracts.does-not-exist"); ok {
		t.Fatalf("expected unknown contract id lookup to fail, got %+v", descriptor)
	}
}

func TestMustContractDescriptorPanicsForUnknownID(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected MustContractDescriptor to panic for unknown contract id")
		}
	}()
	_ = MustContractDescriptor(ContractID("protocol.contracts.does-not-exist"))
}

func TestActionContractBindingsHaveActiveDescriptors(t *testing.T) {
	for _, binding := range ActionContractBindings() {
		if string(binding.Namespace) == "" {
			t.Fatalf("missing namespace for action binding %q", binding.Action)
		}
		if binding.Action == "" {
			t.Fatalf("missing action value for namespace %s", binding.Namespace)
		}
		descriptor, ok := ContractDescriptorFor(binding.Contract)
		if !ok {
			t.Fatalf("missing contract %q for action %s:%s", binding.Contract, binding.Namespace, binding.Action)
		}
		if descriptor.Name == "" {
			t.Fatalf("contract %q has empty name", binding.Contract)
		}
		if descriptor.Description == "" {
			t.Fatalf("contract %q has empty description", binding.Contract)
		}
	}
}

func TestActionContractBindingsReturnedAsCopy(t *testing.T) {
	snapshot := ActionContractBindings()
	if len(snapshot) == 0 {
		t.Fatal("expected action contract bindings")
	}

	original := snapshot[0]
	snapshot[0].Action = ActionVerb("mutation-test")
	refreshed := ActionContractBindings()
	if refreshed[0].Action != original.Action {
		t.Fatalf("expected action contract bindings to be immutable from caller mutation")
	}
}

func TestProtocolActionBindingIDsFollowNamespaceActionPolicy(t *testing.T) {
	entries := ContractCatalogEntries()
	for _, entry := range entries {
		if entry.Namespace == "" && entry.Action == "" {
			continue
		}
		expected := protocolActionContractID(entry.Namespace, entry.Action)
		if entry.ID != expected {
			t.Fatalf("protocol action contract id mismatch for %s:%s: got %s want %s", entry.Namespace, entry.Action, entry.ID, expected)
		}
	}
}

func TestActionContractBindingsAreDeterministicallyOrdered(t *testing.T) {
	uploadBindings := ActionContractBindingsForNamespace(ActionNamespaceUpload)
	if len(uploadBindings) != 2 {
		t.Fatalf("expected 2 upload action bindings, got %d", len(uploadBindings))
	}
	if uploadBindings[0].Action != ActionUploadOpenEndpoint || uploadBindings[1].Action != ActionUploadReset {
		t.Fatalf("unexpected upload binding ordering: %#v", uploadBindings)
	}
	c2cBindings := ActionContractBindingsForNamespace(ActionNamespaceC2C)
	if len(c2cBindings) != 1 || c2cBindings[0].Action != ActionC2CConnect {
		t.Fatalf("unexpected c2c binding ordering: %#v", c2cBindings)
	}
}

func TestTypedActionContractLookupRejectsUnknownNamespaceActionPairs(t *testing.T) {
	catalog, err := LoadRegistry()
	if err != nil {
		t.Fatalf("failed to load contract catalog: %v", err)
	}

	if _, ok := catalog.ActionContractFor(ActionNamespace("invalid"), ActionVerb("open-endpoint")); ok {
		t.Fatal("expected invalid namespace lookup to fail")
	}
	if _, ok := catalog.ActionContractFor(ActionNamespaceUpload, ActionVerb("does-not-exist")); ok {
		t.Fatal("expected invalid action lookup to fail")
	}
	if _, ok := catalog.ActionContractFor(ActionNamespace(""), ActionVerb("")); ok {
		t.Fatal("expected empty namespace/action lookup to fail")
	}
}

func TestActionContractForBindingUsesTypedBindingPath(t *testing.T) {
	binding := ActionContractBinding{
		Namespace: ActionNamespaceUpload,
		Action:    ActionUploadOpenEndpoint,
	}
	contract, ok := ActionContractForBinding(binding)
	if !ok {
		t.Fatal("expected typed binding lookup to succeed")
	}
	if contract.Name == "" {
		t.Fatal("expected contract descriptor to include name")
	}
}

func TestProtocolContractCatalogMatchesCanonicalConstants(t *testing.T) {
	entries := ContractCatalogEntries()
	totalByID := make(map[ContractID]contractCatalogEntry, len(entries))
	for _, entry := range entries {
		if entry.ID == "" {
			t.Fatal("catalog entry missing id")
		}
		if _, exists := totalByID[entry.ID]; exists {
			t.Fatalf("duplicate contract id %s in catalog specs", entry.ID)
		}
		totalByID[entry.ID] = entry
	}
	if len(totalByID) == 0 {
		t.Fatal("expected at least one contract catalog spec entry")
	}

	catalog := mustLoadCatalog(t)
	for _, entry := range entries {
		contract, ok := catalog.ContractDescriptorFor(entry.ID)
		if !ok {
			t.Fatalf("contract %s from canonical specs not present in loaded catalog", entry.ID)
		}
		if contract.Name != entry.Descriptor.Name {
			t.Fatalf("contract %s descriptor name mismatch between canonical spec and catalog", entry.ID)
		}
		if contract.Compatibility != entry.Descriptor.Compatibility {
			t.Fatalf("contract %s compatibility mismatch between canonical spec and catalog", entry.ID)
		}
	}
}

func TestContractCatalogInitErrorReturnsNilWhenInitialized(t *testing.T) {
	if err := ContractCatalogInitError(); err != nil {
		t.Fatalf("expected initialized catalog to have no error, got %v", err)
	}
}
