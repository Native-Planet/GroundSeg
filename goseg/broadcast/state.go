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

	"go.uber.org/zap"
)

var (
	defaultBroadcastStateRuntime = NewBroadcastStateRuntime()
)

type broadcastStateRuntime struct {
	sync.RWMutex      // synchronize access to broadcastState
	broadcastState    structs.AuthBroadcast
	scheduledPacks    map[string]time.Time
	urbitTransitions  map[string]structs.UrbitTransitionBroadcast
	schedulePackBus   chan string
	packMu            sync.RWMutex
	urbTransMu        sync.RWMutex
	sysTransMu        sync.RWMutex
	systemTransitions structs.SystemTransitionBroadcast
	pierCollector     collectors.BroadcastPierCollectorContract
	infoCollector     collectors.BroadcastInfoCollectorContract
	startramCollector collectors.BroadcastStartramCollectorContract
}

func (runtime *broadcastStateRuntime) GetState() structs.AuthBroadcast {
	if runtime == nil {
		runtime = resolveDefaultBroadcastStateRuntime()
	}
	runtime.RLock()
	defer runtime.RUnlock()
	return cloneBroadcastState(runtime.broadcastState)
}

func (runtime *broadcastStateRuntime) UpdateBroadcast(next structs.AuthBroadcast) {
	if runtime == nil {
		runtime = resolveDefaultBroadcastStateRuntime()
	}
	runtime.Lock()
	defer runtime.Unlock()
	runtime.broadcastState = next
}

func (runtime *broadcastStateRuntime) GetScheduledPack(patp string) time.Time {
	if runtime == nil {
		runtime = resolveDefaultBroadcastStateRuntime()
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
		runtime = resolveDefaultBroadcastStateRuntime()
	}
	runtime.packMu.Lock()
	defer runtime.packMu.Unlock()
	runtime.scheduledPacks[patp] = meldNext
	return nil
}

func (runtime *broadcastStateRuntime) PublishSchedulePack(reason string) error {
	if runtime == nil {
		runtime = resolveDefaultBroadcastStateRuntime()
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
		runtime = resolveDefaultBroadcastStateRuntime()
	}
	return runtime.schedulePackBus
}

func (runtime *broadcastStateRuntime) BroadcastToClients() error {
	if runtime == nil {
		runtime = resolveDefaultBroadcastStateRuntime()
	}
	return broadcastToClientsWithRuntime(runtime)
}

var errSchedulePackBusFull = errors.New("broadcast schedule bus is full")

func NewBroadcastStateRuntime() *broadcastStateRuntime {
	return &broadcastStateRuntime{
		broadcastState:    structs.AuthBroadcast{},
		scheduledPacks:    make(map[string]time.Time),
		urbitTransitions:  make(map[string]structs.UrbitTransitionBroadcast),
		schedulePackBus:   make(chan string, 64),
		pierCollector:     collectors.DefaultBroadcastPierCollectorContract(),
		infoCollector:     collectors.DefaultBroadcastInfoCollectorContract(),
		startramCollector: collectors.DefaultBroadcastStartramCollectorContract(),
	}
}

func DefaultBroadcastStateRuntime() *broadcastStateRuntime {
	return resolveDefaultBroadcastStateRuntime()
}

func resolveDefaultBroadcastStateRuntime() *broadcastStateRuntime {
	return defaultBroadcastStateRuntime
}

func resolveBroadcastStateRuntime(runtime ...*broadcastStateRuntime) *broadcastStateRuntime {
	if len(runtime) > 0 && runtime[0] != nil {
		return runtime[0]
	}
	return resolveDefaultBroadcastStateRuntime()
}

