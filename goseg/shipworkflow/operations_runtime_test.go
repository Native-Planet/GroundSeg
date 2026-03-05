package shipworkflow

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"groundseg/shipworkflow/adapters/lifecyclebridge"
)

func TestRunTransitionedOperationSuccessAndError(t *testing.T) {
	var events []string
	runtime := workflowRuntime{
		TransitionEmitter: workflowTransitionFn(func(patp, transitionType, event string) error {
			_ = patp
			_ = transitionType
			events = append(events, event)
			return nil
		}),
		Sleeper: workflowSleeperFn(func(time.Duration) {}),
	}

	err := RunTransitionedOperationWithRuntime(runtime, "~zod", "backup", "loading", "success", time.Second, func() error { return nil })
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !reflect.DeepEqual(events, []string{"loading", "success", ""}) {
		t.Fatalf("unexpected transition events on success: %v", events)
	}

	events = nil
	err = RunTransitionedOperationWithRuntime(runtime, "~zod", "backup", "loading", "success", 0, func() error {
		return errors.New("boom")
	})
	if err == nil {
		t.Fatalf("expected operation error")
	}
	if !reflect.DeepEqual(events, []string{"loading", "error", ""}) {
		t.Fatalf("unexpected transition events on error: %v", events)
	}
}

func TestPollWithTimeout(t *testing.T) {
	attempts := 0
	err := lifecyclebridge.PollWithTimeout(context.Background(), time.Millisecond, func() (bool, error) {
		attempts++
		return attempts >= 2, nil
	})
	if err != nil {
		t.Fatalf("expected successful poll, got %v", err)
	}
	if attempts < 2 {
		t.Fatalf("expected repeated polling attempts, got %d", attempts)
	}

	if err := lifecyclebridge.PollWithTimeout(context.Background(), time.Millisecond, func() (bool, error) {
		return false, errors.New("condition failed")
	}); err == nil {
		t.Fatalf("expected condition error")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	err = lifecyclebridge.PollWithTimeout(ctx, 10*time.Millisecond, func() (bool, error) { return false, nil })
	if err == nil {
		t.Fatalf("expected context timeout/cancel error")
	}
}

func TestPublishTransitionWithPolicy(t *testing.T) {
	var events []int
	runtime := workflowRuntime{
		TransitionEmitter: workflowTransitionFn(func(string, string, string) error { return nil }),
		Sleeper:           workflowSleeperFn(func(time.Duration) {}),
	}
	publish := func(value int) {
		events = append(events, value)
	}
	if err := publishTransition(runtime, publish, 1, 0, time.Second); err != nil {
		t.Fatalf("expected publish transition to succeed, got: %v", err)
	}
	if !reflect.DeepEqual(events, []int{1, 0}) {
		t.Fatalf("unexpected transition sequence: %v", events)
	}
}
