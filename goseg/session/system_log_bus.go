package session

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

func (bus *systemLogMessageBus) Publish(payload []byte) {
	if bus == nil || bus.channel == nil {
		return
	}
	bus.channel <- payload
}
