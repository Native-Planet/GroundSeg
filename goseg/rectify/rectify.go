package rectify

// this package is for watching event channels and rectifying mismatches
// between the desired and actual state, creating broadcast transitions,
// and anything else that needs to be done asyncronously

import (
	"context"
	"fmt"
	"groundseg/broadcast"
	"groundseg/config"
	"groundseg/docker/events"
	"groundseg/internal/seams"
	"groundseg/internal/transitionlifecycle"
	"groundseg/shipworkflow"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"

	"go.uber.org/zap"
)

type rectifyConfigRuntime struct {
	StartramSettingsSnapshotFn func() config.StartramSettings
	GetStartramConfigFn        func() structs.StartramRetrieve
	UpdateConfTypedFn          func(...config.ConfUpdateOption) error
}

type rectifyContainerRuntime struct {
	GetContainerStateFn func() map[string]structs.ContainerState
}

type rectifyUrbitConfigRuntime struct {
	UrbitConfAllFn    func() map[string]structs.UrbitDocker
	UrbitConfFn       func(string) structs.UrbitDocker
	LoadUrbitConfigFn func(string) error
}

type rectifyPersistenceRuntime struct {
	UpdateUrbitFn               func(string, func(*structs.UrbitDocker) error) error
	UpdateUrbitNetworkConfigFn  func(string, func(*structs.UrbitNetworkConfig) error) error
	UpdateUrbitWebConfigFn      func(string, func(*structs.UrbitWebConfig) error) error
	UpdateUrbitRuntimeConfigFn  func(string, func(*structs.UrbitRuntimeConfig) error) error
	UpdateUrbitScheduleConfigFn func(string, func(*structs.UrbitScheduleConfig) error) error
}

type RectifyRuntime struct {
	rectifyConfigRuntime
	rectifyContainerRuntime
	rectifyUrbitConfigRuntime
	rectifyPersistenceRuntime
}

func NewRectifyRuntime() RectifyRuntime {
	return NewRectifyRuntimeWithDependencies(defaultRectifyRuntime())
}

func NewRectifyRuntimeWithDependencies(overrides RectifyRuntime) RectifyRuntime {
	runtime := defaultRectifyRuntime()
	runtime.rectifyConfigRuntime = seams.Merge(runtime.rectifyConfigRuntime, overrides.rectifyConfigRuntime)
	runtime.rectifyContainerRuntime = seams.Merge(runtime.rectifyContainerRuntime, overrides.rectifyContainerRuntime)
	runtime.rectifyUrbitConfigRuntime = seams.Merge(runtime.rectifyUrbitConfigRuntime, overrides.rectifyUrbitConfigRuntime)
	runtime.rectifyPersistenceRuntime = seams.Merge(runtime.rectifyPersistenceRuntime, overrides.rectifyPersistenceRuntime)
	return runtime
}

func defaultRectifyRuntime() RectifyRuntime {
	return RectifyRuntime{
		rectifyConfigRuntime: rectifyConfigRuntime{
			StartramSettingsSnapshotFn: config.StartramSettingsSnapshot,
			GetStartramConfigFn:        config.GetStartramConfig,
			UpdateConfTypedFn:          config.UpdateConfTyped,
		},
		rectifyContainerRuntime: rectifyContainerRuntime{
			GetContainerStateFn: config.GetContainerState,
		},
		rectifyUrbitConfigRuntime: rectifyUrbitConfigRuntime{
			UrbitConfAllFn:    config.UrbitConfAll,
			UrbitConfFn:       config.UrbitConf,
			LoadUrbitConfigFn: config.LoadUrbitConfig,
		},
		rectifyPersistenceRuntime: rectifyPersistenceRuntime{
			UpdateUrbitFn:               config.UpdateUrbit,
			UpdateUrbitNetworkConfigFn:  config.UpdateUrbitNetworkConfig,
			UpdateUrbitWebConfigFn:      config.UpdateUrbitWebConfig,
			UpdateUrbitRuntimeConfigFn:  config.UpdateUrbitRuntimeConfig,
			UpdateUrbitScheduleConfigFn: config.UpdateUrbitScheduleConfig,
		},
	}
}

func (runtime RectifyRuntime) startramSettings() config.StartramSettings {
	if runtime.StartramSettingsSnapshotFn == nil {
		return config.StartramSettings{}
	}
	return runtime.StartramSettingsSnapshotFn()
}

