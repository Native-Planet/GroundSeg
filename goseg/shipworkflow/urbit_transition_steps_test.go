package shipworkflow

import (
	"errors"
	"testing"
)

func TestBuildSingleStepTransitionWrapsRunFunction(t *testing.T) {
	called := false
	steps := buildSingleStepTransition("~zod", func() error {
		called = true
		return nil
	})
	if len(steps) != 1 {
		t.Fatalf("expected one transition step, got %d", len(steps))
	}
	if steps[0].Run == nil {
		t.Fatal("expected run function")
	}
	if err := steps[0].Run(); err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if !called {
		t.Fatal("expected wrapped run function to be invoked")
	}
}

func TestRunShipConfigTransitionReturnsPersistError(t *testing.T) {
	expected := errors.New("persist failed")
	err := runShipConfigTransition(
		"~zod",
		"operation",
		func() error { return expected },
		nil,
		nil,
		nil,
		shipConfigTransitionStrategy{},
	)
	if !errors.Is(err, expected) {
		t.Fatalf("expected persist error to be returned, got %v", err)
	}
}
