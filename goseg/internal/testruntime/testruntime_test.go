package testruntime

import "testing"

type applyRuntime struct {
	Name    string
	Enabled bool
	Count   int
}

func TestApplyRunsMutatorsInOrderAndSkipsNil(t *testing.T) {
	base := applyRuntime{
		Name:    "base",
		Enabled: false,
		Count:   1,
	}

	result := Apply(
		base,
		func(runtime *applyRuntime) { runtime.Enabled = true },
		nil,
		func(runtime *applyRuntime) {
			runtime.Name = runtime.Name + "-updated"
			runtime.Count += 2
		},
	)

	if !result.Enabled {
		t.Fatal("expected Enabled to be updated by mutator")
	}
	if result.Name != "base-updated" {
		t.Fatalf("unexpected Name after mutators: %q", result.Name)
	}
	if result.Count != 3 {
		t.Fatalf("expected Count 3 after mutators, got %d", result.Count)
	}
	if base.Name != "base" || base.Enabled || base.Count != 1 {
		t.Fatalf("expected base value to remain unchanged, got %+v", base)
	}
}