func UrbitTransitionHandler() {
	runTransitionEventLoopWithoutContext("urbit", events.UrbitTransitions(), func(event structs.UrbitTransition) broadcast.BroadcastTransition {
		return urbitTransitionCommand{event: event}
	})
}

func UrbitTransitionHandlerWithContext(ctx context.Context) error {
	return runTransitionEventLoop(ctx, "urbit", events.UrbitTransitions(), func(event structs.UrbitTransition) broadcast.BroadcastTransition {
		return urbitTransitionCommand{event: event}
	})
}

type urbitTransitionCommand struct {
	event structs.UrbitTransition
}

func (command urbitTransitionCommand) Apply(current *structs.AuthBroadcast) error {
	return UrbitTransitionApplier{}.Apply(current, command.event)
}

var urbitTransitionReducers = func() map[transition.UrbitTransitionType]transitionlifecycle.Reducer[transition.UrbitTransitionType, structs.UrbitTransitionBroadcast, structs.UrbitTransition] {
	reducers := map[transition.UrbitTransitionType]transitionlifecycle.Reducer[transition.UrbitTransitionType, structs.UrbitTransitionBroadcast, structs.UrbitTransition]{
		transition.UrbitTransitionChop: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
			state.Chop = event.Event
			return true
		},
		transition.UrbitTransitionShipCompressed: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
			state.ShipCompressed = event.Value
			return true
		},
		transition.UrbitTransitionBucketCompressed: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
			state.BucketCompressed = event.Value
			return true
		},
		transition.UrbitTransitionPenpaiCompanion: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
			state.PenpaiCompanion = event.Event
			return true
		},
		transition.UrbitTransitionGallseg: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
			state.Gallseg = event.Event
			return true
		},
		transition.UrbitTransitionDeleteService: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
			state.StartramServices = event.Event
			return true
		},
		transition.UrbitTransitionLocalTlonBackupsEnabled: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
			state.LocalTlonBackupsEnabled = event.Event
			return true
		},
		transition.UrbitTransitionRemoteTlonBackupsEnabled: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
			state.RemoteTlonBackupsEnabled = event.Event
			return true
		},
		transition.UrbitTransitionLocalTlonBackup: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
			state.LocalTlonBackup = event.Event
			return true
		},
		transition.UrbitTransitionLocalTlonBackupSchedule: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
			state.LocalTlonBackupSchedule = event.Event
			return true
		},
		transition.UrbitTransitionHandleRestoreTlonBackup: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
			state.HandleRestoreTlonBackup = event.Event
			return true
		},
		transition.UrbitTransitionServiceRegistrationStatus: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
			state.ServiceRegistrationStatus = event.Event
			return true
		},
	}
	for transitionType, reducer := range shipworkflow.UrbitTransitionReducerMap() {
		reducers[transitionType] = reducer
	}
	return reducers
}()

func setUrbitTransition(transitionState *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
	return transitionlifecycle.ApplyReducer(
		urbitTransitionReducers,
		transitionState,
		transition.UrbitTransitionType(event.Type),
		event,
	)
}

func NewShipTransitionHandler() {
	runTransitionEventLoopWithoutContext("new ship", events.NewShipTransitions(), func(event structs.NewShipTransition) broadcast.BroadcastTransition {
		return newShipTransitionCommand{event: event}
	})
}

func NewShipTransitionHandlerWithContext(ctx context.Context) error {
	return runTransitionEventLoop(ctx, "new ship", events.NewShipTransitions(), func(event structs.NewShipTransition) broadcast.BroadcastTransition {
		return newShipTransitionCommand{event: event}
	})
}

type newShipTransitionCommand struct {
	event structs.NewShipTransition
}

func (command newShipTransitionCommand) Apply(current *structs.AuthBroadcast) error {
	if !setNewShipTransition(&current.NewShip, command.event) {
		zap.L().Warn(fmt.Sprintf("Unrecognized transition: %v", command.event.Type))
	}
	return nil
}

