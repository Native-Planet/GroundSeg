package auth

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"groundseg/auth/lifecycle"
)

func Initialize() {
	InitializeWithContext(context.Background())
}

func InitializeWithContext(ctx context.Context) {
	if err := Start(ctx); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to initialize auth lifecycle: %v", err))
	}
}

func Start(ctx context.Context) error {
	return lifecycle.Start(ctx)
}

func Stop() {
	lifecycle.Stop()
}
