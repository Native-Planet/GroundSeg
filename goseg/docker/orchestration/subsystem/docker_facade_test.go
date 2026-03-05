package subsystem

import (
	"context"
	"testing"
	"time"
)

func TestDockerSubsystemFacadeIsCancelable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := StartDockerSubsystem(ctx); err != nil {
		t.Fatalf("expected StartDockerSubsystem to return nil on canceled context, got: %v", err)
	}
}

func TestDockerSubsystemHandleCompletesForCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	handle, err := StartDockerSubsystemWithHandle(ctx)
	if err != nil {
		t.Fatalf("expected StartDockerSubsystemWithHandle to return nil on canceled context, got: %v", err)
	}
	if handle == nil {
		t.Fatal("expected async handle")
	}
	select {
	case <-handle.Done():
	case <-time.After(time.Second):
		t.Fatal("expected canceled docker subsystem handle to complete")
	}
}
