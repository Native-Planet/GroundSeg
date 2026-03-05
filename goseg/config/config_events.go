package config

import (
	"context"
	"errors"
	"fmt"
	configevents "groundseg/config/events"
	"sync"

	"go.uber.org/zap"
)

type configEventRuntime struct {
	channelFn func() <-chan string
	processFn func(string)
}

var (
	configEventSourceMu sync.RWMutex
	configEventSourceFn func() <-chan string
)

func RegisterConfigEventSource(channelFn func() <-chan string) {
	configEventSourceMu.Lock()
	defer configEventSourceMu.Unlock()
	configEventSourceFn = channelFn
}

func registeredConfigEventSource() func() <-chan string {
	configEventSourceMu.RLock()
	defer configEventSourceMu.RUnlock()
	return configEventSourceFn
}

func defaultConfigEventRuntime() configEventRuntime {
	return configEventRuntime{
		channelFn: registeredConfigEventSource(),
		processFn: processConfigEvent,
	}
}

// StartConfigEventLoop starts a cancelable event listener for config-related
// asynchronous signals (for example, C2C lifecycle events from system packages).
func StartConfigEventLoop(ctx context.Context, configEvents <-chan string) error {
	return configevents.Run(ctx, configEvents, processConfigEvent)
}

func StartConfigEventLoopWithRuntime(ctx context.Context, runtime configEventRuntime) error {
	if runtime.processFn == nil {
		runtime.processFn = processConfigEvent
	}
	return configevents.Start(ctx, configevents.Runtime{
		Channel: runtime.channelFn,
		Process: runtime.processFn,
	})
}

// StartConfigEventLoopFromSystemChannel starts the config event loop on the system event channel.
func StartConfigEventLoopFromSystemChannel(ctx context.Context) error {
	runtime := defaultConfigEventRuntime()
	if runtime.channelFn == nil {
		return errors.New("config event source callback is not registered")
	}
	return StartConfigEventLoopWithRuntime(ctx, runtime)
}

// processConfigEvent handles config events emitted from other subsystems.
//
// This is extracted for testability and avoids repeating event handling logic.
func processConfigEvent(event string) {
	if event != "c2cInterval" {
		return
	}
	configSnapshot := Config()
	if configSnapshot.Connectivity.C2CInterval == 0 {
		if err := UpdateConfigTyped(WithC2CInterval(600)); err != nil {
			zap.L().Error(fmt.Sprintf("couldn't set C2C interval: %v", err))
		}
	}
}
