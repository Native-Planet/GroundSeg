package broadcast

import "groundseg/structs"

// BroadcastTransition describes a single, typed transition applied to broadcast state.
// Mutations must be scoped to the transition contract implemented by the transition type.
type BroadcastTransition interface {
	Apply(*structs.AuthBroadcast) error
}

// ApplyBroadcastTransition applies a typed transition to the current broadcast state.
// If `notify` is true, it refreshes connected websocket clients afterward.
func ApplyBroadcastTransition(notify bool, transition BroadcastTransition) error {
	if transition == nil {
		return nil
	}
	current := GetState()
	if err := transition.Apply(&current); err != nil {
		return err
	}
	UpdateBroadcast(current)
	if !notify {
		return nil
	}
	return BroadcastToClients()
}

// ApplyBroadcastMutation applies an inline mutation to the current broadcast state.
func ApplyBroadcastMutation(notify bool, mutate func(state *structs.AuthBroadcast)) error {
	if mutate == nil {
		return nil
	}

	current := GetState()
	mutate(&current)
	UpdateBroadcast(current)
	if !notify {
		return nil
	}
	return BroadcastToClients()
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
