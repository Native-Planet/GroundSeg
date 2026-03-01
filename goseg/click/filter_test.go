package click

import (
	"strings"
	"testing"
)

func TestFilterResponseSuccess(t *testing.T) {
	_, ok, err := filterResponse("success", "line\n[0 %avow 0 %noun %success]\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected success to be true")
	}

	_, ok, err = filterResponse("success", "no marker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected success to be false")
	}
}

func TestFilterResponseCode(t *testing.T) {
	got, _, err := filterResponse("code", "[0 %avow 0 %noun %abcdefghijk]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "abcdefghijk" {
		t.Fatalf("unexpected code: %s", got)
	}

	_, _, err = filterResponse("code", "missing avow")
	if err == nil || !strings.Contains(err.Error(), "+code not in poke response") {
		t.Fatalf("expected +code parse error, got %v", err)
	}
}

func TestFilterResponseDesk(t *testing.T) {
	got, _, err := filterResponse("desk", "[0 %avow 0 %noun app status: running]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "running" {
		t.Fatalf("unexpected status: %s", got)
	}

	got, _, err = filterResponse("desk", "[0 %avow 0 %noun does not yet exist]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "not-found" {
		t.Fatalf("unexpected status: %s", got)
	}

	got, _, err = filterResponse("desk", "[0 %avow 0 %noun malformed]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "not-found" {
		t.Fatalf("expected not-found fallback, got: %s", got)
	}
}

func TestFilterResponseDefaultAndUnknown(t *testing.T) {
	_, _, err := filterResponse("default", "anything")
	if err == nil || !strings.Contains(err.Error(), "Unknown poke response") {
		t.Fatalf("expected unknown poke response error, got %v", err)
	}

	_, _, err = filterResponse("something-else", "anything")
	if err == nil || !strings.Contains(err.Error(), "+code not in poke response") {
		t.Fatalf("expected fallback parse error, got %v", err)
	}
}
