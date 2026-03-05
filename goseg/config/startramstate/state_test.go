package startramstate

import (
	"testing"

	"groundseg/structs"
)

func TestSetGetAndSnapshot(t *testing.T) {
	initial := GetSnapshot()

	target := structs.StartramRetrieve{
		Pubkey: "pubkey-test",
	}
	Set(target)
	t.Cleanup(func() {
		Set(initial.Value)
	})

	got := Get()
	if got.Pubkey != target.Pubkey {
		t.Fatalf("get pubkey mismatch: got %q want %q", got.Pubkey, target.Pubkey)
	}
	snapshot := GetSnapshot()
	if !snapshot.Fresh {
		t.Fatal("snapshot should be marked fresh after set")
	}
	if snapshot.UpdatedAt.IsZero() {
		t.Fatal("snapshot should capture update time")
	}
	if snapshot.Value.Pubkey != target.Pubkey {
		t.Fatalf("snapshot pubkey mismatch: got %q want %q", snapshot.Value.Pubkey, target.Pubkey)
	}
}
