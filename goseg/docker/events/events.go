package events

import (
	"context"
	"errors"
	"sync"

	"groundseg/structs"
)

var (
	ErrTransitionPublishTimeout = errors.New("transition publish cancelled before enqueue")
	ErrTransitionBusFull        = errors.New("transition event bus is full")
	errTransitionBusNotDefined  = errors.New("transition broker is not defined")
)

var (
	defaultEventRuntimeMu sync.RWMutex
	defaultEventRuntime   = NewEventRuntime()
)

type EventRuntime struct {
	urbitTransitionBus      chan structs.UrbitTransition
	systemTransitionBus     chan structs.SystemTransition
	newShipTransitionBus    chan structs.NewShipTransition
	importShipTransitionBus chan structs.UploadTransition
}

type DelegatingEventBroker struct {
	delegate EventBroker
}

func DefaultEventRuntime() EventRuntime {
	defaultEventRuntimeMu.RLock()
	defer defaultEventRuntimeMu.RUnlock()
	return defaultEventRuntime
}

func SetDefaultEventRuntime(runtime EventRuntime) {
	defaultEventRuntimeMu.Lock()
	defer defaultEventRuntimeMu.Unlock()
	if runtime.urbitTransitionBus == nil ||
		runtime.systemTransitionBus == nil ||
		runtime.newShipTransitionBus == nil ||
		runtime.importShipTransitionBus == nil {
		runtime = NewEventRuntime()
	}
	defaultEventRuntime = runtime
}

func ResetDefaultEventRuntime() EventRuntime {
	runtime := NewEventRuntime()
	SetDefaultEventRuntime(runtime)
	return runtime
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

func NewDelegatingEventBroker(delegate EventBroker) DelegatingEventBroker {
	return DelegatingEventBroker{delegate: delegate}
}

// NewTransitionBroker returns the default in-memory transition broker implementation.
func NewTransitionBroker(bufferSize int) EventBroker {
	return NewEventRuntimeWithBuffer(bufferSize)
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
	return publishDropOnFull(ctx, runtime.urbitTransitionBus, event)
}

func (runtime EventRuntime) UrbitTransitions() <-chan structs.UrbitTransition {
	return runtime.urbitTransitionBus
}

func (runtime EventRuntime) PublishSystemTransition(ctx context.Context, event structs.SystemTransition) error {
	return publishDropOnFull(ctx, runtime.systemTransitionBus, event)
}

func (runtime EventRuntime) SystemTransitions() <-chan structs.SystemTransition {
	return runtime.systemTransitionBus
}

func (runtime EventRuntime) PublishNewShipTransition(ctx context.Context, event structs.NewShipTransition) error {
	return publishDropOnFull(ctx, runtime.newShipTransitionBus, event)
}

func (runtime EventRuntime) NewShipTransitions() <-chan structs.NewShipTransition {
	return runtime.newShipTransitionBus
}

func (runtime EventRuntime) PublishImportShipTransition(ctx context.Context, event structs.UploadTransition) error {
	return publishDropOnFull(ctx, runtime.importShipTransitionBus, event)
}

func (runtime EventRuntime) ImportShipTransitions() <-chan structs.UploadTransition {
	return runtime.importShipTransitionBus
}

func (runtime DelegatingEventBroker) PublishUrbitTransition(ctx context.Context, event structs.UrbitTransition) error {
	if runtime.delegate == nil {
		return errTransitionBusNotDefined
	}
	return runtime.delegate.PublishUrbitTransition(ctx, event)
}

func (runtime DelegatingEventBroker) UrbitTransitions() <-chan structs.UrbitTransition {
	if runtime.delegate == nil {
		return nil
	}
	return runtime.delegate.UrbitTransitions()
}

func (runtime DelegatingEventBroker) PublishSystemTransition(ctx context.Context, event structs.SystemTransition) error {
	if runtime.delegate == nil {
		return errTransitionBusNotDefined
	}
	return runtime.delegate.PublishSystemTransition(ctx, event)
}

func (runtime DelegatingEventBroker) SystemTransitions() <-chan structs.SystemTransition {
	if runtime.delegate == nil {
		return nil
	}
	return runtime.delegate.SystemTransitions()
}

func (runtime DelegatingEventBroker) PublishNewShipTransition(ctx context.Context, event structs.NewShipTransition) error {
	if runtime.delegate == nil {
		return errTransitionBusNotDefined
	}
	return runtime.delegate.PublishNewShipTransition(ctx, event)
}

func (runtime DelegatingEventBroker) NewShipTransitions() <-chan structs.NewShipTransition {
	if runtime.delegate == nil {
		return nil
	}
	return runtime.delegate.NewShipTransitions()
}

func (runtime DelegatingEventBroker) PublishImportShipTransition(ctx context.Context, event structs.UploadTransition) error {
	if runtime.delegate == nil {
		return errTransitionBusNotDefined
	}
	return runtime.delegate.PublishImportShipTransition(ctx, event)
}

func (runtime DelegatingEventBroker) ImportShipTransitions() <-chan structs.UploadTransition {
	if runtime.delegate == nil {
		return nil
	}
	return runtime.delegate.ImportShipTransitions()
}
