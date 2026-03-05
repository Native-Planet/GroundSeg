package contractspec

import (
	"testing"

	"groundseg/protocol/contracts/familycatalog"
)

func TestUploadActionSpecsDelegatesToFamilyCatalog(t *testing.T) {
	got := UploadActionSpecs()
	want := familycatalog.UploadActionFamilySpecs()
	if len(got) != len(want) {
		t.Fatalf("unexpected upload action spec count: got %d want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("unexpected upload action spec at index %d: got %+v want %+v", i, got[i], want[i])
		}
	}
}
