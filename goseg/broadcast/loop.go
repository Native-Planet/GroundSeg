package broadcast

import (
	"errors"
	"fmt"
	"groundseg/leak"
	"groundseg/session"
	"groundseg/structs"
	"sync"
	"time"

	"go.uber.org/zap"
)

var broadcastInterval = 1 * time.Second

var (
	errBroadcastLoopRuntimeRequired = errors.New("broadcast loop runtime is required")
	errBroadcastLoopAlreadyRunning  = errors.New("broadcast loop is already running")
)

type broadcastLoopRuntime struct {
	stateRuntime         *broadcastStateRuntime
	getClientManagerFn   func() *structs.ClientManager
	getLickStatusesFn    func() map[string]leak.LickStatus
	preserveSystemFn     func(structs.AuthBroadcast, structs.System) structs.System
	preserveUrbitsFn     func(structs.AuthBroadcast, map[string]structs.Urbit) map[string]structs.Urbit
	preserveProfileFn    func(structs.AuthBroadcast, structs.Profile) structs.Profile
	updateBroadcastFn    func(structs.AuthBroadcast) error
	broadcastToClientsFn func() error
	tickErrorFn          func(error)
	tickInterval         time.Duration
	loopController       *broadcastLoopController
}

type broadcastLoopController struct {
	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
}

var (
	defaultBroadcastLoopControllerMu sync.RWMutex
	defaultBroadcastLoopController   = newBroadcastLoopController()
)

func newBroadcastLoopController() *broadcastLoopController {
	return &broadcastLoopController{}
}

func defaultLoopController() *broadcastLoopController {
	defaultBroadcastLoopControllerMu.RLock()
	defer defaultBroadcastLoopControllerMu.RUnlock()
	return defaultBroadcastLoopController
}

func SetDefaultBroadcastLoopController(controller *broadcastLoopController) *broadcastLoopController {
	if controller == nil {
		controller = newBroadcastLoopController()
	}
	defaultBroadcastLoopControllerMu.Lock()
	defaultBroadcastLoopController = controller
	defaultBroadcastLoopControllerMu.Unlock()
	return controller
}

func ResetDefaultBroadcastLoopController() *broadcastLoopController {
	return SetDefaultBroadcastLoopController(newBroadcastLoopController())
}

func newBroadcastLoopRuntimeWithController(
	stateRuntime *broadcastStateRuntime,
	controller *broadcastLoopController,
) *broadcastLoopRuntime {
	if stateRuntime == nil {
		return nil
	}
	return &broadcastLoopRuntime{
		stateRuntime:         stateRuntime,
		getClientManagerFn:   session.GetClientManager,
		getLickStatusesFn:    leak.GetLickStatuses,
		preserveSystemFn:     PreserveSystemTransitions,
		preserveUrbitsFn:     PreserveUrbitsTransitions,
		preserveProfileFn:    PreserveProfileTransitions,
		updateBroadcastFn:    func(next structs.AuthBroadcast) error { return stateRuntime.UpdateBroadcast(next) },
		broadcastToClientsFn: BroadcastToClients,
		tickErrorFn:          func(err error) { zap.L().Error(fmt.Sprintf("broadcast tick failed: %v", err)) },
		tickInterval:         broadcastInterval,
		loopController:       controller,
	}
}

func newBroadcastLoopRuntime(stateRuntime *broadcastStateRuntime) *broadcastLoopRuntime {
	return newBroadcastLoopRuntimeWithController(stateRuntime, nil)
}

func defaultBroadcastLoopRuntime() *broadcastLoopRuntime {
	return newBroadcastLoopRuntimeWithController(DefaultBroadcastStateRuntime(), defaultLoopController())
}

func newTestBroadcastLoopRuntime(overrides func(runtime *broadcastLoopRuntime)) *broadcastLoopRuntime {
	rt := defaultBroadcastLoopRuntime()
	overrides(rt)
	return rt
}

func StartBroadcastLoop() error {
	return StartBroadcastLoopWithRuntime(defaultBroadcastLoopRuntime())
}

