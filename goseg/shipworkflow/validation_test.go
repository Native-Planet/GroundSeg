package shipworkflow

import "testing"

func TestSigRemove(t *testing.T) {
	if got := NormalizePatp("~zod"); got != "zod" {
		t.Fatalf("expected leading sigil removed, got %q", got)
	}
	if got := NormalizePatp("marzod"); got != "marzod" {
		t.Fatalf("expected unchanged value, got %q", got)
	}
}

func TestCheckPatp(t *testing.T) {
	if !IsValidPatp("zod") {
		t.Fatal("expected zod to be valid patp")
	}
	if IsValidPatp("invalid!") {
		t.Fatal("expected invalid! to be rejected")
	}
}
