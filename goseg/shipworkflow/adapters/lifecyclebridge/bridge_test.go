package lifecyclebridge

import (
	"context"
	"errors"
	"testing"
	"time"

	"groundseg/structs"
)

func TestPersistUrbitConfigRejectsNilMutate(t *testing.T) {
	err := PersistUrbitConfig("~zod", nil, func(string, func(*structs.UrbitDocker) error) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected nil mutate callback to return an error")
	}
	if !errors.Is(err, ErrMutateFunctionRequired) {
		t.Fatalf("unexpected mutate error: %v", err)
	}
}

func TestPersistUrbitConfigRejectsNilPersistFn(t *testing.T) {
	err := PersistUrbitConfig("~zod", func(*structs.UrbitDocker) error { return nil }, nil)
	if err == nil {
		t.Fatal("expected nil persist callback to return an error")
	}
	if !errors.Is(err, ErrPersistFunctionRequired) {
		t.Fatalf("unexpected persist error: %v", err)
	}
}

func TestPersistUrbitConfigCallsPersistFn(t *testing.T) {
	called := false
	err := PersistUrbitConfig("~zod", func(*structs.UrbitDocker) error { return nil }, func(patp string, mutate func(*structs.UrbitDocker) error) error {
		called = true
		if patp != "~zod" {
			t.Fatalf("unexpected patp passed to persist callback: %s", patp)
		}
		return mutate(&structs.UrbitDocker{})
	})
	if err != nil {
		t.Fatalf("expected persist callback to succeed: %v", err)
	}
	if !called {
		t.Fatal("expected persist callback to be invoked")
	}
}

func TestPollWithTimeoutRejectsNonPositiveInterval(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conditionCalled := false
	err := PollWithTimeout(ctx, 0, func() (bool, error) {
		conditionCalled = true
		return true, nil
	})
	if err == nil {
		t.Fatal("expected non-positive interval to return an error")
	}
	if !errors.Is(err, ErrPollIntervalNonPositive) {
		t.Fatalf("unexpected interval validation error: %v", err)
	}
	if conditionCalled {
		t.Fatal("expected condition callback not to run when interval is invalid")
	}
}
