package transitionlifecycle

import (
	"time"

	"groundseg/orchestration"
)

// LifecyclePlan defines typed transition emission and cleanup behavior.
type LifecyclePlan[E comparable] struct {
	EmitStart    bool
	StartEvent   E
	SuccessEvent E
	EmitSuccess  bool
	ErrorEvent   func(error) E
	ClearEvent   E
	ClearDelay   time.Duration
}

// LifecycleStep represents one optional emitted step and optional execution unit.
type LifecycleStep[E comparable] struct {
	Event    E
	EmitWhen func() bool
	Run      func() error
}

// Runtime carries transition emission and timing behavior.
type Runtime[E comparable] struct {
	Emit  func(E)
	Sleep func(time.Duration)
}

// Reducer applies one event payload to typed transition state.
type Reducer[K comparable, T any, E any] func(*T, E) bool

// RunLifecycle executes a transition plan and applies optional steps.
func RunLifecycle[E comparable](runtime Runtime[E], plan LifecyclePlan[E], steps ...LifecycleStep[E]) error {
	emit := runtime.Emit
	if emit == nil {
		emit = func(E) {}
	}

	sleepFn := runtime.Sleep
	if sleepFn == nil {
		sleepFn = time.Sleep
	}

	if plan.ErrorEvent == nil {
		plan.ErrorEvent = func(error) E {
			var zero E
			return zero
		}
	}

	if plan.EmitStart {
		emit(plan.StartEvent)
	}

	policy := orchestration.NewTransitionPolicy(plan.ClearDelay, sleepFn)
	defer policy.Cleanup(func() {
		emit(plan.ClearEvent)
	})

	for _, step := range steps {
		var zero E
		if step.Event != zero && (step.EmitWhen == nil || step.EmitWhen()) {
			emit(step.Event)
		}
		if step.Run == nil {
			continue
		}
		if err := step.Run(); err != nil {
			emit(plan.ErrorEvent(err))
			return err
		}
	}

	if plan.EmitSuccess {
		emit(plan.SuccessEvent)
	}

	return nil
}

// ApplyReducer applies the reducer associated with key to target state.
func ApplyReducer[K comparable, T any, E any](reducers map[K]Reducer[K, T, E], target *T, key K, event E) bool {
	if target == nil {
		return false
	}
	if reducer, ok := reducers[key]; ok {
		return reducer(target, event)
	}
	return false
}
