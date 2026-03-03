package config

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"groundseg/system"
)

// StartConfEventLoop starts a cancelable event listener for config-related
// asynchronous signals (for example, C2C lifecycle events from system packages).
func StartConfEventLoop(ctx context.Context, confEvents <-chan string) error {
	if confEvents == nil {
		return errors.New("config event channel is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-confEvents:
				if !ok {
					return
				}
				processConfEvent(event)
			}
		}
	}()
	return nil
}

// processConfEvent handles config events emitted from other subsystems.
//
// This is extracted for testability and avoids repeating event handling logic.
func processConfEvent(event string) {
	if event != "c2cInterval" {
		return
	}
	conf := Conf()
	if conf.C2cInterval == 0 {
		if err := UpdateConfTyped(WithC2cInterval(600)); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't set C2C interval: %v", err))
		}
	}
}

// ConfChannel starts the config event loop from the legacy package channel.
func ConfChannel() {
	// Backward-compatible entrypoint for existing code paths.
	if err := StartConfEventLoop(context.Background(), system.ConfChannel()); err != nil {
		zap.L().Error(fmt.Sprintf("Failed to start config event loop: %v", err))
	}
}
