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
	original[0].Action = ActionToken("mutated")
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
