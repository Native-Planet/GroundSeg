package startram

import "testing"

func TestContractSpecs(t *testing.T) {
	specs := ContractSpecs()
	if len(specs) != 1 {
		t.Fatalf("expected one startram contract spec, got %d", len(specs))
	}
	spec := specs[0]
	if spec.ID != StartramAPIConnectionErrorID {
		t.Fatalf("unexpected id: %q", spec.ID)
	}
	if spec.Owner != string(OwnerStartram) {
		t.Fatalf("unexpected owner: %q", spec.Owner)
	}
}
