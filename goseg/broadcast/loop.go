package broadcast

import (
	"fmt"
	"groundseg/auth"
	"groundseg/leak"
	"groundseg/structs"
	"time"

	"go.uber.org/zap"
)

type broadcastLoopRuntime struct {
	getClientManager     func() *structs.ClientManager
	getLickStatuses      func() map[string]leak.LickStatus
	constructSystemInfo  func() structs.System
	constructPierInfo    func() (map[string]structs.Urbit, error)
	constructAppsInfo    func() structs.Apps
	constructProfileInfo func() structs.Profile
	preserveSystem       func(structs.AuthBroadcast, structs.System) structs.System
	preserveUrbits       func(structs.AuthBroadcast, map[string]structs.Urbit) map[string]structs.Urbit
	preserveProfile      func(structs.AuthBroadcast, structs.Profile) structs.Profile
	updateBroadcast      func(structs.AuthBroadcast)
	broadcastToClients   func() error
}

func newBroadcastLoopRuntime() broadcastLoopRuntime {
	return broadcastLoopRuntime{
		getClientManager:     auth.GetClientManager,
		getLickStatuses:      leak.GetLickStatuses,
		constructSystemInfo:  constructSystemInfo,
		constructPierInfo:    ConstructPierInfo,
		constructAppsInfo:    constructAppsInfo,
		constructProfileInfo: constructProfileInfo,
		preserveSystem:       PreserveSystemTransitions,
		preserveUrbits:       PreserveUrbitsTransitions,
		preserveProfile:      PreserveProfileTransitions,
		updateBroadcast:      UpdateBroadcast,
		broadcastToClients:   BroadcastToClients,
	}
}

func newTestBroadcastLoopRuntime(overrides func(runtime *broadcastLoopRuntime)) broadcastLoopRuntime {
	rt := newBroadcastLoopRuntime()
	overrides(&rt)
	return rt
}

func BroadcastLoop() {
	BroadcastLoopWithRuntime(newBroadcastLoopRuntime())
}

func BroadcastLoopWithRuntime(runtime broadcastLoopRuntime) {
	ticker := time.NewTicker(broadcastInterval)
	if runtime.broadcastToClients == nil {
		runtime.broadcastToClients = BroadcastToClients
	}
	if runtime.getClientManager == nil {
		runtime.getClientManager = auth.GetClientManager
	}
	if runtime.getLickStatuses == nil {
		runtime.getLickStatuses = leak.GetLickStatuses
	}
	if runtime.constructSystemInfo == nil {
		runtime.constructSystemInfo = constructSystemInfo
	}
	if runtime.constructPierInfo == nil {
		runtime.constructPierInfo = ConstructPierInfo
	}
	if runtime.constructAppsInfo == nil {
		runtime.constructAppsInfo = constructAppsInfo
	}
	if runtime.constructProfileInfo == nil {
		runtime.constructProfileInfo = constructProfileInfo
	}
	if runtime.preserveSystem == nil {
		runtime.preserveSystem = PreserveSystemTransitions
	}
	if runtime.preserveUrbits == nil {
		runtime.preserveUrbits = PreserveUrbitsTransitions
	}
	if runtime.preserveProfile == nil {
		runtime.preserveProfile = PreserveProfileTransitions
	}
	if runtime.updateBroadcast == nil {
		runtime.updateBroadcast = UpdateBroadcast
	}

	for {
		select {
		case <-ticker.C:
			runBroadcastTickWithRuntime(runtime)
		}
	}
}

func runBroadcastTick() {
	runBroadcastTickWithRuntime(newBroadcastLoopRuntime())
}

func runBroadcastTickWithRuntime(rt broadcastLoopRuntime) {
	cm := rt.getClientManager()
	if cm == nil {
		return
	}
	if cm.HasAuthSession() || len(rt.getLickStatuses()) > 0 {
		// refresh loop for host info
		systemInfo := rt.constructSystemInfo()

		// pier info
		pierInfo, err := rt.constructPierInfo()
		if err != nil {
			zap.L().Error(fmt.Sprintf("Unable to build pier info: %v", err))
		}
		if pierInfo == nil {
			pierInfo = make(map[string]structs.Urbit)
		}

		// apps info
		appsInfo := rt.constructAppsInfo()

		// profile info
		profileInfo := rt.constructProfileInfo()

		// Retrieve broadcastState
		mu.RLock()
		newState := broadcastState
		mu.RUnlock()

		// Preserve transitions
		systemInfo = rt.preserveSystem(newState, systemInfo)
		pierInfo = rt.preserveUrbits(newState, pierInfo)
		profileInfo = rt.preserveProfile(newState, profileInfo)

		// Update broadcast state
		newState.System = systemInfo
		newState.Urbits = pierInfo
		newState.Apps = appsInfo
		newState.Profile = profileInfo

		rt.updateBroadcast(newState)

		// broadcast
		rt.broadcastToClients()
	}
}

func PreserveProfileTransitions(oldState structs.AuthBroadcast, newProfile structs.Profile) structs.Profile {
	newProfile.Startram.Transition = oldState.Profile.Startram.Transition
	return newProfile
}

func PreserveSystemTransitions(oldState structs.AuthBroadcast, newSystem structs.System) structs.System {
	newSystem.Transition = oldState.System.Transition
	return newSystem
}

func PreserveUrbitsTransitions(oldState structs.AuthBroadcast, newUrbits map[string]structs.Urbit) map[string]structs.Urbit {
	for k, v := range oldState.Urbits {
		urbitStruct, exists := newUrbits[k]
		if !exists {
			urbitStruct = structs.Urbit{}
		}
		urbitStruct.Transition = v.Transition
		newUrbits[k] = urbitStruct
	}
	return newUrbits
}
