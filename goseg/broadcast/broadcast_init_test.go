package broadcast

import (
	"errors"
	"strings"
	"testing"
)

func TestInitializeAppliesRuntimeOptions(t *testing.T) {
	calls := []string{}

	err := Initialize(
		WithBroadcastBootstrap(func() error {
			calls = append(calls, "bootstrap")
			return nil
		}),
		WithBroadcastLoadStartramRegions(func() error {
			calls = append(calls, "regions")
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 initialization calls, got %d", len(calls))
	}
	if calls[0] != "bootstrap" || calls[1] != "regions" {
		t.Fatalf("expected calls in bootstrap->regions order, got %v", calls)
	}
}

func TestInitializeFailsWithInvalidRuntimeFns(t *testing.T) {
	targetErr := errors.New("bootstrap failed")
	err := Initialize(
		WithBroadcastBootstrap(func() error {
			return targetErr
		}),
		WithBroadcastLoadStartramRegions(func() error {
			return nil
		}),
	)
	if err == nil || !strings.Contains(err.Error(), targetErr.Error()) {
		t.Fatalf("expected bootstrap failure to propagate, got %v", err)
	}
}

func TestInitializeRejectsNilBootstrapFn(t *testing.T) {
	err := Initialize(
		WithBroadcastBootstrap(nil),
		WithBroadcastLoadStartramRegions(func() error { return nil }),
	)
	if err == nil || !strings.Contains(err.Error(), "bootstrap function is not configured") {
		t.Fatalf("expected missing bootstrap function error, got %v", err)
	}
}
