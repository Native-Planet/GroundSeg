package events

import (
	"context"
	"testing"

	"groundseg/structs"
)

func TestEventRuntimePublishAndSubscribe(t *testing.T) {
	runtime := NewEventRuntimeWithBuffer(2)
	ctx := context.Background()

	if err := runtime.PublishUrbitTransition(ctx, structs.UrbitTransition{Type: "pack"}); err != nil {
		t.Fatalf("PublishUrbitTransition returned error: %v", err)
	}
	if got := <-runtime.UrbitTransitions(); got.Type != "pack" {
		t.Fatalf("unexpected urbit transition payload: %#v", got)
	}

	if err := runtime.PublishSystemTransition(ctx, structs.SystemTransition{Type: "swap"}); err != nil {
		t.Fatalf("PublishSystemTransition returned error: %v", err)
	}
	if got := <-runtime.SystemTransitions(); got.Type != "swap" {
		t.Fatalf("unexpected system transition payload: %#v", got)
	}

	if err := runtime.PublishNewShipTransition(ctx, structs.NewShipTransition{Type: "error"}); err != nil {
		t.Fatalf("PublishNewShipTransition returned error: %v", err)
	}
	if got := <-runtime.NewShipTransitions(); got.Type != "error" {
		t.Fatalf("unexpected new ship transition payload: %#v", got)
	}

	if err := runtime.PublishImportShipTransition(ctx, structs.UploadTransition{Type: "status"}); err != nil {
		t.Fatalf("PublishImportShipTransition returned error: %v", err)
	}
	if got := <-runtime.ImportShipTransitions(); got.Type != "status" {
		t.Fatalf("unexpected import transition payload: %#v", got)
	}
}

func TestDefaultEventBrokerTracksCurrentDefaultRuntime(t *testing.T) {
	original := DefaultEventRuntime()
	t.Cleanup(func() {
		SetDefaultEventRuntime(original)
	})

	initial := NewEventRuntimeWithBuffer(1)
	replacement := NewEventRuntimeWithBuffer(1)
	SetDefaultEventRuntime(initial)

	broker := NewGlobalDefaultBroker()
	if err := broker.PublishSystemTransition(context.Background(), structs.SystemTransition{Type: "first"}); err != nil {
		t.Fatalf("publish via default broker (initial runtime): %v", err)
	}
	if got := <-initial.SystemTransitions(); got.Type != "first" {
		t.Fatalf("unexpected transition on initial runtime: %#v", got)
	}

	SetDefaultEventRuntime(replacement)
	if err := broker.PublishSystemTransition(context.Background(), structs.SystemTransition{Type: "second"}); err != nil {
		t.Fatalf("publish via default broker (replacement runtime): %v", err)
	}
	if got := <-replacement.SystemTransitions(); got.Type != "second" {
		t.Fatalf("unexpected transition on replacement runtime: %#v", got)
	}
}

func TestIsolatedRuntimeBrokerDoesNotTrackDefaultRuntimeSwaps(t *testing.T) {
	original := DefaultEventRuntime()
	t.Cleanup(func() {
		SetDefaultEventRuntime(original)
	})

	initialDefault := NewEventRuntimeWithBuffer(1)
	replacementDefault := NewEventRuntimeWithBuffer(1)
	SetDefaultEventRuntime(initialDefault)

	broker := NewIsolatedRuntimeBroker(1)
	if err := broker.PublishSystemTransition(context.Background(), structs.SystemTransition{Type: "isolated-first"}); err != nil {
		t.Fatalf("publish via isolated broker before default swap: %v", err)
	}
	if got := <-broker.SystemTransitions(); got.Type != "isolated-first" {
		t.Fatalf("unexpected isolated transition payload: %#v", got)
	}

	SetDefaultEventRuntime(replacementDefault)
	if err := broker.PublishSystemTransition(context.Background(), structs.SystemTransition{Type: "isolated-second"}); err != nil {
		t.Fatalf("publish via isolated broker after default swap: %v", err)
	}
	if got := <-broker.SystemTransitions(); got.Type != "isolated-second" {
		t.Fatalf("unexpected isolated transition payload after swap: %#v", got)
	}

	select {
	case <-initialDefault.SystemTransitions():
		t.Fatal("isolated broker should not publish into initial default runtime")
	default:
	}
	select {
	case <-replacementDefault.SystemTransitions():
		t.Fatal("isolated broker should not publish into replacement default runtime")
	default:
	}
}
