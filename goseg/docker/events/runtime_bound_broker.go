package events

import (
	"context"

	"groundseg/structs"
)

type runtimeBoundEventBroker struct {
	runtime EventRuntime
}

func normalizeRuntime(runtime EventRuntime) EventRuntime {
	if !runtime.urbitTransitionBus.defined() ||
		!runtime.systemTransitionBus.defined() ||
		!runtime.newShipTransitionBus.defined() ||
		!runtime.importShipTransitionBus.defined() {
		return NewEventRuntime()
	}
	return runtime
}

// NewRuntimeBoundBroker returns an EventBroker pinned to a specific runtime snapshot.
// Unlike NewGlobalDefaultBroker, this broker does not follow future default runtime swaps.
func NewRuntimeBoundBroker(runtime EventRuntime) EventBroker {
	return runtimeBoundEventBroker{runtime: normalizeRuntime(runtime)}
}

func (broker runtimeBoundEventBroker) PublishUrbitTransition(ctx context.Context, event structs.UrbitTransition) error {
	return broker.runtime.PublishUrbitTransition(ctx, event)
}

func (broker runtimeBoundEventBroker) UrbitTransitions() <-chan structs.UrbitTransition {
	return broker.runtime.UrbitTransitions()
}

func (broker runtimeBoundEventBroker) PublishSystemTransition(ctx context.Context, event structs.SystemTransition) error {
	return broker.runtime.PublishSystemTransition(ctx, event)
}

func (broker runtimeBoundEventBroker) SystemTransitions() <-chan structs.SystemTransition {
	return broker.runtime.SystemTransitions()
}

func (broker runtimeBoundEventBroker) PublishNewShipTransition(ctx context.Context, event structs.NewShipTransition) error {
	return broker.runtime.PublishNewShipTransition(ctx, event)
}

func (broker runtimeBoundEventBroker) NewShipTransitions() <-chan structs.NewShipTransition {
	return broker.runtime.NewShipTransitions()
}

func (broker runtimeBoundEventBroker) PublishImportShipTransition(ctx context.Context, event structs.UploadTransition) error {
	return broker.runtime.PublishImportShipTransition(ctx, event)
}

func (broker runtimeBoundEventBroker) ImportShipTransitions() <-chan structs.UploadTransition {
	return broker.runtime.ImportShipTransitions()
}
