package events

import (
	"context"
	"errors"
	"sync"

	"groundseg/structs"
)

var (
	defaultBrokerMu sync.RWMutex
	defaultBroker   EventBroker = NewTransitionBroker(100)
)

var (
	ErrTransitionPublishTimeout = errors.New("transition publish cancelled before enqueue")
	ErrTransitionBusFull        = errors.New("transition event bus is full")
	errTransitionBusNotDefined  = errors.New("transition broker is not defined")
	errTransitionBrokerNil      = errors.New("transition broker cannot be nil")
)

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

type transitionBus struct {
	urbitTransitionBus      chan structs.UrbitTransition
	systemTransitionBus     chan structs.SystemTransition
	newShipTransitionBus    chan structs.NewShipTransition
	importShipTransitionBus chan structs.UploadTransition
}

// NewTransitionBroker returns an in-memory broker with bounded buffers per transition type.
func NewTransitionBroker(bufferSize int) EventBroker {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	return &transitionBus{
		urbitTransitionBus:      make(chan structs.UrbitTransition, bufferSize),
		systemTransitionBus:     make(chan structs.SystemTransition, bufferSize),
		newShipTransitionBus:    make(chan structs.NewShipTransition, bufferSize),
		importShipTransitionBus: make(chan structs.UploadTransition, bufferSize),
	}
}

func SetEventBroker(broker EventBroker) {
	if broker == nil {
		panic(errTransitionBrokerNil)
	}
	defaultBrokerMu.Lock()
	defer defaultBrokerMu.Unlock()
	defaultBroker = broker
}

func getEventBroker() EventBroker {
	defaultBrokerMu.RLock()
	defer defaultBrokerMu.RUnlock()
	return defaultBroker
}

func publishWithFallback(ctx context.Context, publish func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return publish(ctx)
}

func publishDropOnFull[T any](ctx context.Context, ch chan T, event T) error {
	if ctx == nil {
		ctx = context.Background()
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

func PublishUrbitTransition(event structs.UrbitTransition) {
	_ = PublishUrbitTransitionWithContext(context.Background(), event)
}

func PublishUrbitTransitionWithContext(ctx context.Context, event structs.UrbitTransition) error {
	broker := getEventBroker()
	if broker == nil {
		return errTransitionBusNotDefined
	}
	return publishWithFallback(ctx, func(ctx context.Context) error {
		return broker.PublishUrbitTransition(ctx, event)
	})
}

func UrbitTransitions() <-chan structs.UrbitTransition {
	return getEventBroker().UrbitTransitions()
}

func PublishSystemTransition(event structs.SystemTransition) {
	_ = PublishSystemTransitionWithContext(context.Background(), event)
}

func PublishSystemTransitionWithContext(ctx context.Context, event structs.SystemTransition) error {
	broker := getEventBroker()
	if broker == nil {
		return errTransitionBusNotDefined
	}
	return publishWithFallback(ctx, func(ctx context.Context) error {
		return broker.PublishSystemTransition(ctx, event)
	})
}

func SystemTransitions() <-chan structs.SystemTransition {
	return getEventBroker().SystemTransitions()
}

func PublishNewShipTransition(event structs.NewShipTransition) {
	_ = PublishNewShipTransitionWithContext(context.Background(), event)
}

func PublishNewShipTransitionWithContext(ctx context.Context, event structs.NewShipTransition) error {
	broker := getEventBroker()
	if broker == nil {
		return errTransitionBusNotDefined
	}
	return publishWithFallback(ctx, func(ctx context.Context) error {
		return broker.PublishNewShipTransition(ctx, event)
	})
}

func NewShipTransitions() <-chan structs.NewShipTransition {
	return getEventBroker().NewShipTransitions()
}

func PublishImportShipTransition(event structs.UploadTransition) {
	_ = PublishImportShipTransitionWithContext(context.Background(), event)
}

func PublishImportShipTransitionWithContext(ctx context.Context, event structs.UploadTransition) error {
	broker := getEventBroker()
	if broker == nil {
		return errTransitionBusNotDefined
	}
	return publishWithFallback(ctx, func(ctx context.Context) error {
		return broker.PublishImportShipTransition(ctx, event)
	})
}

func ImportShipTransitions() <-chan structs.UploadTransition {
	return getEventBroker().ImportShipTransitions()
}

func (runtime *transitionBus) PublishUrbitTransition(ctx context.Context, event structs.UrbitTransition) error {
	return publishDropOnFull(ctx, runtime.urbitTransitionBus, event)
}

func (runtime *transitionBus) PublishSystemTransition(ctx context.Context, event structs.SystemTransition) error {
	return publishDropOnFull(ctx, runtime.systemTransitionBus, event)
}

func (runtime *transitionBus) PublishNewShipTransition(ctx context.Context, event structs.NewShipTransition) error {
	return publishDropOnFull(ctx, runtime.newShipTransitionBus, event)
}

func (runtime *transitionBus) PublishImportShipTransition(ctx context.Context, event structs.UploadTransition) error {
	return publishDropOnFull(ctx, runtime.importShipTransitionBus, event)
}

func (runtime *transitionBus) UrbitTransitions() <-chan structs.UrbitTransition {
	return runtime.urbitTransitionBus
}

func (runtime *transitionBus) SystemTransitions() <-chan structs.SystemTransition {
	return runtime.systemTransitionBus
}

func (runtime *transitionBus) NewShipTransitions() <-chan structs.NewShipTransition {
	return runtime.newShipTransitionBus
}

func (runtime *transitionBus) ImportShipTransitions() <-chan structs.UploadTransition {
	return runtime.importShipTransitionBus
}
