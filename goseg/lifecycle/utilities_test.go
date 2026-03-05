package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPersistWithMutatorValidatesCallbacks(t *testing.T) {
	if err := PersistWithMutator[string]("id", nil, func(string, func(*string) error) error { return nil }); !errors.Is(err, ErrMutateFunctionRequired) {
		t.Fatalf("expected ErrMutateFunctionRequired, got %v", err)
	}

	if err := PersistWithMutator[string]("id", func(*string) error { return nil }, nil); !errors.Is(err, ErrPersistFunctionRequired) {
		t.Fatalf("expected ErrPersistFunctionRequired, got %v", err)
	}
}

func TestPersistWithMutatorInvokesPersistCallback(t *testing.T) {
	called := false
	err := PersistWithMutator[string]("~zod", func(value *string) error {
		*value = "updated"
		return nil
	}, func(id string, mutate func(*string) error) error {
		called = true
		if id != "~zod" {
			t.Fatalf("unexpected id: %s", id)
		}
		state := "initial"
		if err := mutate(&state); err != nil {
			return err
		}
		if state != "updated" {
			t.Fatalf("unexpected mutation result: %s", state)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("PersistWithMutator returned error: %v", err)
	}
	if !called {
		t.Fatal("expected persist callback to be called")
	}
}

func TestPollWithTimeout(t *testing.T) {
	if err := PollWithTimeout(context.Background(), 0, func() (bool, error) { return true, nil }); !errors.Is(err, ErrPollIntervalNonPositive) {
		t.Fatalf("expected ErrPollIntervalNonPositive, got %v", err)
	}

	attempts := 0
	err := PollWithTimeout(context.Background(), time.Millisecond, func() (bool, error) {
		attempts++
		return attempts >= 2, nil
	})
	if err != nil {
		t.Fatalf("PollWithTimeout returned error: %v", err)
	}
	if attempts < 2 {
		t.Fatalf("expected at least two attempts, got %d", attempts)
	}

	conditionErr := errors.New("condition failed")
	if err := PollWithTimeout(context.Background(), time.Millisecond, func() (bool, error) { return false, conditionErr }); !errors.Is(err, conditionErr) {
		t.Fatalf("expected condition error, got %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	if err := PollWithTimeout(ctx, 10*time.Millisecond, func() (bool, error) { return false, nil }); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}
