package events

import "sync"

var (
	defaultEventRuntimeMu sync.RWMutex
	defaultEventRuntime   = NewEventRuntime()
)

func DefaultEventRuntime() EventRuntime {
	defaultEventRuntimeMu.RLock()
	defer defaultEventRuntimeMu.RUnlock()
	return defaultEventRuntime
}

func SetDefaultEventRuntime(runtime EventRuntime) {
	defaultEventRuntimeMu.Lock()
	defer defaultEventRuntimeMu.Unlock()
	if !runtime.urbitTransitionBus.defined() ||
		!runtime.systemTransitionBus.defined() ||
		!runtime.newShipTransitionBus.defined() ||
		!runtime.importShipTransitionBus.defined() {
		runtime = NewEventRuntime()
	}
	defaultEventRuntime = runtime
}

func ResetDefaultEventRuntime() EventRuntime {
	runtime := NewEventRuntime()
	SetDefaultEventRuntime(runtime)
	return runtime
}
