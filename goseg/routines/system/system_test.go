package system

import (
	"context"
	"testing"
	"time"
)

func resetSystemRuntimeSeamsForTest(t *testing.T) {
	t.Helper()
	origWaitIntervals := waitIntervalsFn
	origRunRemote := runRemoteBackupFn
	origRunLocal := runLocalBackupFn
	origRunChop := runChopPassFn
	t.Cleanup(func() {
		waitIntervalsFn = origWaitIntervals
		runRemoteBackupFn = origRunRemote
		runLocalBackupFn = origRunLocal
		runChopPassFn = origRunChop
	})
}

func TestStartBackupRoutinesWithContextReturnsImmediately(t *testing.T) {
	resetSystemRuntimeSeamsForTest(t)
	waitIntervalsFn = func() (time.Duration, time.Duration) { return time.Hour, time.Hour }
	runRemoteBackupFn = func() error { return nil }
	runLocalBackupFn = func() error { return nil }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := time.Now()
	if err := StartBackupRoutinesWithContext(ctx); err != nil {
		t.Fatalf("StartBackupRoutinesWithContext returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("expected StartBackupRoutinesWithContext to be non-blocking, took %v", elapsed)
	}
}

func TestStartBackupRoutinesWithContextHandleCompletesOnCancel(t *testing.T) {
	resetSystemRuntimeSeamsForTest(t)
	waitIntervalsFn = func() (time.Duration, time.Duration) { return time.Hour, time.Hour }
	runRemoteBackupFn = func() error { return nil }
	runLocalBackupFn = func() error { return nil }

	ctx, cancel := context.WithCancel(context.Background())
	handle, err := StartBackupRoutinesWithContextHandle(ctx)
	if err != nil {
		t.Fatalf("StartBackupRoutinesWithContextHandle returned error: %v", err)
	}
	if handle == nil {
		t.Fatal("expected async handle")
	}
	cancel()
	select {
	case <-handle.Done():
	case <-time.After(time.Second):
		t.Fatal("expected backup handle to complete after cancellation")
	}
}

func TestRunBackupRoutinesWithContextBlocksUntilCancel(t *testing.T) {
	resetSystemRuntimeSeamsForTest(t)
	waitIntervalsFn = func() (time.Duration, time.Duration) { return time.Hour, time.Hour }
	runRemoteBackupFn = func() error { return nil }
	runLocalBackupFn = func() error { return nil }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{}, 1)
	go func() {
		_ = RunBackupRoutinesWithContext(ctx)
		done <- struct{}{}
	}()
	select {
	case <-done:
		t.Fatal("expected RunBackupRoutinesWithContext to block before cancellation")
	case <-time.After(30 * time.Millisecond):
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunBackupRoutinesWithContext did not return after cancellation")
	}
}

func TestStartChopRoutinesWithContextReturnsImmediately(t *testing.T) {
	resetSystemRuntimeSeamsForTest(t)
	runChopPassFn = func() error { return nil }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := time.Now()
	if err := StartChopRoutinesWithContext(ctx); err != nil {
		t.Fatalf("StartChopRoutinesWithContext returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("expected StartChopRoutinesWithContext to be non-blocking, took %v", elapsed)
	}
}

func TestStartChopRoutinesWithContextHandleCompletesOnCancel(t *testing.T) {
	resetSystemRuntimeSeamsForTest(t)
	runChopPassFn = func() error { return nil }

	ctx, cancel := context.WithCancel(context.Background())
	handle, err := StartChopRoutinesWithContextHandle(ctx)
	if err != nil {
		t.Fatalf("StartChopRoutinesWithContextHandle returned error: %v", err)
	}
	if handle == nil {
		t.Fatal("expected async handle")
	}
	cancel()
	select {
	case <-handle.Done():
	case <-time.After(time.Second):
		t.Fatal("expected chop handle to complete after cancellation")
	}
}

func TestRunChopRoutinesWithContextBlocksUntilCancel(t *testing.T) {
	resetSystemRuntimeSeamsForTest(t)
	runChopPassFn = func() error { return nil }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{}, 1)
	go func() {
		_ = RunChopRoutinesWithContext(ctx)
		done <- struct{}{}
	}()
	select {
	case <-done:
		t.Fatal("expected RunChopRoutinesWithContext to block before cancellation")
	case <-time.After(30 * time.Millisecond):
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunChopRoutinesWithContext did not return after cancellation")
	}
}
