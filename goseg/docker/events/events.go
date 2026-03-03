package events

import (
	"context"
	"errors"

	"groundseg/structs"
)

var (
	ErrTransitionPublishTimeout = errors.New("transition publish cancelled before enqueue")
	ErrTransitionBusFull        = errors.New("transition event bus is full")
	errTransitionBusNotDefined  = errors.New("transition broker is not defined")
)

type EventRuntime struct {
	urbitTransitionBus      chan structs.UrbitTransition
	systemTransitionBus     chan structs.SystemTransition
	newShipTransitionBus    chan structs.NewShipTransition
	importShipTransitionBus chan structs.UploadTransition
}

func NewDefaultEventRuntime() EventRuntime {
	return NewEventRuntime()
}

func NewEventRuntime(_ ...EventBroker) EventRuntime {
	return newTransitionRuntime(100)
}

var defaultEventRuntime = NewDefaultEventRuntime()

func DefaultEventRuntime() EventRuntime {
	return defaultEventRuntime
}

type EventBroker interface {
	PublishUrbitTransition(context.Context, structs.UrbitTransition) error
	PublishSystemTransition(context.Context, structs.SystemTransition) error
	PublishNewShipTransition(context.Context, structs.NewShipTransition) error
	PublishImportShipTransition(context.Context, structs.UploadTransition) error
	UrbitTransitions() <-chan structs.UrbitTransition
	SystemTransitions() <-chan structs.SystemTransition
	NewShipTransitions() <-chan structs.NewShipTransition
	ImportShipTransitions() <-chan structs.UploadTransition
}

// NewTransitionBroker returns the default in-memory transition broker implementation.
func NewTransitionBroker(bufferSize int) EventBroker {
	return newTransitionRuntime(bufferSize)
}

func newTransitionRuntime(bufferSize int) EventRuntime {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	return EventRuntime{
		urbitTransitionBus:      make(chan structs.UrbitTransition, bufferSize),
		systemTransitionBus:     make(chan structs.SystemTransition, bufferSize),
		newShipTransitionBus:    make(chan structs.NewShipTransition, bufferSize),
		importShipTransitionBus: make(chan structs.UploadTransition, bufferSize),
	}
}

func publishDropOnFull[T any](ctx context.Context, ch chan T, event T) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if ch == nil {
		return errTransitionBusNotDefined
	}
	select {
	case ch <- event:
		return nil
	case <-ctx.Done():
		return ErrTransitionPublishTimeout
	default:
		return ErrTransitionBusFull
	}
}

func (runtime EventRuntime) PublishUrbitTransition(ctx context.Context, event structs.UrbitTransition) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return publishDropOnFull(ctx, runtime.urbitTransitionBus, event)
}

func (runtime EventRuntime) UrbitTransitions() <-chan structs.UrbitTransition {
	return runtime.urbitTransitionBus
}

func (runtime EventRuntime) PublishSystemTransition(ctx context.Context, event structs.SystemTransition) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return publishDropOnFull(ctx, runtime.systemTransitionBus, event)
}

func (runtime EventRuntime) SystemTransitions() <-chan structs.SystemTransition {
	return runtime.systemTransitionBus
}

func (runtime EventRuntime) PublishNewShipTransition(ctx context.Context, event structs.NewShipTransition) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return publishDropOnFull(ctx, runtime.newShipTransitionBus, event)
}

func (runtime EventRuntime) NewShipTransitions() <-chan structs.NewShipTransition {
	return runtime.newShipTransitionBus
}

func (runtime EventRuntime) PublishImportShipTransition(ctx context.Context, event structs.UploadTransition) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return publishDropOnFull(ctx, runtime.importShipTransitionBus, event)
}

func (runtime EventRuntime) ImportShipTransitions() <-chan structs.UploadTransition {
	return runtime.importShipTransitionBus
}
