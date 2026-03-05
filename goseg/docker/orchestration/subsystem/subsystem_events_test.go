package subsystem

import (
	"context"
	"strings"
	"testing"
	"time"

	eventtypes "github.com/docker/docker/api/types/events"
)

func TestStartDockerHealthLoopsIsCancellableByContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	if err := StartDockerHealthLoops(ctx); err != nil {
		t.Fatalf("start docker health loops: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		StopDockerHealthLoops()
	})

	awaitDockerHealthLoopState(t, true)
	cancel()
	awaitDockerHealthLoopState(t, false)
}

func TestStartDockerHealthLoopsNoopOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := StartDockerHealthLoops(ctx); err != nil {
		t.Fatalf("start docker health loops: %v", err)
	}
	awaitDockerHealthLoopState(t, false)
}

func TestStartDockerHealthLoopsDeduplicatesStartCalls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		StopDockerHealthLoops()
	})

	if err := StartDockerHealthLoops(ctx); err != nil {
		t.Fatalf("first start: %v", err)
	}
	if err := StartDockerHealthLoops(ctx); err != nil {
		t.Fatalf("second start: %v", err)
	}
	StopDockerHealthLoops()
	awaitDockerHealthLoopState(t, false)
}

type dockerEventStreamClientStub struct {
	messages <-chan eventtypes.Message
	errs     <-chan error
}

func (stub dockerEventStreamClientStub) Events(context.Context, eventtypes.ListOptions) (<-chan eventtypes.Message, <-chan error) {
	return stub.messages, stub.errs
}

func (dockerEventStreamClientStub) Close() error {
	return nil
}

func TestDockerListenerReturnsErrorWhenMessageStreamCloses(t *testing.T) {
	messages := make(chan eventtypes.Message)
	errs := make(chan error)
	close(messages)

	originalFactory := newDockerEventStreamClient
	newDockerEventStreamClient = func() (dockerEventStreamClient, error) {
		return dockerEventStreamClientStub{messages: messages, errs: errs}, nil
	}
	t.Cleanup(func() {
		newDockerEventStreamClient = originalFactory
	})

	err := dockerListenerWithRuntime(newDockerSubsystemRuntime(), context.Background())
	if err == nil {
		t.Fatal("expected message stream closure to return an error")
	}
	if !strings.Contains(err.Error(), "docker event stream closed") {
		t.Fatalf("unexpected message stream closure error: %v", err)
	}
}

func TestDockerListenerReturnsErrorWhenErrorStreamCloses(t *testing.T) {
	messages := make(chan eventtypes.Message)
	errs := make(chan error)
	close(errs)

	originalFactory := newDockerEventStreamClient
	newDockerEventStreamClient = func() (dockerEventStreamClient, error) {
		return dockerEventStreamClientStub{messages: messages, errs: errs}, nil
	}
	t.Cleanup(func() {
		newDockerEventStreamClient = originalFactory
	})

	err := dockerListenerWithRuntime(newDockerSubsystemRuntime(), context.Background())
	if err == nil {
		t.Fatal("expected error stream closure to return an error")
	}
	if !strings.Contains(err.Error(), "docker event error stream closed") {
		t.Fatalf("unexpected error stream closure error: %v", err)
	}
}

func awaitDockerHealthLoopState(t *testing.T, wantRunning bool) {
	t.Helper()

	const timeout = 2 * time.Second
	deadline := time.Now().Add(timeout)
	for {
		running := defaultDockerSubsystemRuntime.isHealthLoopRunning()
		if running == wantRunning {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected docker health loop running=%v after waiting %v; got %v", wantRunning, timeout, running)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
