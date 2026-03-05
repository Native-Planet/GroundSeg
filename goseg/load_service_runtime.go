package main

import (
	"fmt"

	"go.uber.org/zap"
)

func loadService(loadFn func() error, failureMessage string) {
	if loadFn == nil {
		zap.L().Warn("Startup load function is not configured")
		return
	}
	if err := loadFn(); err != nil {
		zap.L().Error(fmt.Sprintf("%s: %v", failureMessage, err))
	}
}
