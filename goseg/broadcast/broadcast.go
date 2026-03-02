package broadcast

import (
	"fmt"
	"groundseg/auth"
	"groundseg/broadcast/collectors"
)

type BroadcastRuntimeOption func(*broadcastRuntime)

type broadcastRuntime struct {
	bootstrapFn           func() error
	loadStartramRegionsFn func() error
}

func Initialize(options ...BroadcastRuntimeOption) error {
	runtime := defaultBroadcastRuntime()
	for _, option := range options {
		if option != nil {
			option(&runtime)
		}
	}
	return InitializeWithRuntime(runtime)
}

func InitializeWithRuntime(runtime broadcastRuntime) error {
	if auth.GetClientManager() == nil {
		return fmt.Errorf("config subsystem is not initialized")
	}
	if runtime.bootstrapFn == nil {
		return fmt.Errorf("broadcast bootstrap function is not configured")
	}
	if runtime.loadStartramRegionsFn == nil {
		return fmt.Errorf("broadcast startram regions loader is not configured")
	}
	if err := runtime.bootstrapFn(); err != nil {
		return fmt.Errorf("unable to initialize broadcast state: %w", err)
	}
	if err := runtime.loadStartramRegionsFn(); err != nil {
		return fmt.Errorf("unable to load StarTram regions: %w", err)
	}
	return nil
}

func WithBroadcastBootstrap(runtimeFn func() error) BroadcastRuntimeOption {
	return func(rt *broadcastRuntime) {
		rt.bootstrapFn = runtimeFn
	}
}

func WithBroadcastLoadStartramRegions(runtimeFn func() error) BroadcastRuntimeOption {
	return func(rt *broadcastRuntime) {
		rt.loadStartramRegionsFn = runtimeFn
	}
}

func defaultBroadcastRuntime() broadcastRuntime {
	return broadcastRuntime{
		bootstrapFn:           bootstrapBroadcastState,
		loadStartramRegionsFn: LoadStartramRegions,
	}
}

// GetStartramServices refreshes StarTram service metadata from the configured endpoints.
//
// Kept in the top-level broadcast package to preserve the pre-existing contract used by
// orchestration workflows.
func GetStartramServices() error {
	return collectors.GetStartramServices()
}
