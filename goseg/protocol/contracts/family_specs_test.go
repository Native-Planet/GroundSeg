package contracts

import (
	"strings"
	"testing"

	"groundseg/protocol/contracts/familycatalog"
)

func TestBuildContractFamilyEntriesFromSpecs(t *testing.T) {
	specs := []contractFamilyEntrySpec{
		{
			ID:          UploadOpenEndpointAction,
			Namespace:   ActionNamespaceUpload,
			Action:      ActionUploadOpenEndpoint,
			Name:        familycatalog.UploadOpenEndpointActionName,
			Description: familycatalog.UploadOpenEndpointActionDescription,
			Message:     familycatalog.UploadOpenEndpointActionDescription,
			Governance: contractGovernanceMetadata{
				OwnerModule:  string(familycatalog.OwnerUploadService),
				Stability:    CompatibilityBackwardSafe,
				SinceVersion: contractMetadataFor(UploadOpenEndpointAction).IntroducedIn,
			},
		},
	}

	entries := buildContractFamilyEntries(specs)
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.ID != UploadOpenEndpointAction {
		t.Fatalf("unexpected id: %s", entry.ID)
	}
	if entry.Namespace != ActionNamespaceUpload || entry.Action != ActionUploadOpenEndpoint {
		t.Fatalf("unexpected binding: %s:%s", entry.Namespace, entry.Action)
	}
	if entry.Descriptor.Name != familycatalog.UploadOpenEndpointActionName {
		t.Fatalf("unexpected descriptor name: %q", entry.Descriptor.Name)
	}
	if entry.Descriptor.Message != familycatalog.UploadOpenEndpointActionDescription {
		t.Fatalf("unexpected descriptor message: %q", entry.Descriptor.Message)
	}
}

func TestValidateContractFamilySpecsRejectsUnexpectedID(t *testing.T) {
	specs := []contractFamilyEntrySpec{
		{
			ID:          UploadOpenEndpointAction,
			Namespace:   ActionNamespaceUpload,
			Action:      ActionUploadOpenEndpoint,
			Name:        familycatalog.UploadOpenEndpointActionName,
			Description: familycatalog.UploadOpenEndpointActionDescription,
			Message:     familycatalog.UploadOpenEndpointActionDescription,
			Governance: contractGovernanceMetadata{
				OwnerModule:  string(familycatalog.OwnerUploadService),
				Stability:    CompatibilityBackwardSafe,
				SinceVersion: contractMetadataFor(UploadOpenEndpointAction).IntroducedIn,
			},
		},
	}
	entries := buildContractFamilyEntries(specs)

	err := validateContractFamilySpecs(entries, contractFamilyValidationSpec{
		Family:               "protocol",
		IDPrefix:             "protocol.actions.upload.",
		RequireActionBinding: true,
		ExpectedIDs: map[ContractID]struct{}{
			UploadResetAction: {},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "unexpected protocol contract id") {
		t.Fatalf("expected unexpected-id validation error, got %v", err)
	}
}

func TestValidateCatalogEntryGovernanceMetadataRejectsUnknownDeprecationTarget(t *testing.T) {
	entry := contractCatalogEntry{
		ID: UploadOpenEndpointAction,
		Governance: contractGovernanceMetadata{
			OwnerModule:  string(familycatalog.OwnerUploadService),
			Stability:    CompatibilityBackwardSafe,
			SinceVersion: contractMetadataFor(UploadOpenEndpointAction).IntroducedIn,
			Deprecates:   ContractID("protocol.actions.upload.missing"),
		},
		Descriptor: ContractDescriptor{
			Name:             familycatalog.UploadOpenEndpointActionName,
			Description:      familycatalog.UploadOpenEndpointActionDescription,
			ContractMetadata: contractMetadataFor(UploadOpenEndpointAction),
		},
	}

	err := validateCatalogEntryGovernanceMetadata(entry, map[ContractID]struct{}{
		UploadOpenEndpointAction: {},
	})
	if err == nil || !strings.Contains(err.Error(), "deprecates references unknown contract") {
		t.Fatalf("expected unknown deprecates validation error, got %v", err)
	}
}

func TestExpectedIDSetAndBindingIndexFromSpecs(t *testing.T) {
	specs := []contractFamilyEntrySpec{
		{
			ID:        UploadOpenEndpointAction,
			Namespace: ActionNamespaceUpload,
			Action:    ActionUploadOpenEndpoint,
		},
		{
			ID:        C2CConnectAction,
			Namespace: ActionNamespaceC2C,
			Action:    ActionC2CConnect,
		},
	}

	ids := expectedIDSetFromSpecs(specs)
	if len(ids) != 2 {
		t.Fatalf("expected two ids, got %d", len(ids))
	}
	if _, ok := ids[UploadOpenEndpointAction]; !ok {
		t.Fatal("expected upload action id in expected set")
	}
	if _, ok := ids[C2CConnectAction]; !ok {
		t.Fatal("expected c2c action id in expected set")
	}

	index := actionBindingIDIndexFromSpecs(specs)
	if len(index) != 2 {
		t.Fatalf("expected two bindings, got %d", len(index))
	}
	if got := index[actionContractBindingKey{Namespace: ActionNamespaceUpload, Action: ActionUploadOpenEndpoint}]; got != UploadOpenEndpointAction {
		t.Fatalf("unexpected upload binding id: %s", got)
	}
	if got := index[actionContractBindingKey{Namespace: ActionNamespaceC2C, Action: ActionC2CConnect}]; got != C2CConnectAction {
		t.Fatalf("unexpected c2c binding id: %s", got)
	}
}

func TestProtocolFamilyDeclarationsValidateAgainstGovernance(t *testing.T) {
	actionSpecs := familycatalog.ProtocolActionSpecs()
	uploadSpecs := familycatalog.UploadActionSpecs()
	if len(actionSpecs) == 0 {
		t.Fatal("expected protocol family action declarations")
	}
	if len(uploadSpecs) == 0 {
		t.Fatal("expected upload action declarations")
	}
	entries := protocolContractCatalogSpecs()
	if err := validateProtocolContractSpecs(entries); err != nil {
		t.Fatalf("validate protocol contract declarations: %v", err)
	}
}

func TestStartramFamilyDeclarationsValidateAgainstGovernance(t *testing.T) {
	familySpecs := familycatalog.StartramContractSpecs()
	if len(familySpecs) == 0 {
		t.Fatal("expected startram family declarations")
	}
	entries := startramContractCatalogSpecs()
	if err := validateStartramContractSpecs(entries); err != nil {
		t.Fatalf("validate startram contract declarations: %v", err)
	}
}
