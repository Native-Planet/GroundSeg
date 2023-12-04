package broadcast

import (
	"fmt"
	"goseg/auth"
	"goseg/leak"
	"goseg/logger"
	"goseg/structs"
	"time"
)

func BroadcastLoop() {
	ticker := time.NewTicker(broadcastInterval)
	//n := 0
	for {
		select {
		case <-ticker.C:
			//logger.Logger.Warn(fmt.Sprintf("broadcast loop %v", n))
			//n = n + 1
			cm := auth.GetClientManager()
			if cm.HasAuthSession() || len(leak.GetLickStatuses()) > 0 {
				// refresh loop for host info
				systemInfo := constructSystemInfo()

				// pier info
				pierInfo, err := ConstructPierInfo()
				if err != nil {
					logger.Logger.Warn(fmt.Sprintf("Unable to build pier info: %v", err))
				}

				// apps info
				appsInfo := constructAppsInfo()

				// profile info
				profileInfo := constructProfileInfo()

				// Retrieve broadcastState
				mu.RLock()
				newState := broadcastState
				mu.RUnlock()

				// Preserve transitions
				systemInfo = PreserveSystemTransitions(newState, systemInfo)
				pierInfo = PreserveUrbitsTransitions(newState, pierInfo)
				profileInfo = PreserveProfileTransitions(newState, profileInfo)

				// Update broadcast state
				newState.System = systemInfo
				newState.Urbits = pierInfo
				newState.Apps = appsInfo
				newState.Profile = profileInfo

				UpdateBroadcast(newState)

				// broadcast
				BroadcastToClients()
			}
		}
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
