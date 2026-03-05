package routines

import (
	"context"
	"testing"
)

func TestSmartDiskCheckWithContextReturnsOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := SmartDiskCheckWithContext(ctx); err != nil {
		t.Fatalf("SmartDiskCheckWithContext returned error: %v", err)
	}
}

func TestDiskUsageWarningWithContextReturnsOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := DiskUsageWarningWithContext(ctx); err != nil {
		t.Fatalf("DiskUsageWarningWithContext returned error: %v", err)
	}
}
