package events

import (
	"context"
	"testing"

	"groundseg/structs"
)

func TestNewRuntimeBoundBrokerPinsSpecificRuntime(t *testing.T) {
	originalDefault := DefaultEventRuntime()
	t.Cleanup(func() {
		SetDefaultEventRuntime(originalDefault)
	})

	runtimeA := NewEventRuntimeWithBuffer(2)
	runtimeB := NewEventRuntimeWithBuffer(2)
	broker := NewRuntimeBoundBroker(runtimeA)
	SetDefaultEventRuntime(runtimeB)

	if err := broker.PublishImportShipTransition(context.Background(), structs.UploadTransition{Type: "status", Event: "a"}); err != nil {
		t.Fatalf("expected publish on runtime-bound broker to succeed: %v", err)
	}

	select {
	case transition := <-runtimeA.ImportShipTransitions():
		if transition.Event != "a" {
			t.Fatalf("unexpected transition payload from runtime A: %#v", transition)
		}
	default:
		t.Fatal("expected event to be published to runtime A")
	}

	select {
	case transition := <-runtimeB.ImportShipTransitions():
		t.Fatalf("did not expect runtime B to receive event from runtime-bound broker, got %#v", transition)
	default:
	}
}

func TestNewRuntimeBoundBrokerNormalizesUndefinedRuntime(t *testing.T) {
	broker := NewRuntimeBoundBroker(EventRuntime{})
	if err := broker.PublishSystemTransition(context.Background(), structs.SystemTransition{Type: "ok"}); err != nil {
		t.Fatalf("expected normalized runtime-bound broker to publish: %v", err)
	}
}
