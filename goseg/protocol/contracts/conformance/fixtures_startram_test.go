package conformance

import (
	"testing"

	"groundseg/protocol/contracts"
	"groundseg/protocol/contracts/familycatalog"
)

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
