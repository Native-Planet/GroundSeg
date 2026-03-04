package broadcast

import (
	"fmt"
	"groundseg/auth"
	"groundseg/leak"
	"groundseg/structs"
	"sync"
	"time"

	"go.uber.org/zap"
)

var broadcastInterval = 1 * time.Second

type broadcastLoopRuntime struct {
	getClientManagerFn     func() *structs.ClientManager
	getLickStatusesFn      func() map[string]leak.LickStatus
	constructSystemInfoFn  func() structs.System
	constructPierInfoFn    func() (map[string]structs.Urbit, error)
	constructAppsInfoFn    func() structs.Apps
	constructProfileInfoFn func() structs.Profile
	preserveSystemFn       func(structs.AuthBroadcast, structs.System) structs.System
	preserveUrbitsFn       func(structs.AuthBroadcast, map[string]structs.Urbit) map[string]structs.Urbit
	preserveProfileFn      func(structs.AuthBroadcast, structs.Profile) structs.Profile
	updateBroadcastFn      func(structs.AuthBroadcast)
	broadcastToClientsFn   func() error
	tickErrorFn            func(error)
	tickInterval           time.Duration
}

var (
	loopControlMu   sync.Mutex
	loopRunning     bool
	loopStopChannel chan struct{}
)

func newBroadcastLoopRuntime() *broadcastLoopRuntime {
	return &broadcastLoopRuntime{
		getClientManagerFn: auth.GetClientManager,
		getLickStatusesFn:  leak.GetLickStatuses,
		constructSystemInfoFn: func() structs.System {
			return constructSystemInfoWithRuntime(resolveBroadcastStateRuntime())
		},
		constructPierInfoFn: func() (map[string]structs.Urbit, error) {
			resolved := resolveBroadcastStateRuntime()
			state := resolved.GetState()
			return constructPierInfoWithRuntime(resolved, state.Urbits, resolved.GetScheduledPack)
		},
		constructAppsInfoFn: func() structs.Apps {
			return constructAppsInfoWithRuntime(resolveBroadcastStateRuntime())
		},
		constructProfileInfoFn: func() structs.Profile {
			resolved := resolveBroadcastStateRuntime()
			state := resolved.GetState()
			return constructProfileInfoWithRuntime(resolved, state.Profile.Startram.Info.Regions)
		},
		preserveSystemFn:     PreserveSystemTransitions,
		preserveUrbitsFn:     PreserveUrbitsTransitions,
		preserveProfileFn:    PreserveProfileTransitions,
		updateBroadcastFn:    func(next structs.AuthBroadcast) { DefaultBroadcastStateRuntime().UpdateBroadcast(next) },
		broadcastToClientsFn: BroadcastToClients,
		tickErrorFn:          func(err error) { zap.L().Error(fmt.Sprintf("broadcast tick failed: %v", err)) },
		tickInterval:         broadcastInterval,
	}
}

func newTestBroadcastLoopRuntime(overrides func(runtime *broadcastLoopRuntime)) *broadcastLoopRuntime {
	rt := newBroadcastLoopRuntime()
	overrides(rt)
	return rt
}

func BroadcastLoop() {
	StartBroadcastLoop()
}

func BroadcastLoopWithRuntime(runtime *broadcastLoopRuntime) {
	StartBroadcastLoopWithRuntime(runtime)
}

func StartBroadcastLoop() bool {
	return StartBroadcastLoopWithRuntime(newBroadcastLoopRuntime())
}

func StartBroadcastLoopWithRuntime(runtime *broadcastLoopRuntime) bool {
	if runtime == nil {
		runtime = newBroadcastLoopRuntime()
	}

	stop := make(chan struct{})

	loopControlMu.Lock()
	if loopRunning {
		loopControlMu.Unlock()
		return false
	}
	loopRunning = true
	loopStopChannel = stop
	loopControlMu.Unlock()

	go runBroadcastLoop(runtime, stop)
	return true
}

func StopBroadcastLoop() {
	loopControlMu.Lock()
	defer loopControlMu.Unlock()

	if !loopRunning {
		return
	}
	if loopStopChannel != nil {
		close(loopStopChannel)
	}
	loopRunning = false
	loopStopChannel = nil
}

func runBroadcastLoop(runtime *broadcastLoopRuntime, stop <-chan struct{}) {
	tickerDuration := broadcastInterval
	if runtime != nil && runtime.tickInterval > 0 {
		tickerDuration = runtime.tickInterval
	}
	ticker := time.NewTicker(tickerDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := runBroadcastTickWithRuntime(runtime); err != nil {
				if runtime.tickErrorFn != nil {
					runtime.tickErrorFn(err)
				}
			}
		case <-stop:
			return
		}
	}
}

func runBroadcastTick() {
	_ = runBroadcastTickWithRuntime(newBroadcastLoopRuntime())
}

func runBroadcastTickWithRuntime(rt *broadcastLoopRuntime) error {
	if rt == nil {
		rt = newBroadcastLoopRuntime()
	}
	cm := rt.getClientManagerFn()
	if cm == nil {
		return nil
	}
	if cm.HasAuthSession() || len(rt.getLickStatusesFn()) > 0 {
		// refresh loop for host info
		systemInfo := rt.constructSystemInfoFn()

		// pier info
		pierInfo, err := rt.constructPierInfoFn()
		if err != nil {
			zap.L().Error(fmt.Sprintf("Unable to build pier info: %v", err))
			pierInfo = DefaultBroadcastStateRuntime().GetState().Urbits
		}
		if pierInfo == nil {
			pierInfo = make(map[string]structs.Urbit)
		}

		// apps info
		appsInfo := rt.constructAppsInfoFn()

		// profile info
		profileInfo := rt.constructProfileInfoFn()

		// Retrieve broadcastState
		newState := DefaultBroadcastStateRuntime().GetState()

		// Preserve transitions
		systemInfo = rt.preserveSystemFn(newState, systemInfo)
		pierInfo = rt.preserveUrbitsFn(newState, pierInfo)
		profileInfo = rt.preserveProfileFn(newState, profileInfo)

		// Update broadcast state
		newState.System = systemInfo
		newState.Urbits = pierInfo
		newState.Apps = appsInfo
		newState.Profile = profileInfo

		rt.updateBroadcastFn(newState)

		// broadcast
		if err := rt.broadcastToClientsFn(); err != nil {
			return err
		}
	}

	return nil
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
