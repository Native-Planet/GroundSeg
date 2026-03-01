//go:build integration

package handler

import (
	"os"
	"testing"
)

func TestCheckDevMode(t *testing.T) {
	original := os.Args
	t.Cleanup(func() {
		os.Args = original
	})

	os.Args = []string{"groundseg", "dev"}
	if !checkDevMode() {
		t.Fatal("expected dev mode when dev arg is present")
	}

	os.Args = []string{"groundseg", "start"}
	if checkDevMode() {
		t.Fatal("expected non-dev mode without dev arg")
	}
}

func TestSigRemove(t *testing.T) {
	if got := sigRemove("~zod"); got != "zod" {
		t.Fatalf("expected leading sigil removed, got %q", got)
	}
	if got := sigRemove("marzod"); got != "marzod" {
		t.Fatalf("expected unchanged value, got %q", got)
	}
}

func TestCheckPatp(t *testing.T) {
	if !checkPatp("zod") {
		t.Fatal("expected zod to be valid patp")
	}
	if checkPatp("invalid!") {
		t.Fatal("expected invalid! to be rejected")
	}
}
