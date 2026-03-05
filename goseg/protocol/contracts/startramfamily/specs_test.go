package startramfamily

import (
	"testing"

	"groundseg/protocol/contracts/familycatalog"
)

func TestContractSpecsDelegatesToFamilyCatalog(t *testing.T) {
	got := ContractSpecs()
	want := familycatalog.StartramContractSpecs()
	if len(got) != len(want) {
		t.Fatalf("unexpected startram family contract spec count: got %d want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("unexpected startram family spec at index %d: got %+v want %+v", i, got[i], want[i])
		}
	}
}
