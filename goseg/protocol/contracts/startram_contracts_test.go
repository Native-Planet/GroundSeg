package contracts

import (
	"testing"
	"time"
)

func TestStartramContractCatalogSpecsAreSelfConsistent(t *testing.T) {
	t.Helper()

	entries, ok := contractCatalogEntriesForFamily(contractCatalogFamilyStartram)
	if !ok {
		t.Fatal("startram contract catalog family not registered")
	}
	if len(entries) == 0 {
		t.Fatal("expected startram contract catalog entries")
	}

	seenIDs := map[ContractID]struct{}{}
	for _, entry := range entries {
		if _, exists := seenIDs[entry.ID]; exists {
			t.Fatalf("duplicate startram contract id %s", entry.ID)
		}
		seenIDs[entry.ID] = struct{}{}

		if entry.ID == "" {
			t.Fatal("startram contract entry missing id")
		}
		if entry.Namespace != "" {
			t.Fatalf("startram contract %s has namespace %q but should not bind action namespace", entry.ID, entry.Namespace)
		}
		if entry.Action != "" {
			t.Fatalf("startram contract %s has action %q but should not bind action contract", entry.ID, entry.Action)
		}
		if entry.Descriptor.Name == "" {
			t.Fatalf("startram contract %s missing name", entry.ID)
		}
		if entry.Descriptor.Description == "" {
			t.Fatalf("startram contract %s missing description", entry.ID)
		}
		if entry.Descriptor.IntroducedIn == "" {
			t.Fatalf("startram contract %s missing IntroducedIn", entry.ID)
		}
		if _, err := time.Parse(contractVersionLayout, entry.Descriptor.IntroducedIn); err != nil {
			t.Fatalf("startram contract %s has invalid IntroducedIn %q: %v", entry.ID, entry.Descriptor.IntroducedIn, err)
		}
		if entry.Descriptor.Compatibility == "" {
			t.Fatalf("startram contract %s missing compatibility", entry.ID)
		}
		if !isKnownCompatibility(entry.Descriptor.Compatibility) {
			t.Fatalf("startram contract %s has unknown compatibility %q", entry.ID, entry.Descriptor.Compatibility)
		}
		if entry.Descriptor.DeprecatedIn != "" {
			if _, err := time.Parse(contractVersionLayout, entry.Descriptor.DeprecatedIn); err != nil {
				t.Fatalf("startram contract %s has invalid DeprecatedIn %q: %v", entry.ID, entry.Descriptor.DeprecatedIn, err)
			}
		}
		if entry.Descriptor.RemovedIn != "" {
			if _, err := time.Parse(contractVersionLayout, entry.Descriptor.RemovedIn); err != nil {
				t.Fatalf("startram contract %s has invalid RemovedIn %q: %v", entry.ID, entry.Descriptor.RemovedIn, err)
			}
		}
	}

	if _, err := NewCatalogFromEntries(entries); err != nil {
		t.Fatalf("startram contract catalog fragment should be internally valid: %v", err)
	}
}

func TestStartramContractCatalogEntriesAreSnapshots(t *testing.T) {
	original, ok := contractCatalogEntriesForFamily(contractCatalogFamilyStartram)
	if !ok {
		t.Fatal("startram contract catalog family not registered")
	}
	if len(original) == 0 {
		t.Fatal("expected startram contract catalog entries")
	}

	baseline := original[0]
	original[0].ID = ContractID("startram.errors.broken")

	refreshed, ok := contractCatalogEntriesForFamily(contractCatalogFamilyStartram)
	if !ok {
		t.Fatal("startram contract catalog family not registered")
	}
	if refreshed[0].ID != baseline.ID {
		t.Fatalf("startram contract catalog entries should not be affected by caller mutation")
	}
}

func TestStartramContractCatalogSnapshotEntriesAreSnapshots(t *testing.T) {
	original, ok := contractCatalogEntriesForFamilySnapshot(contractCatalogFamilyStartram)
	if !ok {
		t.Fatal("startram contract catalog family not registered")
	}
	if len(original) == 0 {
		t.Fatal("expected startram contract snapshot entries")
	}

	baseline := original[0]
	original[0].Descriptor.Description = "mutated"

	snapshot, ok := contractCatalogEntriesForFamilySnapshot(contractCatalogFamilyStartram)
	if !ok {
		t.Fatal("startram contract catalog family not registered")
	}
	if snapshot[0].Descriptor.Description != baseline.Descriptor.Description {
		t.Fatalf("startram contract snapshot should preserve description")
	}
}
