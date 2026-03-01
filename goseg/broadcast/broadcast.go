package broadcast

import (
	"encoding/json"
	"fmt"
	"groundseg/auth"
	"groundseg/broadcast/collectors"
	"groundseg/leak"
	"groundseg/structs"
	"groundseg/transition"
	"maps"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	broadcastInterval = 1 * time.Second // how often we refresh system info
	broadcastState    structs.AuthBroadcast
	scheduledPacks    = make(map[string]time.Time)
	urbitTransitions  = make(map[string]structs.UrbitTransitionBroadcast)
	schedulePackBus   = make(chan string)
	systemTransitions structs.SystemTransitionBroadcast
	packMu            sync.RWMutex
	urbTransMu        sync.RWMutex
	sysTransMu        sync.RWMutex
	mu                sync.RWMutex // synchronize access to broadcastState
)

var isConfigReady = func() bool { return true }

func SetConfigReadyCheck(fn func() bool) {
	if fn == nil {
		isConfigReady = func() bool { return true }
		return
	}
	isConfigReady = fn
}

type broadcastToClientsRuntime interface {
	emitToLeak(structs.AuthBroadcast)
	hasAuthSession() bool
	broadcastAuth([]byte) error
}

type authBroadcastEnvelope struct {
	Type      string                `json:"type"`
	AuthLevel string                `json:"auth_level"`
	Payload   structs.AuthBroadcast `json:"payload"`
}

type broadcastToClientsRuntimeDefault struct{}

func (broadcastToClientsRuntimeDefault) emitToLeak(bState structs.AuthBroadcast) {
	select {
	case leak.LeakChan <- bState:
	default:
		go func() {
			leak.LeakChan <- bState
		}()
	}
}

func (broadcastToClientsRuntimeDefault) hasAuthSession() bool {
	cm := auth.GetClientManager()
	if cm == nil {
		return false
	}
	return cm.HasAuthSession()
}

func (broadcastToClientsRuntimeDefault) broadcastAuth(payload []byte) error {
	auth.ClientManager.BroadcastAuth(payload)
	return nil
}

func PublishSchedulePack(reason string) {
	schedulePackBus <- reason
}

func SchedulePackEvents() <-chan string {
	return schedulePackBus
}

func Initialize() error {
	if !isConfigReady() {
		return fmt.Errorf("config subsystem is not initialized")
	}
	if err := bootstrapBroadcastState(); err != nil {
		return fmt.Errorf("unable to initialize broadcast state: %v", err)
	}
	if err := LoadStartramRegions(); err != nil {
		zap.L().Error("Couldn't load StarTram regions")
	}
	return nil
}

// serialized single thread for ws writes (mutex instead so this isnt necessary)
// func WsDigester() {
// 	for {
// 		event := <-structs.WsEventBus
// 		if event.Conn.Conn != nil {
// 			if err := event.Conn.Conn.WriteMessage(websocket.TextMessage, event.Data); err != nil {
// 				zap.L().Warn(fmt.Sprintf("WS error: %v", err))
// 				if err = auth.WsNilSession(event.Conn.Conn); err != nil {
// 					zap.L().Warn("Couldn't remove WS session")
// 				}
// 				continue
// 			}
// 		}
// 	}
// }

func UpdateScheduledPack(patp string, meldNext time.Time) error {
	packMu.Lock()
	defer packMu.Unlock()
	scheduledPacks[patp] = meldNext
	return nil
}

func GetScheduledPack(patp string) time.Time {
	packMu.Lock()
	defer packMu.Unlock()
	nextPack, exists := scheduledPacks[patp]
	if !exists {
		return time.Time{}
	}
	return nextPack
}

// take in config file and addt'l info to initialize broadcast
func bootstrapBroadcastState() error {
	zap.L().Info("Bootstrapping state")
	// this returns a map of ship:running status
	zap.L().Info("Resolving pier status")
	urbits, err := ConstructPierInfo()
	if err != nil {
		return err
	}
	// update broadcastState with ship info
	mu.Lock()
	broadcastState.Urbits = urbits
	mu.Unlock()
	// update with system state
	sysInfo := constructSystemInfo()
	mu.Lock()
	broadcastState.System = sysInfo
	mu.Unlock()
	// update with profile state
	profileInfo := constructProfileInfo()
	mu.Lock()
	broadcastState.Profile = profileInfo
	mu.Unlock()
	// update with apps state
	appsInfo := constructAppsInfo()
	mu.Lock()
	broadcastState.Apps = appsInfo
	mu.Unlock()
	// start looping info refreshes
	go BroadcastLoop()
	return nil
}

func GetStartramServices() error {
	return collectors.GetStartramServices()
}

// put startram regions into broadcast struct
func LoadStartramRegions() error {
	zap.L().Info("Retrieving StarTram region info")
	regions, err := collectors.LoadStartramRegions()
	if err != nil {
		return err
	}
	mu.Lock()
	broadcastState.Profile.Startram.Info.Regions = regions
	mu.Unlock()
	return nil
}

// this is for building the broadcast objects describing piers
func ConstructPierInfo() (map[string]structs.Urbit, error) {
	state := GetState()
	return collectors.ConstructPierInfo(state, GetScheduledPack)
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

// stupid update method instead of psychotic recursion
func UpdateBroadcast(broadcast structs.AuthBroadcast) {
	mu.Lock()
	defer mu.Unlock()
	broadcastState = broadcast
}

// ApplyBroadcastUpdate mutates the current broadcast snapshot and optionally notifies connected clients.
func ApplyBroadcastUpdate(notify bool, mutate func(state *structs.AuthBroadcast)) error {
	current := GetState()
	mutate(&current)
	UpdateBroadcast(current)
	if !notify {
		return nil
	}
	return BroadcastToClients()
}

// return broadcast state
func GetState() structs.AuthBroadcast {
	mu.RLock()
	defer mu.RUnlock()
	return cloneBroadcastState(broadcastState)
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
		errmsg := fmt.Sprintf("Error marshalling response: %v", err)
		zap.L().Error(errmsg)
		return nil, err
	}
	return broadcastJson, nil
}

// broadcast the global state to auth'd clients
func BroadcastToClients() error {
	return broadcastToClientsWithRuntime(broadcastToClientsRuntimeDefault{}, GetState())
}

func broadcastToClientsWithRuntime(rt broadcastToClientsRuntime, bState structs.AuthBroadcast) error {
	rt.emitToLeak(bState) // vere 3.0
	if !rt.hasAuthSession() {
		return nil
	}
	authJson, err := GetStateJson(bState)
	if err != nil {
		return err
	}
	return rt.broadcastAuth(authJson)
}

// broadcast to unauth clients
func UnauthBroadcast(input []byte) error {
	auth.ClientManager.BroadcastUnauth(input)
	return nil
}

func ReloadUrbits() error {
	zap.L().Info("Reloading ships in broadcast")
	urbits, err := ConstructPierInfo()
	if err != nil {
		return err
	}
	mu.Lock()
	broadcastState.Urbits = urbits
	mu.Unlock()
	return nil
}
