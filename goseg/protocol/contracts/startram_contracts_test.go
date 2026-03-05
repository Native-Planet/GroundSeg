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
	if err := validateStartramContractSpecs(entries); err != nil {
		t.Fatalf("startram contract invariants should hold: %v", err)
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

func TestStartramContractDefinitionsAreDeterministic(t *testing.T) {
	entries, ok := contractCatalogEntriesForFamily(contractCatalogFamilyStartram)
	if !ok {
		t.Fatal("startram contract catalog family not registered")
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one startram contract entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.ID != APIConnectionError {
		t.Fatalf("unexpected startram contract id: got %q want %q", entry.ID, APIConnectionError)
	}
	if entry.Namespace != "" {
		t.Fatalf("startram contract should not define namespace")
	}
	if entry.Action != "" {
		t.Fatalf("startram contract should not define action")
	}
	if entry.Descriptor.Name != "APIConnectionError" {
		t.Fatalf("startram contract name mismatch: got %q want %q", entry.Descriptor.Name, "APIConnectionError")
	}
	if entry.Descriptor.Description != "Masks transport detail when the StarTram API is unavailable or unreachable." {
		t.Fatalf("startram contract description mismatch: got %q want %q", entry.Descriptor.Description, "Masks transport detail when the StarTram API is unavailable or unreachable.")
	}
	if entry.Descriptor.Message != "Unable to connect to API server" {
		t.Fatalf("startram contract message mismatch: got %q want %q", entry.Descriptor.Message, "Unable to connect to API server")
	}
	metadata := entry.Descriptor.ContractMetadata
	if metadata != (ContractMetadata{
		IntroducedIn:  "2026.01.20",
		Compatibility: CompatibilityBackwardSafe,
	}) {
		t.Fatalf("startram contract metadata mismatch: got %#v", metadata)
	}
}

func TestStartramContractCatalogSpecsDeclareLifecycleMetadata(t *testing.T) {
	entries := startramContractCatalogSpecs()
	if len(entries) == 0 {
		t.Fatal("expected startram contract catalog specs")
	}
	if len(entries) != 1 {
		t.Fatalf("expected one startram contract spec, got %d", len(entries))
	}
	if err := validateStartramContractSpecs(entries); err != nil {
		t.Fatalf("startram contract invariant validation failed: %v", err)
	}

	entry := entries[0]
	if entry.ID != APIConnectionError {
		t.Fatalf("unexpected startram contract id %q", entry.ID)
	}
	if entry.Namespace != "" || entry.Action != "" {
		t.Fatalf("startram contract %q must not define namespace/action: namespace=%q action=%q", entry.ID, entry.Namespace, entry.Action)
	}
	if entry.Descriptor.Name == "" || entry.Descriptor.Description == "" || entry.Descriptor.Message == "" {
		t.Fatalf("startram contract %q has incomplete descriptor metadata", entry.ID)
	}

	metadata := entry.Descriptor.ContractMetadata
	if metadata.IntroducedIn == "" {
		t.Fatalf("startram contract %q missing introduced metadata", entry.ID)
	}
	if _, err := time.Parse(contractVersionLayout, metadata.IntroducedIn); err != nil {
		t.Fatalf("startram contract %q has invalid introduced version %q: %v", entry.ID, metadata.IntroducedIn, err)
	}
	if metadata.Compatibility != CompatibilityBackwardSafe {
		t.Fatalf("startram contract %q expected compatibility %q, got %q", entry.ID, CompatibilityBackwardSafe, metadata.Compatibility)
	}
	if expected := contractMetadataFor(entry.ID); expected != metadata {
		t.Fatalf("startram contract %s contract metadata must match catalog metadata map: got %#v want %#v", entry.ID, metadata, expected)
	}
}
