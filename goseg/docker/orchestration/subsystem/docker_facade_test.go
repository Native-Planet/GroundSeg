package subsystem

import (
	"context"
	"testing"
)

func TestDockerSubsystemFacadeIsCancelable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	StartDockerSubsystem(ctx)
}
