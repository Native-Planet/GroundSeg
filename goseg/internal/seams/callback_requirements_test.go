package seams

import (
	"reflect"
	"testing"
)

func TestCallbackRequirementsTracksConfiguredDependencies(t *testing.T) {
	requirements := NewCallbackRequirements("alpha", "beta", "gamma")
	requirements = requirements.With("alpha", true)
	requirements = requirements.With("beta", false)
	requirements = requirements.With("gamma", true)

	got := requirements.Missing()
	want := []string{"beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected missing callbacks: got %v want %v", got, want)
	}
}
