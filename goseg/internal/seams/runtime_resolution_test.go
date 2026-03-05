package seams

import (
	"errors"
	"testing"
)

type resolveRuntimeTestRuntime struct {
	Name    string
	Enabled bool
	Count   int
}

func TestResolveRuntimeUsesDefaultsWhenNoOverrides(t *testing.T) {
	defaults := resolveRuntimeTestRuntime{
		Name:    "default",
		Enabled: true,
		Count:   1,
	}
	validateCalled := false
	resolved, err := ResolveRuntime(defaults, func(runtime resolveRuntimeTestRuntime) error {
		validateCalled = true
		if runtime.Name != "default" || !runtime.Enabled || runtime.Count != 1 {
			t.Fatalf("unexpected resolved runtime: %+v", runtime)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ResolveRuntime returned error: %v", err)
	}
	if !validateCalled {
		t.Fatal("expected validate callback to run")
	}
	if resolved != defaults {
		t.Fatalf("expected defaults to be returned unchanged, got %+v", resolved)
	}
}

func TestResolveRuntimeMergesOverrides(t *testing.T) {
	defaults := resolveRuntimeTestRuntime{
		Name:    "default",
		Enabled: true,
		Count:   1,
	}
	overrides := resolveRuntimeTestRuntime{
		Name:  "override",
		Count: 9,
	}
	resolved, err := ResolveRuntime(defaults, nil, overrides)
	if err != nil {
		t.Fatalf("ResolveRuntime returned error: %v", err)
	}
	if resolved.Name != "override" {
		t.Fatalf("expected override name, got %q", resolved.Name)
	}
	if !resolved.Enabled {
		t.Fatal("expected non-zero default bool to be preserved")
	}
	if resolved.Count != 9 {
		t.Fatalf("expected override count 9, got %d", resolved.Count)
	}
}

func TestResolveRuntimeReturnsValidationError(t *testing.T) {
	defaults := resolveRuntimeTestRuntime{Name: "default"}
	expectedErr := errors.New("validation failed")
	resolved, err := ResolveRuntime(defaults, func(resolveRuntimeTestRuntime) error {
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected validation error %v, got %v", expectedErr, err)
	}
	if resolved.Name != defaults.Name {
		t.Fatalf("expected resolved runtime to be returned with defaults, got %+v", resolved)
	}
}
