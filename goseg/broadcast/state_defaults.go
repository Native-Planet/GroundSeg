package broadcast

import "sync"

var (
	defaultBroadcastStateRuntimeMu sync.RWMutex
	defaultBroadcastStateRuntime   = NewBroadcastStateRuntime()
)

// DefaultBroadcastStateRuntime returns the shared process-wide broadcast runtime for bootstrap
// code and callers that do not yet inject an explicit runtime.
func DefaultBroadcastStateRuntime() *broadcastStateRuntime {
	defaultBroadcastStateRuntimeMu.RLock()
	defer defaultBroadcastStateRuntimeMu.RUnlock()
	return defaultBroadcastStateRuntime
}

func SetDefaultBroadcastStateRuntime(runtime *broadcastStateRuntime) *broadcastStateRuntime {
	if runtime == nil {
		runtime = NewBroadcastStateRuntime()
	}
	defaultBroadcastStateRuntimeMu.Lock()
	defaultBroadcastStateRuntime = runtime
	defaultBroadcastStateRuntimeMu.Unlock()
	return runtime
}

func ResetDefaultBroadcastStateRuntime() *broadcastStateRuntime {
	return SetDefaultBroadcastStateRuntime(NewBroadcastStateRuntime())
}
