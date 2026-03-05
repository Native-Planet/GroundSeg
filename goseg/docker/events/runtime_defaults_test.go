package events

import (
	"context"
	"testing"

	"groundseg/structs"
)

func TestSetDefaultEventRuntimeUsesProvidedRuntime(t *testing.T) {
	original := DefaultEventRuntime()
	t.Cleanup(func() {
		SetDefaultEventRuntime(original)
	})

	custom := NewEventRuntimeWithBuffer(2)
	SetDefaultEventRuntime(custom)

	got := DefaultEventRuntime()
	if got != custom {
		t.Fatal("expected default event runtime to use provided runtime")
	}
	if err := got.PublishSystemTransition(context.Background(), structs.SystemTransition{Type: "test"}); err != nil {
		t.Fatalf("expected provided runtime publish to succeed: %v", err)
	}
	select {
	case transition := <-got.SystemTransitions():
		if transition.Type != "test" {
			t.Fatalf("unexpected transition payload: %#v", transition)
		}
	default:
		t.Fatal("expected transition published to provided runtime channel")
	}
}

func TestSetDefaultEventRuntimeRejectsUndefinedBuses(t *testing.T) {
	original := DefaultEventRuntime()
	t.Cleanup(func() {
		SetDefaultEventRuntime(original)
	})

	SetDefaultEventRuntime(EventRuntime{})
	got := DefaultEventRuntime()
	if got == (EventRuntime{}) {
		t.Fatal("expected undefined runtime to be replaced with a fresh default")
	}
	if err := got.PublishUrbitTransition(context.Background(), structs.UrbitTransition{Type: "ok"}); err != nil {
		t.Fatalf("expected publish against replacement runtime to succeed: %v", err)
	}
}

func TestResetDefaultEventRuntimeReplacesCurrentDefault(t *testing.T) {
	original := DefaultEventRuntime()
	t.Cleanup(func() {
		SetDefaultEventRuntime(original)
	})

	custom := NewEventRuntimeWithBuffer(1)
	SetDefaultEventRuntime(custom)
	reset := ResetDefaultEventRuntime()
	if reset == custom {
		t.Fatal("expected reset to allocate a fresh runtime")
	}
	if DefaultEventRuntime() != reset {
		t.Fatal("expected reset runtime to be installed as default")
	}
}