// take in config file and addt'l info to initialize broadcast
func bootstrapBroadcastState(runtime ...*broadcastStateRuntime) error {
	zap.L().Info("Bootstrapping state")
	// this returns a map of ship:running status
	zap.L().Info("Resolving pier status")
	resolved := resolveBroadcastStateRuntime(runtime...)
	state := resolved.GetState()
	urbits, err := constructPierInfoWithRuntime(resolved, state.Urbits, resolved.GetScheduledPack)
	if err != nil {
		return fmt.Errorf("bootstrap broadcast state: %w", err)
	}
	nextState := structs.AuthBroadcast{
		Urbits:  urbits,
		System:  constructSystemInfoWithRuntime(resolved),
		Profile: constructProfileInfoWithRuntime(resolved, state.Profile.Startram.Info.Regions),
		Apps:    constructAppsInfoWithRuntime(resolved),
	}
	resolved.Lock()
	resolved.broadcastState = nextState
	resolved.Unlock()
	// start looping info refreshes
	StartBroadcastLoop()
	return nil
}

func LoadStartramRegionsWithRuntime(runtime ...*broadcastStateRuntime) error {
	resolved := resolveBroadcastStateRuntime(runtime...)
	zap.L().Info("Retrieving StarTram region info")
	regions, err := broadcastStartramCollectorContract(resolved).LoadStartramRegions()
	if err != nil {
		return fmt.Errorf("load startram regions: %w", err)
	}
	resolved.Lock()
	resolved.broadcastState.Profile.Startram.Info.Regions = regions
	resolved.Unlock()
	return nil
}

func broadcastPierCollectorContract(runtime *broadcastStateRuntime) collectors.BroadcastPierCollectorContract {
	if runtime != nil && runtime.pierCollector != nil {
		return runtime.pierCollector
	}
	return collectors.DefaultBroadcastPierCollectorContract()
}

func broadcastInfoCollectorContract(runtime *broadcastStateRuntime) collectors.BroadcastInfoCollectorContract {
	if runtime != nil && runtime.infoCollector != nil {
		return runtime.infoCollector
	}
	return collectors.DefaultBroadcastInfoCollectorContract()
}

func broadcastStartramCollectorContract(runtime *broadcastStateRuntime) collectors.BroadcastStartramCollectorContract {
	if runtime != nil && runtime.startramCollector != nil {
		return runtime.startramCollector
	}
	return collectors.DefaultBroadcastStartramCollectorContract()
}

func constructPierInfoWithRuntime(runtime *broadcastStateRuntime, urbits map[string]structs.Urbit, scheduled func(string) time.Time) (map[string]structs.Urbit, error) {
	return broadcastPierCollectorContract(runtime).CollectPierInfo(urbits, scheduled)
}

func constructAppsInfoWithRuntime(runtime *broadcastStateRuntime) structs.Apps {
	return broadcastInfoCollectorContract(runtime).CollectAppsInfo()
}

func constructProfileInfoWithRuntime(runtime *broadcastStateRuntime, regions map[string]structs.StartramRegion) structs.Profile {
	return broadcastInfoCollectorContract(runtime).CollectProfileInfo(regions)
}

func constructSystemInfoWithRuntime(runtime *broadcastStateRuntime) structs.System {
	return broadcastInfoCollectorContract(runtime).CollectSystemInfo()
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

// return json string of current broadcast state
func GetStateJson(bState structs.AuthBroadcast) ([]byte, error) {
	envelope := authBroadcastEnvelope{
		Type:      string(transition.BroadcastMessageTypeStructure),
		AuthLevel: string(transition.BroadcastAuthLevelAuthorized),
		Payload:   bState,
	}
	broadcastJson, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshalling broadcast state payload: %w", err)
	}
	return broadcastJson, nil
}

func ReloadUrbits() error {
	zap.L().Info("Reloading ships in broadcast")
	resolved := resolveBroadcastStateRuntime()
	urbits, err := constructPierInfoWithRuntime(resolved, resolved.GetState().Urbits, resolved.GetScheduledPack)
	if err != nil {
		return fmt.Errorf("reload urbit states for broadcast: %w", err)
	}
	resolved.Lock()
	resolved.broadcastState.Urbits = urbits
	resolved.Unlock()
	return nil
}
