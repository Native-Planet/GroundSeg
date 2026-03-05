package broadcast

import (
	"errors"
	"groundseg/broadcast/collectors"
	"groundseg/structs"
	"sync"
	"time"
)

var (
	ErrBroadcastRuntimeRequired    = errors.New("broadcast state runtime is required")
	defaultSchedulePackPublishWait = 200 * time.Millisecond
)

type broadcastStateRuntime struct {
	// Nil receiver policy:
	// - read-only accessors return zero values
	// - mutating/publish operations return ErrBroadcastRuntimeRequired
	sync.RWMutex      // synchronize access to broadcastState
	broadcastState    structs.AuthBroadcast
	scheduledPacks    map[string]time.Time
	schedulePackBus   chan string
	packMu            sync.RWMutex
	pierCollector     collectors.BroadcastPierCollectorContract
	infoCollector     collectors.BroadcastInfoCollectorContract
	startramCollector collectors.BroadcastStartramCollectorContract
	delivery          broadcastDeliveryRuntime
}

const maxSystemTransitionErrors = 8

func (runtime *broadcastStateRuntime) BroadcastToClients() error {
	if runtime == nil {
		return ErrBroadcastRuntimeRequired
	}
	return broadcastToClientsWithRuntime(runtime)
}

func NewBroadcastStateRuntime() *broadcastStateRuntime {
	return &broadcastStateRuntime{
		broadcastState:    structs.AuthBroadcast{},
		scheduledPacks:    make(map[string]time.Time),
		schedulePackBus:   make(chan string, 64),
		pierCollector:     collectors.DefaultBroadcastPierCollectorContract(),
		infoCollector:     collectors.DefaultBroadcastInfoCollectorContract(),
		startramCollector: collectors.DefaultBroadcastStartramCollectorContract(),
		delivery:          defaultBroadcastDeliveryRuntime(),
	}
}
