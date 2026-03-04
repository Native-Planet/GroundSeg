package rectify

import (
	"context"
	"fmt"

	"groundseg/broadcast"
	"groundseg/transition"
)

func applyTransitionUpdate(context string, transitionCommand broadcast.BroadcastTransition, publishPolicy transition.TransitionPublishPolicy, broadcastRuntime ...broadcast.BroadcastStore) error {
	if transitionCommand == nil {
		return nil
	}
	resolvedRuntime := resolveBroadcastRuntime(runtimeOption(broadcastRuntime))
	if err := broadcast.ApplyBroadcastTransition(true, transitionCommand, resolvedRuntime); err != nil {
		return transition.HandleTransitionPublishError(
			fmt.Sprintf("unable to publish %s transition update", context),
			err,
			publishPolicy,
		)
	}
	return nil
}

func runTransitionEventLoop[T any](ctx context.Context, label string, publishPolicy transition.TransitionPublishPolicy, ch <-chan T, mapEvent func(T) broadcast.BroadcastTransition, broadcastRuntime ...broadcast.BroadcastStore) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if publishPolicy == "" {
		publishPolicy = transition.TransitionPublishStrict
	}
	resolvedRuntime := resolveBroadcastRuntime(runtimeOption(broadcastRuntime))
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-ch:
			command := mapEvent(event)
			if command == nil {
				continue
			}
			if err := applyTransitionUpdate(label, command, publishPolicy, resolvedRuntime); err != nil {
				return fmt.Errorf("failed to apply %s transition update: %w", label, err)
			}
		}
	}
}

func runtimeOption(runtimes []broadcast.BroadcastStore) broadcast.BroadcastStore {
	if len(runtimes) == 0 {
		return nil
	}
	return runtimes[0]
}

func resolveBroadcastRuntime(runtime broadcast.BroadcastStore) broadcast.BroadcastStore {
	if runtime != nil {
		return runtime
	}
	return broadcast.DefaultBroadcastStateRuntime()
}