var newShipTransitionReducers = map[transition.NewShipTransitionType]transitionlifecycle.Reducer[transition.NewShipTransitionType, structs.NewShip, structs.NewShipTransition]{
	transition.NewShipTransitionError: func(target *structs.NewShip, event structs.NewShipTransition) bool {
		target.Transition.Error = event.Event
		return true
	},
	transition.NewShipTransitionBootStage: func(target *structs.NewShip, event structs.NewShipTransition) bool {
		// Events
		// starting: setting up docker and config
		// creating: actually create and start the container
		// booting: waiting until +code shows up
		// completed: ready to reset
		// aborted: something went wrong and we ran the cleanup routine
		// <empty>: free for new ship
		target.Transition.BootStage = event.Event
		return true
	},
	transition.NewShipTransitionPatp: func(target *structs.NewShip, event structs.NewShipTransition) bool {
		target.Transition.Patp = event.Event
		return true
	},
	transition.NewShipTransitionFreeError: func(target *structs.NewShip, event structs.NewShipTransition) bool {
		target.Transition.FreeError = event.Event
		return true
	},
}

func setNewShipTransition(target *structs.NewShip, event structs.NewShipTransition) bool {
	return transitionlifecycle.ApplyReducer(
		newShipTransitionReducers,
		target,
		transition.NewShipTransitionType(event.Type),
		event,
	)
}

var systemTransitionReducers = map[transition.SystemTransitionType]transitionlifecycle.Reducer[transition.SystemTransitionType, structs.SystemTransitionBroadcast, structs.SystemTransition]{
	transition.SystemTransitionWifiConnect: func(target *structs.SystemTransitionBroadcast, event structs.SystemTransition) bool {
		target.WifiConnect = event.Event
		return true
	},
	transition.SystemTransitionSwap: func(target *structs.SystemTransitionBroadcast, event structs.SystemTransition) bool {
		target.Swap = event.BoolEvent
		return true
	},
	transition.SystemTransitionBugReport: func(target *structs.SystemTransitionBroadcast, event structs.SystemTransition) bool {
		target.BugReport = event.Event
		return true
	},
	transition.SystemTransitionBugReportError: func(target *structs.SystemTransitionBroadcast, event structs.SystemTransition) bool {
		target.BugReportError = event.Event
		return true
	},
}

func setSystemTransition(target *structs.SystemTransitionBroadcast, event structs.SystemTransition) bool {
	return transitionlifecycle.ApplyReducer(
		systemTransitionReducers,
		target,
		transition.SystemTransitionType(event.Type),
		event,
	)
}

var startramTransitionReducers = map[transition.EventType]transitionlifecycle.Reducer[transition.EventType, structs.StartramTransition, any]{
	transition.StartramTransitionRestart: func(target *structs.StartramTransition, eventData any) bool {
		target.Restart = fmt.Sprintf("%v", eventData)
		return true
	},
	transition.StartramTransitionEndpoint: func(target *structs.StartramTransition, eventData any) bool {
		if eventData == nil {
			target.Endpoint = ""
			return true
		}
		target.Endpoint = fmt.Sprintf("%v", eventData)
		return true
	},
	transition.StartramTransitionToggle: func(target *structs.StartramTransition, eventData any) bool {
		target.Toggle = eventData
		return true
	},
	transition.StartramTransitionRegister: func(target *structs.StartramTransition, eventData any) bool {
		target.Register = eventData
		return true
	},
}

func setStartramTransition(target *structs.StartramTransition, eventType string, eventData any) bool {
	return transitionlifecycle.ApplyReducer(
		startramTransitionReducers,
		target,
		transition.EventType(eventType),
		eventData,
	)
}

func applyTransitionUpdate(context string, transition broadcast.BroadcastTransition) {
	if err := broadcast.ApplyBroadcastTransition(true, transition); err != nil {
		zap.L().Warn(fmt.Sprintf("Unable to publish %s transition update: %v", context, err))
	}
}

func RectifyUrbit() {
	runtime := NewRectifyRuntime()
	runTransitionEventLoopWithoutContext("startram", startram.Events(), func(event structs.Event) broadcast.BroadcastTransition {
		transitionType := transition.EventType(event.Type)
		switcher, ok := rectifyStartramTransitionRouters[transitionType]
		if !ok {
			return nil
		}
		return switcher(runtime, event)
	})
}

func RectifyUrbitWithContext(ctx context.Context) error {
	runtime := NewRectifyRuntime()
	return runTransitionEventLoop(ctx, "startram", startram.Events(), func(event structs.Event) broadcast.BroadcastTransition {
		transitionType := transition.EventType(event.Type)
		switcher, ok := rectifyStartramTransitionRouters[transitionType]
		if !ok {
			return nil
		}
		return switcher(runtime, event)
	})
}

