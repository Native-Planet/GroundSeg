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
	dockerOrchestration "groundseg/docker/orchestration"
	"groundseg/internal/seams"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"
)

type RectifyRuntime struct {
	EventRuntime events.EventBroker
	StateRuntime *broadcast.BroadcastStateRuntime
	dockerOrchestration.RuntimeContainerOps
	dockerOrchestration.RuntimeUrbitOps
	dockerOrchestration.RuntimeSnapshotOps
	dockerOrchestration.RuntimeStartupOps
}

var rectifyStateRuntimeFactory = func() *broadcast.BroadcastStateRuntime {
	return broadcast.DefaultBroadcastStateRuntime()
}

func newRectifyRuntime(stateRuntime *broadcast.BroadcastStateRuntime) RectifyRuntime {
	if stateRuntime == nil {
		stateRuntime = rectifyStateRuntimeFactory()
	}
	orchestrationRuntime := dockerOrchestration.NewRuntime()
	return RectifyRuntime{
		EventRuntime:        events.NewRuntimeBoundBroker(events.DefaultEventRuntime()),
		StateRuntime:        stateRuntime,
		RuntimeContainerOps: orchestrationRuntime.RuntimeContainerOps,
		RuntimeUrbitOps:     orchestrationRuntime.RuntimeUrbitOps,
		RuntimeSnapshotOps:  orchestrationRuntime.RuntimeSnapshotOps,
		RuntimeStartupOps: dockerOrchestration.RuntimeStartupOps{
			UpdateConfigTypedFn: orchestrationRuntime.RuntimeStartupOps.UpdateConfigTypedFn,
		},
	}
}

func NewRectifyRuntime(overrides ...RectifyRuntime) RectifyRuntime {
	return NewRectifyRuntimeWithStateRuntime(nil, overrides...)
}

func NewRectifyRuntimeWithStateRuntime(stateRuntime *broadcast.BroadcastStateRuntime, overrides ...RectifyRuntime) RectifyRuntime {
	runtime := newRectifyRuntime(stateRuntime)
	if len(overrides) == 0 {
		return runtime
	}
	return seams.Merge(runtime, overrides[0])
}

func resolveRectifyRuntime(overrides ...RectifyRuntime) (RectifyRuntime, error) {
	runtime := NewRectifyRuntime(overrides...)
	if err := runtime.validate(); err != nil {
		return runtime, err
	}
	return runtime, nil
}

func (runtime RectifyRuntime) UpdateConfig(opts ...config.ConfigUpdateOption) error {
	if runtime.UpdateConfigTypedFn == nil {
		return fmt.Errorf("rectify runtime missing update config callback")
	}
	return runtime.UpdateConfigTypedFn(opts...)
}

func (runtime RectifyRuntime) validate() error {
	if runtime.StateRuntime == nil {
		return seams.MissingRuntimeDependency("rectify runtime", "missing broadcast state")
	}
	if runtime.EventRuntime == nil {
		return seams.MissingRuntimeDependency("rectify runtime", "missing event runtime")
	}
	if err := seams.NewCallbackRequirementsWithGroups("rectify").ValidateCallbacks(runtime, "rectify runtime"); err != nil {
		return seams.MissingRuntimeDependency("rectify runtime", err.Error())
	}
	return nil
}

func UrbitTransitionHandlerWithContextAndRuntime(ctx context.Context, runtime RectifyRuntime) error {
	runtimeResolved, err := resolveRectifyRuntime(runtime)
	if err != nil {
		return err
	}
	runtime = runtimeResolved
	return runTransitionEventLoop(
		ctx,
		"urbit",
		transition.TransitionPublishStrict,
		runtime.EventRuntime.UrbitTransitions(),
		func(event structs.UrbitTransition) broadcast.BroadcastTransition {
			return urbitTransitionCommand{event: event}
		},
		runtime.StateRuntime,
	)
}

func NewShipTransitionHandlerWithContextAndRuntime(ctx context.Context, runtime RectifyRuntime) error {
	runtimeResolved, err := resolveRectifyRuntime(runtime)
	if err != nil {
		return err
	}
	runtime = runtimeResolved
	return runTransitionEventLoop(
		ctx,
		"new ship",
		transition.TransitionPublishStrict,
		runtime.EventRuntime.NewShipTransitions(),
		func(event structs.NewShipTransition) broadcast.BroadcastTransition {
			return newShipTransitionCommand{event: event}
		},
		runtime.StateRuntime,
	)
}

func RectifyUrbitWithContext(ctx context.Context) error {
	runtime := NewRectifyRuntime()
	runtimeResolved, err := resolveRectifyRuntime(runtime)
	if err != nil {
		return err
	}
	runtime = runtimeResolved
	return runTransitionEventLoop(
		ctx,
		"startram",
		transition.TransitionPublishStrict,
		startram.Events(),
		func(event structs.Event) broadcast.BroadcastTransition {
			transitionType := transition.EventType(event.Type)
			switcher, ok := rectifyStartramTransitionRouters[transitionType]
			if !ok {
				return nil
			}
			return switcher(runtime, event)
		},
		runtime.StateRuntime,
	)
}

func publishUrbitServiceRegistrationTransitionWithRuntime(patp string, serviceCreated bool, runtime *broadcast.BroadcastStateRuntime) error {
	return applyTransitionUpdate("urbit", urbitServiceRegistrationTransitionCommand{
		patp:          patp,
		serviceStatus: serviceCreated,
	}, transition.TransitionPublishBestEffort, runtime)
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

func SystemTransitionHandlerWithContextAndRuntime(ctx context.Context, runtime RectifyRuntime) error {
	runtimeResolved, err := resolveRectifyRuntime(runtime)
	if err != nil {
		return err
	}
	runtime = runtimeResolved
	publishPolicy := transition.TransitionPublishBestEffort
	return runTransitionEventLoop(ctx, "system", publishPolicy, runtime.EventRuntime.SystemTransitions(), func(event structs.SystemTransition) broadcast.BroadcastTransition {
		return systemTransitionCommand{event: event}
	}, runtime.StateRuntime)
}
