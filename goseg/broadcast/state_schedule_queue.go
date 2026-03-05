package broadcast

import (
	"errors"
	"time"
)

var errSchedulePackBusFull = errors.New("broadcast schedule bus is full")

func (runtime *broadcastStateRuntime) GetScheduledPack(patp string) time.Time {
	if runtime == nil {
		return time.Time{}
	}
	runtime.packMu.RLock()
	defer runtime.packMu.RUnlock()
	nextPack, exists := runtime.scheduledPacks[patp]
	if !exists {
		return time.Time{}
	}
	return nextPack
}

func (runtime *broadcastStateRuntime) UpdateScheduledPack(patp string, meldNext time.Time) error {
	if runtime == nil {
		return ErrBroadcastRuntimeRequired
	}
	runtime.packMu.Lock()
	defer runtime.packMu.Unlock()
	runtime.scheduledPacks[patp] = meldNext
	return nil
}

func (runtime *broadcastStateRuntime) PublishSchedulePack(reason string) error {
	if runtime == nil {
		return ErrBroadcastRuntimeRequired
	}
	timer := time.NewTimer(defaultSchedulePackPublishWait)
	defer timer.Stop()
	select {
	case runtime.schedulePackBus <- reason:
		return nil
	case <-timer.C:
		return errSchedulePackBusFull
	}
}

func (runtime *broadcastStateRuntime) SchedulePackEvents() <-chan string {
	if runtime == nil {
		return nil
	}
	return runtime.schedulePackBus
}
