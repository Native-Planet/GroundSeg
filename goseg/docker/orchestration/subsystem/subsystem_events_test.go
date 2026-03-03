package subsystem

import (
	"context"
	"testing"
	"time"
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

func awaitDockerHealthLoopState(t *testing.T, wantRunning bool) {
	t.Helper()

	const timeout = 2 * time.Second
	deadline := time.Now().Add(timeout)
	for {
		dockerHealthLoopMu.Lock()
		running := healthLoopRunning
		dockerHealthLoopMu.Unlock()
		if running == wantRunning {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected docker health loop running=%v after waiting %v; got %v", wantRunning, timeout, running)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
