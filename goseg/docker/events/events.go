package events

import (
	"context"
	"errors"
	"time"

	"groundseg/structs"
)

var (
	ErrTransitionPublishCancelled = errors.New("transition publish cancelled before enqueue")
	ErrTransitionPublishTimeout   = errors.New("transition publish timed out before enqueue")
	ErrTransitionBusFull          = errors.New("transition event bus is full")
	errTransitionBusNotDefined    = errors.New("transition broker is not defined")

	defaultTransitionPublishWait = 200 * time.Millisecond
)

type EventRuntime struct {
	urbitTransitionBus      transitionBus[structs.UrbitTransition]
	systemTransitionBus     transitionBus[structs.SystemTransition]
	newShipTransitionBus    transitionBus[structs.NewShipTransition]
	importShipTransitionBus transitionBus[structs.UploadTransition]
}

type transitionBus[T any] struct {
	ch chan T
}

func NewEventRuntime() EventRuntime {
	return newTransitionRuntime(100)
}

func NewEventRuntimeWithBuffer(bufferSize int) EventRuntime {
	return newTransitionRuntime(bufferSize)
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

type defaultEventBroker struct{}

// NewGlobalDefaultBroker returns an EventBroker bound to the mutable process default runtime.
// Calls are resolved against the current default on each publish/subscribe call.
func NewGlobalDefaultBroker() EventBroker {
	return defaultEventBroker{}
}

// NewIsolatedRuntimeBroker returns an in-memory broker with private channels
// that are unaffected by global default runtime swaps.
func NewIsolatedRuntimeBroker(bufferSize int) EventBroker {
	return NewEventRuntimeWithBuffer(bufferSize)
}

// NewDefaultEventBroker returns an EventBroker that always routes through the
// current process default runtime snapshot at construction time.
func NewDefaultEventBroker() EventBroker {
	return NewRuntimeBoundBroker(DefaultEventRuntime())
}

// NewTransitionBroker returns the default in-memory transition broker implementation.
func NewTransitionBroker(bufferSize int) EventBroker {
	return NewIsolatedRuntimeBroker(bufferSize)
}

func newTransitionRuntime(bufferSize int) EventRuntime {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	return EventRuntime{
		urbitTransitionBus:      newTransitionBus[structs.UrbitTransition](bufferSize),
		systemTransitionBus:     newTransitionBus[structs.SystemTransition](bufferSize),
		newShipTransitionBus:    newTransitionBus[structs.NewShipTransition](bufferSize),
		importShipTransitionBus: newTransitionBus[structs.UploadTransition](bufferSize),
	}
}

func newTransitionBus[T any](bufferSize int) transitionBus[T] {
	return transitionBus[T]{ch: make(chan T, bufferSize)}
}

func (bus transitionBus[T]) defined() bool {
	return bus.ch != nil
}

func (bus transitionBus[T]) publish(ctx context.Context, event T) error {
	return publishWithBoundedWait(ctx, bus.ch, event, defaultTransitionPublishWait)
}

func (bus transitionBus[T]) subscribe() <-chan T {
	return bus.ch
}

func publishWithBoundedWait[T any](ctx context.Context, ch chan T, event T, maxWait time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if ch == nil {
		return errTransitionBusNotDefined
	}
	if maxWait <= 0 {
		maxWait = defaultTransitionPublishWait
	}
	timer := time.NewTimer(maxWait)
	defer timer.Stop()
	select {
	case ch <- event:
		return nil
	case <-ctx.Done():
		return errors.Join(ErrTransitionPublishCancelled, ctx.Err())
	case <-timer.C:
		return errors.Join(ErrTransitionPublishTimeout, ErrTransitionBusFull)
	}
}

func (runtime EventRuntime) PublishUrbitTransition(ctx context.Context, event structs.UrbitTransition) error {
	return runtime.urbitTransitionBus.publish(ctx, event)
}

func (runtime EventRuntime) UrbitTransitions() <-chan structs.UrbitTransition {
	return runtime.urbitTransitionBus.subscribe()
}

func (runtime EventRuntime) PublishSystemTransition(ctx context.Context, event structs.SystemTransition) error {
	return runtime.systemTransitionBus.publish(ctx, event)
}

func (runtime EventRuntime) SystemTransitions() <-chan structs.SystemTransition {
	return runtime.systemTransitionBus.subscribe()
}

func (runtime EventRuntime) PublishNewShipTransition(ctx context.Context, event structs.NewShipTransition) error {
	return runtime.newShipTransitionBus.publish(ctx, event)
}

func (runtime EventRuntime) NewShipTransitions() <-chan structs.NewShipTransition {
	return runtime.newShipTransitionBus.subscribe()
}

func (runtime EventRuntime) PublishImportShipTransition(ctx context.Context, event structs.UploadTransition) error {
	return runtime.importShipTransitionBus.publish(ctx, event)
}

func (runtime EventRuntime) ImportShipTransitions() <-chan structs.UploadTransition {
	return runtime.importShipTransitionBus.subscribe()
}

func (defaultEventBroker) PublishUrbitTransition(ctx context.Context, event structs.UrbitTransition) error {
	return DefaultEventRuntime().PublishUrbitTransition(ctx, event)
}

func (defaultEventBroker) UrbitTransitions() <-chan structs.UrbitTransition {
	return DefaultEventRuntime().UrbitTransitions()
}

func (defaultEventBroker) PublishSystemTransition(ctx context.Context, event structs.SystemTransition) error {
	return DefaultEventRuntime().PublishSystemTransition(ctx, event)
}

func (defaultEventBroker) SystemTransitions() <-chan structs.SystemTransition {
	return DefaultEventRuntime().SystemTransitions()
}

func (defaultEventBroker) PublishNewShipTransition(ctx context.Context, event structs.NewShipTransition) error {
	return DefaultEventRuntime().PublishNewShipTransition(ctx, event)
}

func (defaultEventBroker) NewShipTransitions() <-chan structs.NewShipTransition {
	return DefaultEventRuntime().NewShipTransitions()
}

func (defaultEventBroker) PublishImportShipTransition(ctx context.Context, event structs.UploadTransition) error {
	return DefaultEventRuntime().PublishImportShipTransition(ctx, event)
}

func (defaultEventBroker) ImportShipTransitions() <-chan structs.UploadTransition {
	return DefaultEventRuntime().ImportShipTransitions()
}
