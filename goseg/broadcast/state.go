package broadcast

import (
	"encoding/json"
	"fmt"
	"groundseg/broadcast/collectors"
	"groundseg/structs"
	"groundseg/transition"
	"maps"
	"time"
	"sync"

	"go.uber.org/zap"
)

var defaultBroadcastStateRuntime = newBroadcastStateRuntime()

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
}

func newBroadcastStateRuntime() *broadcastStateRuntime {
	return &broadcastStateRuntime{
		broadcastState:   structs.AuthBroadcast{},
		scheduledPacks:   make(map[string]time.Time),
		urbitTransitions: make(map[string]structs.UrbitTransitionBroadcast),
		schedulePackBus:  make(chan string),
	}
}

func NewBroadcastStateRuntime() *broadcastStateRuntime {
	return newBroadcastStateRuntime()
}

func DefaultBroadcastStateRuntime() *broadcastStateRuntime {
	return defaultBroadcastStateRuntime
}

func resolveBroadcastStateRuntime(runtime ...*broadcastStateRuntime) *broadcastStateRuntime {
	if len(runtime) > 0 && runtime[0] != nil {
		return runtime[0]
	}
	return DefaultBroadcastStateRuntime()
}

func PublishSchedulePack(reason string, runtime ...*broadcastStateRuntime) {
	resolved := resolveBroadcastStateRuntime(runtime...)
	resolved.schedulePackBus <- reason
}

func SchedulePackEvents(runtime ...*broadcastStateRuntime) <-chan string {
	return resolveBroadcastStateRuntime(runtime...).schedulePackBus
}

// take in config file and addt'l info to initialize broadcast
func bootstrapBroadcastState(runtime ...*broadcastStateRuntime) error {
	zap.L().Info("Bootstrapping state")
	// this returns a map of ship:running status
	zap.L().Info("Resolving pier status")
	urbits, err := ConstructPierInfo()
	if err != nil {
		return fmt.Errorf("bootstrap broadcast state: %w", err)
	}
	nextState := structs.AuthBroadcast{
		Urbits:  urbits,
		System:  constructSystemInfo(),
		Profile: constructProfileInfo(),
		Apps:    constructAppsInfo(),
	}
	resolved := resolveBroadcastStateRuntime(runtime...)
	resolved.Lock()
	resolved.broadcastState = nextState
	resolved.Unlock()
	// start looping info refreshes
	StartBroadcastLoop()
	return nil
}

// put startram regions into broadcast struct
func LoadStartramRegions() error {
	return LoadStartramRegionsWithRuntime()
}

func LoadStartramRegionsWithRuntime(runtime ...*broadcastStateRuntime) error {
	zap.L().Info("Retrieving StarTram region info")
	regions, err := collectors.LoadStartramRegions()
	if err != nil {
		return fmt.Errorf("load startram regions: %w", err)
	}
	resolved := resolveBroadcastStateRuntime(runtime...)
	resolved.Lock()
	resolved.broadcastState.Profile.Startram.Info.Regions = regions
	resolved.Unlock()
	return nil
}

// this is for building the broadcast objects describing piers
func ConstructPierInfo() (map[string]structs.Urbit, error) {
	state := GetState()
	return collectors.ConstructPierInfo(state.Urbits, GetScheduledPack)
}

// Return a clone of apps info built from config state.
func constructAppsInfo() structs.Apps {
	return collectors.ConstructAppsInfo()
}

func constructProfileInfo() structs.Profile {
	state := GetState()
	return collectors.ConstructProfileInfo(state.Profile.Startram.Info.Regions)
}

// put together the system[usage] subobject
func constructSystemInfo() structs.System {
	return collectors.ConstructSystemInfo()
}

func UpdateScheduledPack(patp string, meldNext time.Time) error {
	resolved := resolveBroadcastStateRuntime()
	resolved.packMu.Lock()
	defer resolved.packMu.Unlock()
	resolved.scheduledPacks[patp] = meldNext
	return nil
}

func GetScheduledPack(patp string) time.Time {
	resolved := resolveBroadcastStateRuntime()
	resolved.packMu.RLock()
	defer resolved.packMu.RUnlock()
	nextPack, exists := resolved.scheduledPacks[patp]
	if !exists {
		return time.Time{}
	}
	return nextPack
}

// stupid update method instead of psychotic recursion
func UpdateBroadcast(broadcast structs.AuthBroadcast) {
	resolved := resolveBroadcastStateRuntime()
	resolved.Lock()
	defer resolved.Unlock()
	resolved.broadcastState = broadcast
}

// return broadcast state
func GetState() structs.AuthBroadcast {
	resolved := resolveBroadcastStateRuntime()
	resolved.RLock()
	defer resolved.RUnlock()
	return cloneBroadcastState(resolved.broadcastState)
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
	urbits, err := ConstructPierInfo()
	if err != nil {
		return fmt.Errorf("reload urbit states for broadcast: %w", err)
	}
	resolved := resolveBroadcastStateRuntime()
	resolved.Lock()
	resolved.broadcastState.Urbits = urbits
	resolved.Unlock()
	return nil
}
