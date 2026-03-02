package routines

import (
	"context"

	"groundseg/routines/healthcheck"
)

func SmartDiskCheck() {
	healthcheck.SmartDiskCheck()
}

func SmartDiskCheckWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return healthcheck.SmartDiskCheckWithContext(ctx)
}

func DiskUsageWarning() {
	healthcheck.DiskUsageWarning()
}

func DiskUsageWarningWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return healthcheck.DiskUsageWarningWithContext(ctx)
}
