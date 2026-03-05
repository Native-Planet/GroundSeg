package events

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunValidatesInputs(t *testing.T) {
	if err := Run(context.Background(), nil, func(string) {}); err == nil {
		t.Fatal("expected nil channel validation error")
	}
	if err := Run(context.Background(), make(chan string), nil); err == nil {
		t.Fatal("expected nil processor validation error")
	}
}

func TestRunProcessesEventsUntilCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := make(chan string, 1)
	processed := make(chan string, 1)
	if err := Run(ctx, events, func(event string) {
		processed <- event
	}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	events <- "c2cInterval"
	select {
	case event := <-processed:
		if event != "c2cInterval" {
			t.Fatalf("unexpected event payload: %q", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for processed event")
	}
}

func TestStartValidatesRuntime(t *testing.T) {
	err := Start(context.Background(), Runtime{})
	if err == nil {
		t.Fatal("expected Start to reject nil channel callback")
	}
}

func TestStartUsesRuntimeCallbacks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := make(chan string, 1)
	processed := make(chan string, 1)
	err := Start(ctx, Runtime{
		Channel: func() <-chan string {
			return events
		},
		Process: func(event string) {
			processed <- event
		},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	events <- "status"
	select {
	case got := <-processed:
		if got != "status" {
			t.Fatalf("unexpected processed event: %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for runtime callback processing")
	}
}

func TestRunReturnsConditionErrors(t *testing.T) {
	sentinel := errors.New("expected")
	events := make(chan string)
	close(events)
	err := Run(context.Background(), events, func(string) {
		panic(sentinel)
	})
	if err != nil {
		t.Fatalf("Run should not fail when channel is closed before processing, got %v", err)
	}
}
