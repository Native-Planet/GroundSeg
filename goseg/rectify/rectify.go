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
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"

	"go.uber.org/zap"
)

var (
	errRectifyConfigUpdateMissing = errors.New("rectify runtime config update callback is not configured")
)

type RectifyRuntime struct {
	EventRuntime         events.EventBroker
	GetContainerStateFn  func() map[string]structs.ContainerState
	UpdateConfigFn       func(...config.ConfUpdateOption) error
	LoadUrbitConfigFn    func(string) error
	UrbitConfFn          func(string) structs.UrbitDocker
	UrbitConfAllFn       func() map[string]structs.UrbitDocker
	UpdateUrbitSectionFn func(string, config.UrbitConfigSection, any) error
	orchestration.RuntimeHealthOps
}

func newRectifyRuntime() RectifyRuntime {
	orchestrationRuntime := orchestration.NewRuntime()
	return RectifyRuntime{
		EventRuntime:         events.DefaultEventRuntime(),
		GetContainerStateFn:  orchestrationRuntime.GetContainerStateFn,
		UpdateConfigFn:       orchestrationRuntime.UpdateConfig,
		LoadUrbitConfigFn:    orchestrationRuntime.LoadUrbitConfigFn,
		UrbitConfFn:          orchestrationRuntime.UrbitConfFn,
		UrbitConfAllFn:       orchestrationRuntime.UrbitConfAllFn,
		UpdateUrbitSectionFn: orchestrationRuntime.UpdateUrbitSectionFn,
		RuntimeHealthOps:     orchestrationRuntime.RuntimeHealthOps,
	}
}

func NewRectifyRuntime() RectifyRuntime {
	return mergeRectifyRuntime(newRectifyRuntime(), RectifyRuntime{})
}

func DefaultRectifyRuntime() RectifyRuntime {
	return NewRectifyRuntime()
}

func NewRectifyRuntimeWithDependencies(overrides RectifyRuntime) RectifyRuntime {
	return mergeRectifyRuntime(newRectifyRuntime(), overrides)
}

func mergeRectifyRuntime(defaults, overrides RectifyRuntime) RectifyRuntime {
	if overrides.EventRuntime != nil {
		defaults.EventRuntime = overrides.EventRuntime
	}
	if overrides.GetContainerStateFn != nil {
		defaults.GetContainerStateFn = overrides.GetContainerStateFn
	}
	if overrides.UpdateConfigFn != nil {
		defaults.UpdateConfigFn = overrides.UpdateConfigFn
	}
	if overrides.LoadUrbitConfigFn != nil {
		defaults.LoadUrbitConfigFn = overrides.LoadUrbitConfigFn
	}
	if overrides.UrbitConfFn != nil {
		defaults.UrbitConfFn = overrides.UrbitConfFn
	}
	if overrides.UrbitConfAllFn != nil {
		defaults.UrbitConfAllFn = overrides.UrbitConfAllFn
	}
	if overrides.UpdateUrbitSectionFn != nil {
		defaults.UpdateUrbitSectionFn = overrides.UpdateUrbitSectionFn
	}
	if overrides.StartramSettingsSnapshotFn != nil {
		defaults.RuntimeHealthOps.StartramSettingsSnapshotFn = overrides.StartramSettingsSnapshotFn
	}
	if overrides.GetStartramConfigFn != nil {
		defaults.RuntimeHealthOps.GetStartramConfigFn = overrides.GetStartramConfigFn
	}
	if overrides.Check502SettingsSnapshotFn != nil {
		defaults.RuntimeHealthOps.Check502SettingsSnapshotFn = overrides.Check502SettingsSnapshotFn
	}
	if overrides.ConfFn != nil {
		defaults.RuntimeHealthOps.ConfFn = overrides.ConfFn
	}
	if overrides.ShipSettingsSnapshotFn != nil {
		defaults.RuntimeHealthOps.ShipSettingsSnapshotFn = overrides.ShipSettingsSnapshotFn
	}
	if overrides.ShipRuntimeSettingsSnapshotFn != nil {
		defaults.RuntimeHealthOps.ShipRuntimeSettingsSnapshotFn = overrides.ShipRuntimeSettingsSnapshotFn
	}
	return defaults
}

func (runtime RectifyRuntime) UpdateConfig(opts ...config.ConfUpdateOption) error {
	if runtime.UpdateConfigFn == nil {
		return errRectifyConfigUpdateMissing
	}
	return runtime.UpdateConfigFn(opts...)
}

func resolveRectifyRuntime(overrides ...RectifyRuntime) RectifyRuntime {
	if len(overrides) == 0 {
		return DefaultRectifyRuntime()
	}
	return NewRectifyRuntimeWithDependencies(overrides[0])
}

func UrbitTransitionHandlerWithContext(ctx context.Context) error {
	return UrbitTransitionHandlerWithContextAndRuntime(ctx, NewRectifyRuntime())
}

func UrbitTransitionHandlerWithContextAndRuntime(ctx context.Context, runtime RectifyRuntime) error {
	runtime = resolveRectifyRuntime(runtime)
	return runTransitionEventLoop(
		ctx,
		"urbit",
		transition.TransitionPublishStrict,
		runtime.EventRuntime.UrbitTransitions(),
		func(event structs.UrbitTransition) broadcast.BroadcastTransition {
			return urbitTransitionCommand{event: event}
		},
	)
}

func NewShipTransitionHandlerWithContext(ctx context.Context) error {
	return NewShipTransitionHandlerWithContextAndRuntime(ctx, NewRectifyRuntime())
}

func NewShipTransitionHandlerWithContextAndRuntime(ctx context.Context, runtime RectifyRuntime) error {
	runtime = resolveRectifyRuntime(runtime)
	return runTransitionEventLoop(
		ctx,
		"new ship",
		transition.TransitionPublishStrict,
		runtime.EventRuntime.NewShipTransitions(),
		func(event structs.NewShipTransition) broadcast.BroadcastTransition {
			return newShipTransitionCommand{event: event}
		},
	)
}

func RectifyUrbitWithContext(ctx context.Context) error {
	runtime := NewRectifyRuntime()
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
	)
}

func publishUrbitServiceRegistrationTransition(patp string, serviceCreated bool) {
	if err := applyTransitionUpdate("urbit", urbitServiceRegistrationTransitionCommand{
		patp:          patp,
		serviceStatus: serviceCreated,
	}, transition.TransitionPublishBestEffort); err != nil {
		zap.L().Warn(fmt.Sprintf("Failed to publish urbit service registration transition: %v", err))
	}
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
	runtime := NewRectifyRuntime()
	return SystemTransitionHandlerWithContextAndRuntime(ctx, runtime)
}

func SystemTransitionHandlerWithContextAndRuntime(ctx context.Context, runtime RectifyRuntime) error {
	runtime = resolveRectifyRuntime(runtime)
	publishPolicy := transition.TransitionPublishBestEffort
	return runTransitionEventLoop(ctx, "system", publishPolicy, runtime.EventRuntime.SystemTransitions(), func(event structs.SystemTransition) broadcast.BroadcastTransition {
		return systemTransitionCommand{event: event}
	})
}
