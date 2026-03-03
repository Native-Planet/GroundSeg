package rectify

// this package is for watching event channels and rectifying mismatches
// between the desired and actual state, creating broadcast transitions,
// and anything else that needs to be done asyncronously

import (
	"context"
	"errors"
	"fmt"
	"groundseg/broadcast"
	"groundseg/config"
	"groundseg/docker/events"
	"groundseg/docker/orchestration"
	"groundseg/internal/transitionlifecycle"
	"groundseg/shipworkflow"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"

	"go.uber.org/zap"
)

var (
	errRectifyStartramSettingsMissing = errors.New("rectify runtime startram settings callback is not configured")
	errRectifyStartramConfigMissing   = errors.New("rectify runtime startram config callback is not configured")
)

type RectifyRuntime struct {
	orchestration.Runtime
}

func NewRectifyRuntime() RectifyRuntime {
	return NewRectifyRuntimeWithDependencies(defaultRectifyRuntime())
}

func NewRectifyRuntimeWithDependencies(overrides RectifyRuntime) RectifyRuntime {
	return RectifyRuntime{Runtime: orchestration.NewRuntime(orchestration.WithRuntimeDependencies(overrides.Runtime))}
}

func defaultRectifyRuntime() RectifyRuntime {
	return RectifyRuntime{Runtime: orchestration.NewRuntime()}
}

func (runtime RectifyRuntime) startramSettings() (config.StartramSettings, error) {
	if runtime.StartramSettingsSnapshotFn == nil {
		return config.StartramSettings{}, errRectifyStartramSettingsMissing
	}
	return runtime.StartramSettingsSnapshotFn(), nil
}

func (runtime RectifyRuntime) startramConfig() (structs.StartramRetrieve, error) {
	if runtime.GetStartramConfigFn == nil {
		return structs.StartramRetrieve{}, errRectifyStartramConfigMissing
	}
	return runtime.GetStartramConfigFn(), nil
}

func UrbitTransitionHandlerWithContext(ctx context.Context) error {
	return runTransitionEventLoop(ctx, "urbit", events.DefaultEventRuntime().UrbitTransitions(), func(event structs.UrbitTransition) broadcast.BroadcastTransition {
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

func NewShipTransitionHandlerWithContext(ctx context.Context) error {
	return runTransitionEventLoop(ctx, "new ship", events.DefaultEventRuntime().NewShipTransitions(), func(event structs.NewShipTransition) broadcast.BroadcastTransition {
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
	transition.StartramTransitionRestart: newStartramServiceTransitionRouter(),
	transition.StartramTransitionEndpoint: newStartramServiceTransitionRouter(),
	transition.StartramTransitionToggle:  newStartramServiceTransitionRouter(),
	transition.StartramTransitionRegister: newStartramServiceTransitionRouter(),
	transition.StartramTransitionRetrieve: func(runtime RectifyRuntime, _ structs.Event) broadcast.BroadcastTransition {
		return startramRetrieveTransition{
			reconciler: NewStartramRetrieveReconciler(runtime),
		}
	},
}

func newStartramServiceTransitionRouter() rectifyStartramTransitionRouter {
	return func(runtime RectifyRuntime, event structs.Event) broadcast.BroadcastTransition {
		return startramTransitionCommand{
			event:   event,
			service: NewStartramTransitionService(runtime),
		}
	}
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

func SystemTransitionHandlerWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-events.DefaultEventRuntime().SystemTransitions():
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
