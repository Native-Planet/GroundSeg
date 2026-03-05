package session

import "errors"

var (
	ErrSystemLogBusNotDefined = errors.New("system log bus is not defined")
	ErrSystemLogBusFull       = errors.New("system log bus is full")
)

type systemLogMessageBus struct {
	channel chan []byte
}

func newSystemLogMessageBus(buffer int) *systemLogMessageBus {
	if buffer <= 0 {
		buffer = 1
	}
	return &systemLogMessageBus{
		channel: make(chan []byte, buffer),
	}
}

func (bus *systemLogMessageBus) Messages() <-chan []byte {
	if bus == nil || bus.channel == nil {
		return nil
	}
	return bus.channel
}

func (bus *systemLogMessageBus) Publish(payload []byte) error {
	if bus == nil || bus.channel == nil {
		return ErrSystemLogBusNotDefined
	}
	select {
	case bus.channel <- payload:
		return nil
	default:
		return ErrSystemLogBusFull
	}
}
