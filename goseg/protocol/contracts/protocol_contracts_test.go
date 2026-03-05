package contracts

import (
	"strings"
	"testing"
	"time"
)

func TestProtocolContractCatalogSpecsAreSelfConsistent(t *testing.T) {
	t.Helper()

	entries, ok := contractCatalogEntriesForFamily(contractCatalogFamilyProtocol)
	if !ok {
		t.Fatal("protocol contract catalog family not registered")
	}
	if len(entries) == 0 {
		t.Fatal("expected protocol contract catalog entries")
	}
	if err := validateProtocolContractSpecs(entries); err != nil {
		t.Fatalf("protocol contract invariants should hold: %v", err)
	}

	seenIDs := map[ContractID]struct{}{}
	for _, entry := range entries {
		if _, exists := seenIDs[entry.ID]; exists {
			t.Fatalf("duplicate protocol contract id %s", entry.ID)
		}
		seenIDs[entry.ID] = struct{}{}

		if entry.ID == "" {
			t.Fatal("protocol contract entry missing id")
		}
		if entry.Namespace == "" {
			t.Fatalf("protocol contract %s missing namespace", entry.ID)
		}
		if entry.Action == "" {
			t.Fatalf("protocol contract %s missing action", entry.ID)
		}
		if !strings.HasPrefix(string(entry.ID), "protocol.actions.") {
			t.Fatalf("protocol action contract id %s must start with protocol.actions.", entry.ID)
		}
		namespacePrefix := "protocol.actions." + string(entry.Namespace) + "."
		if !strings.HasPrefix(string(entry.ID), namespacePrefix) {
			t.Fatalf("protocol contract id %s must include namespace prefix %q", entry.ID, namespacePrefix)
		}
		if !strings.HasSuffix(string(entry.ID), "."+string(entry.Action)) {
			t.Fatalf("protocol contract %s should be suffixed by action %s", entry.ID, entry.Action)
		}

		if entry.Descriptor.Name == "" {
			t.Fatalf("protocol contract %s missing name", entry.ID)
		}
		if entry.Descriptor.Description == "" {
			t.Fatalf("protocol contract %s missing description", entry.ID)
		}
		if entry.Descriptor.IntroducedIn == "" {
			t.Fatalf("protocol contract %s missing IntroducedIn", entry.ID)
		}
		if _, err := time.Parse(contractVersionLayout, entry.Descriptor.IntroducedIn); err != nil {
			t.Fatalf("protocol contract %s has invalid IntroducedIn %q: %v", entry.ID, entry.Descriptor.IntroducedIn, err)
		}
		if entry.Descriptor.Compatibility == "" {
			t.Fatalf("protocol contract %s missing compatibility", entry.ID)
		}
		if !isKnownCompatibility(entry.Descriptor.Compatibility) {
			t.Fatalf("protocol contract %s has unknown compatibility %q", entry.ID, entry.Descriptor.Compatibility)
		}
		if entry.Descriptor.DeprecatedIn != "" {
			if _, err := time.Parse(contractVersionLayout, entry.Descriptor.DeprecatedIn); err != nil {
				t.Fatalf("protocol contract %s has invalid DeprecatedIn %q: %v", entry.ID, entry.Descriptor.DeprecatedIn, err)
			}
		}
		if entry.Descriptor.RemovedIn != "" {
			if _, err := time.Parse(contractVersionLayout, entry.Descriptor.RemovedIn); err != nil {
				t.Fatalf("protocol contract %s has invalid RemovedIn %q: %v", entry.ID, entry.Descriptor.RemovedIn, err)
			}
		}
	}

	if _, err := NewCatalogFromEntries(entries); err != nil {
		t.Fatalf("protocol contract catalog fragment should be internally valid: %v", err)
	}
}

func TestProtocolContractCatalogEntriesAreSnapshots(t *testing.T) {
	original, ok := contractCatalogEntriesForFamily(contractCatalogFamilyProtocol)
	if !ok {
		t.Fatal("protocol contract catalog family not registered")
	}
	if len(original) == 0 {
		t.Fatal("expected protocol contract catalog entries")
	}

	baseline := original[0]
	original[0].ID = ContractID("protocol.actions.broken.suffix")

	refreshed, ok := contractCatalogEntriesForFamily(contractCatalogFamilyProtocol)
	if !ok {
		t.Fatal("protocol contract catalog family not registered")
	}
	if refreshed[0].ID != baseline.ID {
		t.Fatalf("protocol contract catalog entries should not be affected by caller mutation")
	}
}

func TestProtocolContractCatalogSnapshotAreSnapshots(t *testing.T) {
	original, ok := contractCatalogEntriesForFamilySnapshot(contractCatalogFamilyProtocol)
	if !ok {
		t.Fatal("protocol contract catalog family not registered")
	}
	if len(original) == 0 {
		t.Fatal("expected protocol contract snapshot entries")
	}

	baseline := original[0]
	original[0].Action = ActionVerb("mutated")
	original[0].Namespace = ActionNamespace("mutated")
	original[0].Descriptor.IntroducedIn = "1900.01.01"

	refreshed, ok := contractCatalogEntriesForFamilySnapshot(contractCatalogFamilyProtocol)
	if !ok {
		t.Fatal("protocol contract catalog family not registered")
	}
	if refreshed[0].Action != baseline.Action {
		t.Fatalf("protocol contract snapshot should preserve action")
	}
	if refreshed[0].Namespace != baseline.Namespace {
		t.Fatalf("protocol contract snapshot should preserve namespace")
	}
	if refreshed[0].Descriptor.IntroducedIn != baseline.Descriptor.IntroducedIn {
		t.Fatalf("protocol contract snapshot should preserve IntroducedIn")
	}
}

