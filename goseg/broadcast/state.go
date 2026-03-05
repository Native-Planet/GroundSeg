package broadcast

import (
	"encoding/json"
	"errors"
	"fmt"
	"groundseg/broadcast/collectors"
	"groundseg/structs"
	"groundseg/transition"
	"maps"
	"sync"
	"time"
)

var (
	defaultBroadcastStateRuntimeMu sync.RWMutex
	defaultBroadcastStateRuntime   = NewBroadcastStateRuntime()
	errBroadcastRuntimeRequired    = errors.New("broadcast state runtime is required")
)

type broadcastStateRuntime struct {
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

func (runtime *broadcastStateRuntime) GetState() structs.AuthBroadcast {
	if runtime == nil {
		return structs.AuthBroadcast{}
	}
	runtime.RLock()
	defer runtime.RUnlock()
	return cloneBroadcastState(runtime.broadcastState)
}

func (runtime *broadcastStateRuntime) UpdateBroadcast(next structs.AuthBroadcast) {
	if runtime == nil {
		return
	}
	runtime.Lock()
	defer runtime.Unlock()
	runtime.broadcastState = cloneBroadcastState(next)
}

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

func (runtime *broadcastStateRuntime) AddSystemTransitionError(message string) {
	if runtime == nil || message == "" {
		return
	}
	runtime.Lock()
	defer runtime.Unlock()
	next := append([]string{message}, runtime.broadcastState.System.Transition.Error...)
	if len(next) > maxSystemTransitionErrors {
		next = next[:maxSystemTransitionErrors]
	}
	runtime.broadcastState.System.Transition.Error = next
}

func (runtime *broadcastStateRuntime) UpdateScheduledPack(patp string, meldNext time.Time) error {
	if runtime == nil {
		return errBroadcastRuntimeRequired
	}
	runtime.packMu.Lock()
	defer runtime.packMu.Unlock()
	runtime.scheduledPacks[patp] = meldNext
	return nil
}

func (runtime *broadcastStateRuntime) PublishSchedulePack(reason string) error {
	if runtime == nil {
		return errBroadcastRuntimeRequired
	}
	select {
	case runtime.schedulePackBus <- reason:
		return nil
	default:
		return errSchedulePackBusFull
	}
}

func (runtime *broadcastStateRuntime) SchedulePackEvents() <-chan string {
	if runtime == nil {
		return nil
	}
	return runtime.schedulePackBus
}

func (runtime *broadcastStateRuntime) BroadcastToClients() error {
	if runtime == nil {
		return errBroadcastRuntimeRequired
	}
	return broadcastToClientsWithRuntime(runtime)
}

func (runtime *broadcastStateRuntime) pierCollectorContract() collectors.BroadcastPierCollectorContract {
	if runtime != nil && runtime.pierCollector != nil {
		return runtime.pierCollector
	}
	return collectors.DefaultBroadcastPierCollectorContract()
}

func (runtime *broadcastStateRuntime) infoCollectorContract() collectors.BroadcastInfoCollectorContract {
	if runtime != nil && runtime.infoCollector != nil {
		return runtime.infoCollector
	}
	return collectors.DefaultBroadcastInfoCollectorContract()
}

func (runtime *broadcastStateRuntime) startramCollectorContract() collectors.BroadcastStartramCollectorContract {
	if runtime != nil && runtime.startramCollector != nil {
		return runtime.startramCollector
	}
	return collectors.DefaultBroadcastStartramCollectorContract()
}

func (runtime *broadcastStateRuntime) collectPierInfo(urbits map[string]structs.Urbit, scheduled func(string) time.Time) (map[string]structs.Urbit, error) {
	return runtime.pierCollectorContract().CollectPierInfo(urbits, scheduled)
}

func (runtime *broadcastStateRuntime) collectAppsInfo() structs.Apps {
	return runtime.infoCollectorContract().CollectAppsInfo()
}

func (runtime *broadcastStateRuntime) collectProfileInfo(regions map[string]structs.StartramRegion) structs.Profile {
	return runtime.infoCollectorContract().CollectProfileInfo(regions)
}

func (runtime *broadcastStateRuntime) collectSystemInfo() structs.System {
	return runtime.infoCollectorContract().CollectSystemInfo()
}

var errSchedulePackBusFull = errors.New("broadcast schedule bus is full")

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

// DefaultBroadcastStateRuntime returns the shared process-wide broadcast runtime for bootstrap
// code and callers that do not yet inject an explicit runtime.
func DefaultBroadcastStateRuntime() *broadcastStateRuntime {
	defaultBroadcastStateRuntimeMu.RLock()
	defer defaultBroadcastStateRuntimeMu.RUnlock()
	return defaultBroadcastStateRuntime
}

func SetDefaultBroadcastStateRuntime(runtime *broadcastStateRuntime) *broadcastStateRuntime {
	if runtime == nil {
		runtime = NewBroadcastStateRuntime()
	}
	defaultBroadcastStateRuntimeMu.Lock()
	defaultBroadcastStateRuntime = runtime
	defaultBroadcastStateRuntimeMu.Unlock()
	return runtime
}

func ResetDefaultBroadcastStateRuntime() *broadcastStateRuntime {
	return SetDefaultBroadcastStateRuntime(NewBroadcastStateRuntime())
}

func cloneBroadcastState(in structs.AuthBroadcast) structs.AuthBroadcast {
	out := in
	if in.Urbits != nil {
		out.Urbits = make(map[string]structs.Urbit, len(in.Urbits))
		for patp, urbit := range in.Urbits {
			cloned := urbit
			cloned.Info.RemoteTlonBackups = append([]structs.BackupObject(nil), urbit.Info.RemoteTlonBackups...)
			cloned.Info.LocalDailyTlonBackups = append([]structs.BackupObject(nil), urbit.Info.LocalDailyTlonBackups...)
			cloned.Info.LocalWeeklyTlonBackups = append([]structs.BackupObject(nil), urbit.Info.LocalWeeklyTlonBackups...)
			cloned.Info.LocalMonthlyTlonBackups = append([]structs.BackupObject(nil), urbit.Info.LocalMonthlyTlonBackups...)
			out.Urbits[patp] = cloned
		}
	}
	if in.System.Info.Drives != nil {
		out.System.Info.Drives = maps.Clone(in.System.Info.Drives)
	}
	if in.System.Info.SMART != nil {
		out.System.Info.SMART = maps.Clone(in.System.Info.SMART)
	}
	if in.System.Info.Usage.Disk != nil {
		out.System.Info.Usage.Disk = maps.Clone(in.System.Info.Usage.Disk)
	}
	out.System.Info.Usage.RAM = append([]uint64(nil), in.System.Info.Usage.RAM...)
	out.System.Info.Wifi.Networks = append([]string(nil), in.System.Info.Wifi.Networks...)
	if in.Profile.Startram.Info.Regions != nil {
		out.Profile.Startram.Info.Regions = maps.Clone(in.Profile.Startram.Info.Regions)
	}
	out.Profile.Startram.Info.StartramServices = append([]string(nil), in.Profile.Startram.Info.StartramServices...)
	out.System.Transition.Error = append([]string(nil), in.System.Transition.Error...)
	out.Logs.Containers.Wireguard.Logs = append([]any(nil), in.Logs.Containers.Wireguard.Logs...)
	out.Logs.System.Logs = append([]any(nil), in.Logs.System.Logs...)
	out.Apps.Penpai.Info.Models = append([]string(nil), in.Apps.Penpai.Info.Models...)
	return out
}

// GetStateJson returns a serialized broadcast envelope with an explicit auth level.
func GetStateJson(bState structs.AuthBroadcast, authLevel transition.BroadcastAuthLevel) ([]byte, error) {
	envelope := authBroadcastEnvelope{
		Type:      string(transition.BroadcastMessageTypeStructure),
		AuthLevel: string(authLevel),
		Payload:   bState,
	}
	broadcastJson, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshalling broadcast state payload: %w", err)
	}
	return broadcastJson, nil
}
