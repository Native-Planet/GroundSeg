package rectify

import (
	"context"
	"fmt"

	"groundseg/broadcast"
	"groundseg/transition"
)

func applyTransitionUpdate(context string, transitionCommand broadcast.BroadcastTransition, publishPolicy transition.TransitionPublishPolicy) error {
	if transitionCommand == nil {
		return nil
	}
	if err := broadcast.ApplyBroadcastTransition(true, transitionCommand); err != nil {
		return transition.HandleTransitionPublishError(
			fmt.Sprintf("unable to publish %s transition update", context),
			err,
			publishPolicy,
		)
	}
	return nil
}

func runTransitionEventLoop[T any](ctx context.Context, label string, publishPolicy transition.TransitionPublishPolicy, ch <-chan T, mapEvent func(T) broadcast.BroadcastTransition) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if publishPolicy == "" {
		publishPolicy = transition.TransitionPublishStrict
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-ch:
			command := mapEvent(event)
			if command == nil {
				continue
			}
			if err := applyTransitionUpdate(label, command, publishPolicy); err != nil {
				return fmt.Errorf("failed to apply %s transition update: %w", label, err)
			}
		}
	}
}
