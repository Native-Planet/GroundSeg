//go:build integration

package routines

import "testing"

func TestGenerateTimeOfDayDeterministic(t *testing.T) {
	a := generateTimeOfDay("~zod")
	b := generateTimeOfDay("~zod")
	if !a.Equal(b) {
		t.Fatalf("expected deterministic output for same seed: %v vs %v", a, b)
	}
}

func TestGenerateTimeOfDayRange(t *testing.T) {
	got := generateTimeOfDay("~nec")
	if got.Hour() < 0 || got.Hour() > 23 {
		t.Fatalf("hour out of range: %d", got.Hour())
	}
	if got.Minute() < 0 || got.Minute() > 59 {
		t.Fatalf("minute out of range: %d", got.Minute())
	}
	if got.Second() < 0 || got.Second() > 59 {
		t.Fatalf("second out of range: %d", got.Second())
	}
}
