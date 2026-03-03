package rectify

import (
	"fmt"
	"groundseg/broadcast"
	"groundseg/internal/transitionlifecycle"
	"groundseg/shipworkflow"
	"groundseg/structs"
	"groundseg/transition"

	"go.uber.org/zap"
)

type urbitTransitionReducerDescriptor struct {
	transition transition.UrbitTransitionType
	setString  func(*structs.UrbitTransitionBroadcast, string)
	setInt     func(*structs.UrbitTransitionBroadcast, int)
}

var urbitTransitionReducerDescriptors = []urbitTransitionReducerDescriptor{
	{transition: transition.UrbitTransitionChop, setString: func(state *structs.UrbitTransitionBroadcast, value string) {
		state.Chop = value
	}},
	{transition: transition.UrbitTransitionShipCompressed, setInt: func(state *structs.UrbitTransitionBroadcast, value int) {
		state.ShipCompressed = value
	}},
	{transition: transition.UrbitTransitionBucketCompressed, setInt: func(state *structs.UrbitTransitionBroadcast, value int) {
		state.BucketCompressed = value
	}},
	{transition: transition.UrbitTransitionPenpaiCompanion, setString: func(state *structs.UrbitTransitionBroadcast, value string) {
		state.PenpaiCompanion = value
	}},
	{transition: transition.UrbitTransitionGallseg, setString: func(state *structs.UrbitTransitionBroadcast, value string) {
		state.Gallseg = value
	}},
	{transition: transition.UrbitTransitionDeleteService, setString: func(state *structs.UrbitTransitionBroadcast, value string) {
		state.StartramServices = value
	}},
	{transition: transition.UrbitTransitionLocalTlonBackupsEnabled, setString: func(state *structs.UrbitTransitionBroadcast, value string) {
		state.LocalTlonBackupsEnabled = value
	}},
	{transition: transition.UrbitTransitionRemoteTlonBackupsEnabled, setString: func(state *structs.UrbitTransitionBroadcast, value string) {
		state.RemoteTlonBackupsEnabled = value
	}},
	{transition: transition.UrbitTransitionLocalTlonBackup, setString: func(state *structs.UrbitTransitionBroadcast, value string) {
		state.LocalTlonBackup = value
	}},
	{transition: transition.UrbitTransitionLocalTlonBackupSchedule, setString: func(state *structs.UrbitTransitionBroadcast, value string) {
		state.LocalTlonBackupSchedule = value
	}},
	{transition: transition.UrbitTransitionHandleRestoreTlonBackup, setString: func(state *structs.UrbitTransitionBroadcast, value string) {
		state.HandleRestoreTlonBackup = value
	}},
	{transition: transition.UrbitTransitionServiceRegistrationStatus, setString: func(state *structs.UrbitTransitionBroadcast, value string) {
		state.ServiceRegistrationStatus = value
	}},
}

var urbitTransitionReducers = func() map[transition.UrbitTransitionType]transitionlifecycle.Reducer[transition.UrbitTransitionType, structs.UrbitTransitionBroadcast, structs.UrbitTransition] {
	reducers := shipworkflow.UrbitTransitionReducerMap()
	for _, descriptor := range urbitTransitionReducerDescriptors {
		switch {
		case descriptor.setString != nil:
			setString := descriptor.setString
			reducers[descriptor.transition] = func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
				setString(state, event.Event)
				return true
			}
		case descriptor.setInt != nil:
			setInt := descriptor.setInt
			reducers[descriptor.transition] = func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
				setInt(state, event.Value)
				return true
			}
		}
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

type rectifyStartramTransitionRouter func(RectifyRuntime, structs.Event) broadcast.BroadcastTransition

var rectifyStartramTransitionRouters = map[transition.EventType]rectifyStartramTransitionRouter{
	transition.StartramTransitionRestart:  newStartramServiceTransitionRouter(),
	transition.StartramTransitionEndpoint: newStartramServiceTransitionRouter(),
	transition.StartramTransitionToggle:   newStartramServiceTransitionRouter(),
	transition.StartramTransitionRegister: newStartramServiceTransitionRouter(),
	transition.StartramTransitionRetrieve: func(runtime RectifyRuntime, _ structs.Event) broadcast.BroadcastTransition {
		return startramRetrieveTransition{reconciler: NewStartramRetrieveReconciler(runtime)}
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

type urbitTransitionCommand struct {
	event structs.UrbitTransition
}

func (command urbitTransitionCommand) Apply(current *structs.AuthBroadcast) error {
	return UrbitTransitionApplier{}.Apply(current, command.event)
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

type urbitServiceRegistrationTransitionCommand struct {
	patp          string
	serviceStatus bool
}

func (command urbitServiceRegistrationTransitionCommand) Apply(current *structs.AuthBroadcast) error {
	publishUrbitServiceRegistrationTransitionWithCurrentState(current, command.patp, command.serviceStatus)
	return nil
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