type rectifyStartramTransitionRouter func(RectifyRuntime, structs.Event) broadcast.BroadcastTransition

var rectifyStartramTransitionRouters = map[transition.EventType]rectifyStartramTransitionRouter{
	transition.StartramTransitionRestart: func(runtime RectifyRuntime, event structs.Event) broadcast.BroadcastTransition {
		return startramTransitionCommand{
			event:   event,
			service: NewStartramTransitionService(runtime),
		}
	},
	transition.StartramTransitionEndpoint: func(runtime RectifyRuntime, event structs.Event) broadcast.BroadcastTransition {
		return startramTransitionCommand{
			event:   event,
			service: NewStartramTransitionService(runtime),
		}
	},
	transition.StartramTransitionToggle: func(runtime RectifyRuntime, event structs.Event) broadcast.BroadcastTransition {
		return startramTransitionCommand{
			event:   event,
			service: NewStartramTransitionService(runtime),
		}
	},
	transition.StartramTransitionRegister: func(runtime RectifyRuntime, event structs.Event) broadcast.BroadcastTransition {
		return startramTransitionCommand{
			event:   event,
			service: NewStartramTransitionService(runtime),
		}
	},
	transition.StartramTransitionRetrieve: func(runtime RectifyRuntime, _ structs.Event) broadcast.BroadcastTransition {
		return startramRetrieveTransition{
			reconciler: NewStartramRetrieveReconciler(runtime),
		}
	},
}

type startramTransitionCommand struct {
	event   structs.Event
	service StartramTransitionService
}

func (command startramTransitionCommand) Apply(current *structs.AuthBroadcast) error {
	return command.service.Apply(current, command.event)
}

type startramRetrieveTransition struct {
	reconciler *StartramRetrieveReconciler
}

func (transitionCommand startramRetrieveTransition) Apply(current *structs.AuthBroadcast) error {
	return transitionCommand.reconciler.Reconcile(current)
}

func publishUrbitServiceRegistrationTransition(patp string, serviceCreated bool) {
	applyTransitionUpdate("urbit", urbitServiceRegistrationTransitionCommand{
		patp:          patp,
		serviceStatus: serviceCreated,
	})
}

type urbitServiceRegistrationTransitionCommand struct {
	patp          string
	serviceStatus bool
}

func (command urbitServiceRegistrationTransitionCommand) Apply(current *structs.AuthBroadcast) error {
	publishUrbitServiceRegistrationTransitionWithCurrentState(current, command.patp, command.serviceStatus)
	return nil
}

func publishUrbitServiceRegistrationTransitionWithCurrentState(current *structs.AuthBroadcast, patp string, serviceCreated bool) {
	urbitStruct, ok := current.Urbits[patp]
	if !ok {
		return
	}
	if serviceCreated {
		urbitStruct.Transition.ServiceRegistrationStatus = string(transition.TransitionStatusEmpty)
	} else {
		urbitStruct.Transition.ServiceRegistrationStatus = string(transition.StartramServiceStatusCreating)
	}
	current.Urbits[patp] = urbitStruct
}

func SystemTransitionHandler() {
	if err := SystemTransitionHandlerWithContext(context.Background()); err != nil {
		zap.L().Warn(fmt.Sprintf("system transition handler stopped with error: %v", err))
	}
}

func SystemTransitionHandlerWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-events.SystemTransitions():
			applyTransitionUpdate("system", systemTransitionCommand{event: event})
		}
	}
}

type systemTransitionCommand struct {
	event structs.SystemTransition
}

func (command systemTransitionCommand) Apply(current *structs.AuthBroadcast) error {
	if !setSystemTransition(&current.System.Transition, command.event) {
		zap.L().Warn(fmt.Sprintf("Unrecognized transition: %v", command.event.Type))
	}
	return nil
}

func runTransitionEventLoop[T any](ctx context.Context, label string, ch <-chan T, mapEvent func(T) broadcast.BroadcastTransition) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-ch:
			command := mapEvent(event)
			if command == nil {
				continue
			}
			applyTransitionUpdate(label, command)
		}
	}
}

func runTransitionEventLoopWithoutContext[T any](label string, ch <-chan T, mapEvent func(T) broadcast.BroadcastTransition) {
	if err := runTransitionEventLoop(context.Background(), label, ch, mapEvent); err != nil {
		zap.L().Warn(fmt.Sprintf("%s transition handler stopped with error: %v", label, err))
	}
}