func StartBroadcastLoopWithRuntime(runtime *broadcastLoopRuntime) error {
	if runtime == nil || runtime.stateRuntime == nil {
		return errBroadcastLoopRuntimeRequired
	}

	stop := make(chan struct{})
	controller := runtime.loopController
	if controller == nil {
		controller = defaultLoopController()
	}

	controller.mu.Lock()
	if controller.running {
		controller.mu.Unlock()
		return errBroadcastLoopAlreadyRunning
	}
	controller.running = true
	controller.stopCh = stop
	controller.mu.Unlock()

	go runBroadcastLoop(runtime, stop)
	return nil
}

func StopBroadcastLoop() {
	stopBroadcastLoop(defaultLoopController())
}

func StopBroadcastLoopWithRuntime(runtime *broadcastLoopRuntime) {
	controller := defaultLoopController()
	if runtime != nil && runtime.loopController != nil {
		controller = runtime.loopController
	}
	stopBroadcastLoop(controller)
}

func stopBroadcastLoop(controller *broadcastLoopController) {
	if controller == nil {
		return
	}

	controller.mu.Lock()
	defer controller.mu.Unlock()
	if !controller.running || controller.stopCh == nil {
		return
	}
	close(controller.stopCh)
	controller.running = false
	controller.stopCh = nil
}

func runBroadcastLoop(runtime *broadcastLoopRuntime, stop <-chan struct{}) {
	if runtime == nil || runtime.stateRuntime == nil {
		return
	}

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

func runBroadcastTick() error {
	return runBroadcastTickWithRuntime(defaultBroadcastLoopRuntime())
}

func runBroadcastTickWithRuntime(rt *broadcastLoopRuntime) error {
	if rt == nil || rt.stateRuntime == nil {
		return fmt.Errorf("run broadcast tick: %w", ErrBroadcastRuntimeRequired)
	}
	stateRuntime := rt.stateRuntime
	if !shouldRunBroadcastTick(rt) {
		return nil
	}
	nextState, err := buildBroadcastTickState(rt, stateRuntime)
	if err != nil {
		if persistErr := stateRuntime.AddSystemTransitionError(err.Error()); persistErr != nil {
			return errors.Join(err, fmt.Errorf("record system transition error: %w", persistErr))
		}
		return err
	}
	if err := rt.updateBroadcastFn(nextState); err != nil {
		return fmt.Errorf("persist broadcast state: %w", err)
	}
	if err := rt.broadcastToClientsFn(); err != nil {
		return fmt.Errorf("broadcast tick publish to clients: %w", err)
	}

	return nil
}

func shouldRunBroadcastTick(rt *broadcastLoopRuntime) bool {
	if rt == nil {
		return false
	}
	cm := rt.getClientManagerFn()
	if cm == nil {
		return false
	}
	return cm.HasAuthSession() || len(rt.getLickStatusesFn()) > 0
}

func buildBroadcastTickState(rt *broadcastLoopRuntime, stateRuntime *broadcastStateRuntime) (structs.AuthBroadcast, error) {
	if rt == nil || stateRuntime == nil {
		return structs.AuthBroadcast{}, fmt.Errorf("build broadcast tick state: %w", ErrBroadcastRuntimeRequired)
	}
	currentState := stateRuntime.GetState()
	systemInfo := stateRuntime.collectSystemInfo()
	pierInfo, err := stateRuntime.collectPierInfo(currentState.Urbits, stateRuntime.GetScheduledPack)
	if err != nil {
		return structs.AuthBroadcast{}, fmt.Errorf("unable to build pier info: %w", err)
	}
	if pierInfo == nil {
		pierInfo = make(map[string]structs.Urbit)
	}
	appsInfo := stateRuntime.collectAppsInfo()
	profileInfo := stateRuntime.collectProfileInfo(currentState.Profile.Startram.Info.Regions)

	nextState := currentState
	nextState.System = rt.preserveSystemFn(nextState, systemInfo)
	nextState.Urbits = rt.preserveUrbitsFn(nextState, pierInfo)
	nextState.Apps = appsInfo
	nextState.Profile = rt.preserveProfileFn(nextState, profileInfo)
	return nextState, nil
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
