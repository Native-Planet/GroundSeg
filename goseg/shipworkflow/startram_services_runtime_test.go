package shipworkflow

import (
	"errors"
	"strings"
	"testing"
)

func TestRunStartramServicesWithRuntimeDelegatesToRuntime(t *testing.T) {
	t.Parallel()

	calls := 0
	runtime := defaultStartramRuntime()
	runtime.GetStartramServicesFn = func() error {
		calls++
		return nil
	}

	if err := runStartramServicesWithRuntime(runtime); err != nil {
		t.Fatalf("runStartramServicesWithRuntime returned error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected GetStartramServicesFn to be called once, got %d", calls)
	}
}

func TestRunStartramRegionsWithRuntimeDelegatesToRuntime(t *testing.T) {
	t.Parallel()

	runtime := defaultStartramRuntime()
	runtime.LoadStartramRegionsFn = func() error {
		return errors.New("region refresh failed")
	}

	err := runStartramRegionsWithRuntime(runtime)
	if err == nil {
		t.Fatal("expected runtime region error to propagate")
	}
	if !strings.Contains(err.Error(), "region refresh failed") {
		t.Fatalf("unexpected region error: %v", err)
	}
}
