package contracts

import "testing"

func TestGovernanceSurfaceCoversProtocolAndStartramContracts(t *testing.T) {
	if err := ValidateGovernanceSchema(); err != nil {
		t.Fatalf("governance schema validation failed: %v", err)
	}

	entries := ContractCatalogEntries()
	if len(entries) == 0 {
		t.Fatal("expected contract catalog entries")
	}

	seenProtocol := map[ContractID]struct{}{}
	seenStartram := map[ContractID]struct{}{}
	for _, entry := range entries {
		if entry.ID == "" {
			t.Fatal("contract entry missing id")
		}
		if entry.Descriptor.Name == "" || entry.Descriptor.Description == "" {
			t.Fatalf("contract %s missing governance descriptor fields", entry.ID)
		}
		if entry.Namespace == "" && entry.Action == "" {
			seenStartram[entry.ID] = struct{}{}
			continue
		}
		expectedID := protocolActionContractID(entry.Namespace, entry.Action)
		if entry.ID != expectedID {
			t.Fatalf("protocol action %s:%s id drift: got %s want %s", entry.Namespace, entry.Action, entry.ID, expectedID)
		}
		seenProtocol[entry.ID] = struct{}{}
	}

	requiredProtocol := []ContractID{
		UploadOpenEndpointAction,
		UploadResetAction,
		C2CConnectAction,
	}
	for _, id := range requiredProtocol {
		if _, ok := seenProtocol[id]; !ok {
			t.Fatalf("missing required protocol contract id %s", id)
		}
	}

	descriptor, ok := ContractDescriptorFor(APIConnectionError)
	if !ok {
		t.Fatalf("missing startram contract descriptor for %s", APIConnectionError)
	}
	if descriptor.Message != APIConnectionErrorMessage {
		t.Fatalf("startram API connection message drift: got %q want %q", descriptor.Message, APIConnectionErrorMessage)
	}
	if _, ok := seenStartram[APIConnectionError]; !ok {
		t.Fatalf("missing required startram contract id %s", APIConnectionError)
	}

	for _, family := range []string{contractCatalogFamilyProtocol, contractCatalogFamilyStartram} {
		familyEntries, ok := contractCatalogEntriesForFamilySnapshot(family)
		if !ok {
			t.Fatalf("missing family snapshot for %s", family)
		}
		if len(familyEntries) == 0 {
			t.Fatalf("family %s has no catalog entries", family)
		}
		for _, entry := range familyEntries {
			if entry.Governance.OwnerModule == "" {
				t.Fatalf("contract %s missing governance owner module", entry.ID)
			}
			if entry.Governance.SinceVersion == "" {
				t.Fatalf("contract %s missing governance since version", entry.ID)
			}
			if !isKnownCompatibility(entry.Governance.Stability) {
				t.Fatalf("contract %s has unknown governance stability %q", entry.ID, entry.Governance.Stability)
			}
		}
	}
}
