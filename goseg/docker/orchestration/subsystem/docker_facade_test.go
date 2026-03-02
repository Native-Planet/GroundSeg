package subsystem

import (
	"context"
	"testing"
)

func TestDockerSubsystemFacadeIsCancelable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := StartDockerSubsystem(ctx); err != nil {
		t.Fatalf("expected StartDockerSubsystem to return nil on canceled context, got: %v", err)
	}
}
