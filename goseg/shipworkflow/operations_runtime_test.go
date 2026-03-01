package shipworkflow

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"groundseg/docker"
	"groundseg/structs"
)

func resetWorkflowSeams() {
	publishUrbitTransitionForWorkflow = docker.PublishUrbitTransition
	sleepForWorkflow = time.Sleep
}

func TestRunTransitionedOperationSuccessAndError(t *testing.T) {
	t.Cleanup(resetWorkflowSeams)
	sleepForWorkflow = func(time.Duration) {}
	var events []string
	publishUrbitTransitionForWorkflow = func(trans structs.UrbitTransition) {
		events = append(events, trans.Event)
	}

	err := RunTransitionedOperation("~zod", "backup", "loading", "success", time.Second, func() error { return nil })
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !reflect.DeepEqual(events, []string{"loading", "success", ""}) {
		t.Fatalf("unexpected transition events on success: %v", events)
	}

	events = nil
	err = RunTransitionedOperation("~zod", "backup", "loading", "success", 0, func() error {
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
	t.Cleanup(resetWorkflowSeams)

	attempts := 0
	err := PollWithTimeout(context.Background(), time.Millisecond, func() (bool, error) {
		attempts++
		return attempts >= 2, nil
	})
	if err != nil {
		t.Fatalf("expected successful poll, got %v", err)
	}
	if attempts < 2 {
		t.Fatalf("expected repeated polling attempts, got %d", attempts)
	}

	if err := PollWithTimeout(context.Background(), time.Millisecond, func() (bool, error) {
		return false, errors.New("condition failed")
	}); err == nil {
		t.Fatalf("expected condition error")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	err = PollWithTimeout(ctx, 10*time.Millisecond, func() (bool, error) { return false, nil })
	if err == nil {
		t.Fatalf("expected context timeout/cancel error")
	}
}

func TestPublishTransitionWithPolicy(t *testing.T) {
	t.Cleanup(resetWorkflowSeams)
	sleepForWorkflow = func(time.Duration) {}

	var events []int
	publish := func(value int) {
		events = append(events, value)
	}
	PublishTransitionWithPolicy(publish, 1, 0, time.Second)
	if !reflect.DeepEqual(events, []int{1, 0}) {
		t.Fatalf("unexpected transition sequence: %v", events)
	}
}
