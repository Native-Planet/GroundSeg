package broadcast

import "groundseg/structs"

func (runtime *broadcastStateRuntime) GetState() structs.AuthBroadcast {
	if runtime == nil {
		return structs.AuthBroadcast{}
	}
	runtime.RLock()
	defer runtime.RUnlock()
	return cloneBroadcastState(runtime.broadcastState)
}

func (runtime *broadcastStateRuntime) UpdateBroadcast(next structs.AuthBroadcast) error {
	if runtime == nil {
		return ErrBroadcastRuntimeRequired
	}
	runtime.Lock()
	defer runtime.Unlock()
	runtime.broadcastState = cloneBroadcastState(next)
	return nil
}

func (runtime *broadcastStateRuntime) AddSystemTransitionError(message string) error {
	if runtime == nil || message == "" {
		if runtime == nil {
			return ErrBroadcastRuntimeRequired
		}
		return nil
	}
	runtime.Lock()
	defer runtime.Unlock()
	next := append([]string{message}, runtime.broadcastState.System.Transition.Error...)
	if len(next) > maxSystemTransitionErrors {
		next = next[:maxSystemTransitionErrors]
	}
	runtime.broadcastState.System.Transition.Error = next
	return nil
}
