package broadcast

import (
	"time"

	"groundseg/structs"
)

// BroadcastTransition describes a single, typed transition applied to broadcast state.
// Mutations must be scoped to the transition contract implemented by the transition type.
type BroadcastTransition interface {
	Apply(*structs.AuthBroadcast) error
}

type BroadcastStore interface {
	GetState() structs.AuthBroadcast
	UpdateBroadcast(structs.AuthBroadcast)
	GetScheduledPack(string) time.Time
	UpdateScheduledPack(string, time.Time) error
	BroadcastToClients() error
}

// ApplyBroadcastTransition applies a typed transition to the current broadcast state.
// If `notify` is true, it refreshes connected websocket clients afterward.
func ApplyBroadcastTransition(notify bool, transition BroadcastTransition, runtime ...BroadcastStore) error {
	if transition == nil {
		return nil
	}
	resolvedRuntime := resolveBroadcastStateTransitionRuntime(runtime...)
	current := resolvedRuntime.GetState()
	if err := transition.Apply(&current); err != nil {
		return err
	}
	resolvedRuntime.UpdateBroadcast(current)
	if !notify {
		return nil
	}
	return resolvedRuntime.BroadcastToClients()
}

// ApplyBroadcastMutation applies an inline mutation to the current broadcast state.
func ApplyBroadcastMutation(notify bool, mutate func(state *structs.AuthBroadcast), runtime ...BroadcastStore) error {
	if mutate == nil {
		return nil
	}

	resolvedRuntime := resolveBroadcastStateTransitionRuntime(runtime...)
	current := resolvedRuntime.GetState()
	mutate(&current)
	resolvedRuntime.UpdateBroadcast(current)
	if !notify {
		return nil
	}
	return resolvedRuntime.BroadcastToClients()
}

// SetStartramRunning updates the startram "running" state via a broadcast transition.
func SetStartramRunning(running bool) error {
	return ApplyBroadcastTransition(true, startramRunningTransition{running: running})
}

type startramRunningTransition struct {
	running bool
}

func (transition startramRunningTransition) Apply(current *structs.AuthBroadcast) error {
	current.Profile.Startram.Info.Running = transition.running
	return nil
}

func resolveBroadcastStateTransitionRuntime(runtime ...BroadcastStore) BroadcastStore {
	if len(runtime) > 0 && runtime[0] != nil {
		return runtime[0]
	}
	return DefaultBroadcastStateRuntime()
}
