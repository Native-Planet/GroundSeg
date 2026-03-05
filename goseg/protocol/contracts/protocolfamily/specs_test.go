package protocolfamily

import (
	"testing"

	"groundseg/protocol/contracts/familycatalog"
)

func TestActionSpecsDelegatesToFamilyCatalog(t *testing.T) {
	got := ActionSpecs()
	want := familycatalog.ProtocolActionSpecs()
	if len(got) != len(want) {
		t.Fatalf("unexpected protocol family action spec count: got %d want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("unexpected protocol family spec at index %d: got %+v want %+v", i, got[i], want[i])
		}
	}
}
