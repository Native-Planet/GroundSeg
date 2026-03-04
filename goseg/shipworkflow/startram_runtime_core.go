package shipworkflow

import (
	"fmt"

	"groundseg/broadcast"
	"groundseg/docker/orchestration"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"
)

type startramRuntime = orchestration.StartramRuntime

func defaultStartramRuntime() startramRuntime {
	return resolveStartramRuntime(orchestration.NewStartramRuntime(
		orchestration.WithStartramServiceLoaders(
			broadcast.GetStartramServices,
			func() error {
				return broadcast.LoadStartramRegionsWithRuntime()
			},
		),
	))
}

func resolveStartramRuntime(runtime startramRuntime) startramRuntime {
	return orchestration.ResolveStartramRuntime(runtime, defaultStartramRuntimeOps())
}

func defaultStartramRuntimeOps() orchestration.RuntimeStartramOps {
	return orchestration.RuntimeStartramOps{
		DispatchUrbitPayloadFn: func(payload structs.WsUrbitPayload) error {
			return ToggleNetwork(payload.Payload.Patp)
		},
		PublishEventFn: startram.PublishEvent,
		RecoverWireguardFleetFn: func(piers []string, deleteMinioClient bool) error {
			rt := orchestration.NewRuntime()
			return RecoverWireguardFleet(NewWireguardRecoveryRuntime(rt), piers, deleteMinioClient)
		},
	}
}

func defaultDispatchUrbitPayload(payload structs.WsUrbitPayload) error {
	return dispatchUrbitPayloadWithRuntime(defaultStartramRuntime(), payload)
}

func dispatchUrbitPayloadWithRuntime(runtime startramRuntime, payload structs.WsUrbitPayload) error {
	runtime = resolveStartramRuntime(runtime)
	if payload.Payload.Action == "" {
		return fmt.Errorf("no urbit action provided")
	}
	if payload.Payload.Patp == "" {
		return fmt.Errorf("no patp provided for urbit action %q", payload.Payload.Action)
	}

	switch payload.Payload.Action {
	case "toggle-network":
		return runtime.DispatchUrbitPayloadFn(payload)
	default:
		return fmt.Errorf("unsupported urbit action in startram dispatch: %q", payload.Payload.Action)
	}
}

func startramEvent(transitionType transition.EventType, data any) structs.Event {
	return structs.Event{
		Type: string(transitionType),
		Data: data,
	}
}

func runStartramTransitionWithRuntime(runtime startramRuntime, transitionType transition.EventType, plan transitionPlan[structs.Event], steps ...transitionStep[structs.Event]) error {
	runtime = resolveStartramRuntime(runtime)
	if plan.ErrorEvent == nil {
		plan.ErrorEvent = func(err error) structs.Event {
			return startramEvent(transitionType, fmt.Sprintf("Error: %v", err))
		}
	}
	return runTransitionLifecycle[structs.Event](
		defaultWorkflowRuntime(),
		func(event structs.Event) error {
			runtime.PublishEventFn(event)
			return nil
		},
		plan,
		steps...,
	)
}
