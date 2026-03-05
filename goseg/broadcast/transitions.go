package broadcast

import (
	"errors"
	"fmt"

	"groundseg/structs"
	"groundseg/transition"
)

type BroadcastStateRuntime = broadcastStateRuntime

// BroadcastTransition describes a single, typed transition applied to broadcast state.
// Mutations must be scoped to the transition contract implemented by the transition type.
type BroadcastTransition interface {
	Apply(*structs.AuthBroadcast) error
}

// ApplyBroadcastTransition applies a typed transition to the current broadcast state.
// If `notify` is true, it refreshes connected websocket clients afterward.
func ApplyBroadcastTransition(notify bool, operation BroadcastTransition, runtime *BroadcastStateRuntime) error {
	if operation == nil {
		return nil
	}
	if runtime == nil {
		return fmt.Errorf("broadcast runtime is required for transition")
	}
	current := runtime.GetState()
	if err := operation.Apply(&current); err != nil {
		return fmt.Errorf("apply broadcast transition %T: %w", operation, err)
	}
	runtime.UpdateBroadcast(current)
	if !notify {
		return nil
	}
	if err := runtime.BroadcastToClients(); err != nil {
		if errors.Is(err, ErrBroadcastLeakBackpressure) {
			return transition.HandleTransitionPublishError(
				fmt.Sprintf("publish broadcast transition %T", operation),
				err,
				transition.TransitionPolicyForCriticality(transition.TransitionPublishNonCritical),
			)
		}
		return err
	}
	return nil
}

// ApplyBroadcastMutation applies an inline mutation to the current broadcast state.
func ApplyBroadcastMutation(notify bool, mutate func(state *structs.AuthBroadcast), runtime *BroadcastStateRuntime) error {
	if mutate == nil {
		return nil
	}
	return ApplyBroadcastTransition(notify, broadcastMutationTransition{mutate: mutate}, runtime)
}

// SetStartramRunning updates the startram "running" state via a broadcast transition.
func SetStartramRunning(running bool) error {
	err := ApplyBroadcastTransition(true, startramRunningTransition{running: running}, DefaultBroadcastStateRuntime())
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrBroadcastLeakBackpressure) {
		return transition.HandleTransitionPublishError(
			"publish startram running broadcast update",
			err,
			transition.TransitionPolicyForCriticality(transition.TransitionPublishNonCritical),
		)
	}
	return err
}

type startramRunningTransition struct {
	running bool
}

func (transition startramRunningTransition) Apply(current *structs.AuthBroadcast) error {
	current.Profile.Startram.Info.Running = transition.running
	return nil
}

type broadcastMutationTransition struct {
	mutate func(*structs.AuthBroadcast)
}

func (transition broadcastMutationTransition) Apply(current *structs.AuthBroadcast) error {
	if transition.mutate == nil {
		return nil
	}
	transition.mutate(current)
	return nil
}