func TestProtocolContractDefinitionsAreDeterministic(t *testing.T) {
	entries, ok := contractCatalogEntriesForFamily(contractCatalogFamilyProtocol)
	if !ok {
		t.Fatal("protocol contract catalog family not registered")
	}

	cases := []struct {
		id        ContractID
		namespace ActionNamespace
		action    ActionVerb
		name      string
		desc      string
		metadata  ContractMetadata
	}{
		{
			id:        UploadOpenEndpointAction,
			namespace: ActionNamespaceUpload,
			action:    ActionUploadOpenEndpoint,
			name:      "UploadOpenEndpointAction",
			desc:      "open upload endpoint",
			metadata: ContractMetadata{
				IntroducedIn:  "2026.03.02",
				Compatibility: CompatibilityBackwardSafe,
			},
		},
		{
			id:        UploadResetAction,
			namespace: ActionNamespaceUpload,
			action:    ActionUploadReset,
			name:      "UploadResetAction",
			desc:      "reset upload session",
			metadata: ContractMetadata{
				IntroducedIn:  "2026.03.02",
				Compatibility: CompatibilityBackwardSafe,
			},
		},
		{
			id:        C2CConnectAction,
			namespace: ActionNamespaceC2C,
			action:    ActionC2CConnect,
			name:      "C2CConnectAction",
			desc:      "connect c2c client",
			metadata: ContractMetadata{
				IntroducedIn:  "2026.03.02",
				Compatibility: CompatibilityBackwardSafe,
			},
		},
	}

	entryByID := make(map[ContractID]contractCatalogEntry, len(entries))
	for _, entry := range entries {
		entryByID[entry.ID] = entry
	}

	for _, tc := range cases {
		entry, ok := entryByID[tc.id]
		if !ok {
			t.Fatalf("protocol contract %s missing from catalog specs", tc.id)
		}
		if entry.Namespace != tc.namespace {
			t.Fatalf("protocol contract %s namespace mismatch: got %q want %q", tc.id, entry.Namespace, tc.namespace)
		}
		if entry.Action != tc.action {
			t.Fatalf("protocol contract %s action mismatch: got %q want %q", tc.id, entry.Action, tc.action)
		}
		if entry.Descriptor.Name != tc.name {
			t.Fatalf("protocol contract %s name mismatch: got %q want %q", tc.id, entry.Descriptor.Name, tc.name)
		}
		if entry.Descriptor.Description != tc.desc {
			t.Fatalf("protocol contract %s description mismatch: got %q want %q", tc.id, entry.Descriptor.Description, tc.desc)
		}
		if got := entry.Descriptor.ContractMetadata; got != tc.metadata {
			t.Fatalf("protocol contract %s metadata mismatch: got %#v want %#v", tc.id, got, tc.metadata)
		}
	}
}

func TestProtocolContractCatalogSpecsDeclareStableLifecycles(t *testing.T) {
	entries := protocolContractCatalogSpecs()
	if len(entries) == 0 {
		t.Fatal("expected protocol contract catalog specs")
	}
	if err := validateProtocolContractSpecs(entries); err != nil {
		t.Fatalf("protocol contract invariant validation failed: %v", err)
	}

	for _, entry := range entries {
		if entry.ID == "" || entry.Namespace == "" || entry.Action == "" {
			t.Fatalf("protocol contract spec missing core identity: %+v", entry)
		}
		if entry.ID != ContractID("protocol.actions."+string(entry.Namespace)+"."+string(entry.Action)) {
			t.Fatalf("protocol contract %s identity should follow protocol actions contract naming", entry.ID)
		}
		if entry.Descriptor.Name == "" {
			t.Fatalf("protocol contract %s missing descriptor name", entry.ID)
		}
		if entry.Descriptor.Description == "" {
			t.Fatalf("protocol contract %s missing descriptor description", entry.ID)
		}

		metadata := entry.Descriptor.ContractMetadata
		if metadata.IntroducedIn == "" {
			t.Fatalf("protocol contract %s missing introduced metadata", entry.ID)
		}
		if metadata.Compatibility == "" {
			t.Fatalf("protocol contract %s missing compatibility metadata", entry.ID)
		}
		if !isKnownCompatibility(metadata.Compatibility) {
			t.Fatalf("protocol contract %s unknown compatibility %q", entry.ID, metadata.Compatibility)
		}
		if _, err := time.Parse(contractVersionLayout, metadata.IntroducedIn); err != nil {
			t.Fatalf("protocol contract %s has invalid introduced version %q: %v", entry.ID, metadata.IntroducedIn, err)
		}
		if metadata.RemovedIn != "" && metadata.DeprecatedIn != "" && !IsVersionAtLeastOrEqual(metadata.RemovedIn, metadata.DeprecatedIn) {
			t.Fatalf("protocol contract %s has removed-before-deprecated lifecycle", entry.ID)
		}
		if expected := contractMetadataFor(entry.ID); expected != metadata {
			t.Fatalf("protocol contract %s contract metadata must match catalog metadata map: got %#v want %#v", entry.ID, metadata, expected)
		}
	}
}
