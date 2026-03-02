package shipworkflow

import (
	"time"

	"groundseg/structs"
	"groundseg/transition"
)

type urbitTransitionTemplate struct {
	transitionType string
	startEvent     string
	successEvent   string
	clearEvent     string
	clearDelay     time.Duration
	err            func(error) string
	emitSuccess    bool
}

func runUrbitTransitionTemplate(patp string, spec urbitTransitionTemplate, steps ...transitionStep[string]) error {
	if spec.err == nil {
		spec.err = func(error) string { return "error" }
	}
	return runUrbitTransition(
		patp,
		spec.transitionType,
		transitionPlan[string]{
			EmitStart:    true,
			StartEvent:   spec.startEvent,
			SuccessEvent: spec.successEvent,
			EmitSuccess:  spec.emitSuccess,
			ErrorEvent:   spec.err,
			ClearEvent:   spec.clearEvent,
			ClearDelay:   spec.clearDelay,
		},
		steps...,
	)
}

type startramTransitionTemplate struct {
	transitionType transition.EventType
	startEvent     structs.Event
	successEvent   structs.Event
	clearEvent     structs.Event
	clearDelay     time.Duration
	err            func(error) structs.Event
	emitSuccess    bool
}

func runStartramTransitionTemplate(runtime startramRuntime, spec startramTransitionTemplate, steps ...transitionStep[structs.Event]) error {
	if spec.err == nil {
		spec.err = func(err error) structs.Event {
			return startramEvent(spec.transitionType, err)
		}
	}
	return runStartramTransitionWithRuntime(
		runtime,
		spec.transitionType,
		transitionPlan[structs.Event]{
			EmitStart:    true,
			StartEvent:   spec.startEvent,
			SuccessEvent: spec.successEvent,
			EmitSuccess:  spec.emitSuccess,
			ErrorEvent:   spec.err,
			ClearEvent:   spec.clearEvent,
			ClearDelay:   spec.clearDelay,
		},
		steps...,
	)
}
