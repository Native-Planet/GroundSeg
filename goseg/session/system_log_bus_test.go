package session

import (
	"errors"
	"testing"
)

func TestNewSystemLogMessageBusUsesMinimumBuffer(t *testing.T) {
	bus := newSystemLogMessageBus(0)
	if bus == nil {
		t.Fatal("expected message bus instance")
	}
	if bus.channel == nil {
		t.Fatal("expected message channel to be initialized")
	}

	select {
	case bus.channel <- []byte("first"):
	default:
		t.Fatal("expected at least one buffered slot")
	}
}

func TestSystemLogMessageBusMessagesHandlesNilReceiver(t *testing.T) {
	var bus *systemLogMessageBus
	if messages := bus.Messages(); messages != nil {
		t.Fatalf("expected nil message channel for nil bus, got %v", messages)
	}
}

func TestSystemLogMessageBusPublishNoopsOnNilReceiver(t *testing.T) {
	var bus *systemLogMessageBus
	if err := bus.Publish([]byte("ignored")); !errors.Is(err, ErrSystemLogBusNotDefined) {
		t.Fatalf("expected nil bus publish error %v, got %v", ErrSystemLogBusNotDefined, err)
	}
}

func TestSystemLogMessageBusPublishWritesPayload(t *testing.T) {
	bus := newSystemLogMessageBus(1)
	payload := []byte("system-log-entry")
	if err := bus.Publish(payload); err != nil {
		t.Fatalf("publish payload: %v", err)
	}

	select {
	case got := <-bus.Messages():
		if string(got) != string(payload) {
			t.Fatalf("unexpected payload: got %q want %q", string(got), string(payload))
		}
	default:
		t.Fatal("expected published payload to be readable from message channel")
	}
}

func TestSystemLogMessageBusPublishReturnsFullError(t *testing.T) {
	bus := newSystemLogMessageBus(1)
	if err := bus.Publish([]byte("first")); err != nil {
		t.Fatalf("publish first payload: %v", err)
	}
	if err := bus.Publish([]byte("second")); !errors.Is(err, ErrSystemLogBusFull) {
		t.Fatalf("expected full bus error %v, got %v", ErrSystemLogBusFull, err)
	}
}
