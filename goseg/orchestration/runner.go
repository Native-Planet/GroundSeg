package orchestration

import (
	"groundseg/lifecycle"
	"time"
)

type TransitionPolicy struct {
	ClearDelay time.Duration
	Sleep      func(time.Duration)
}

func NewTransitionPolicy(clearDelay time.Duration, sleep func(time.Duration)) TransitionPolicy {
	return TransitionPolicy{
		ClearDelay: clearDelay,
		Sleep:      sleep,
	}
}

func (p TransitionPolicy) Cleanup(cleanup func()) {
	if p.ClearDelay > 0 {
		p.Sleep(p.ClearDelay)
	}
	cleanup()
}

func RunPhases(
	steps []lifecycle.Step,
	emit func(lifecycle.Phase),
	onError func(lifecycle.Phase, error),
	onSuccess func(),
	onCleanup func(),
) error {
	runner := lifecycle.Runner{
		Emit:      emit,
		OnError:   onError,
		OnSuccess: onSuccess,
		OnCleanup: onCleanup,
	}
	return runner.Run(steps...)
}

func RunSinglePhase(
	phase lifecycle.Phase,
	operation func() error,
	emit func(lifecycle.Phase),
	onError func(lifecycle.Phase, error),
	onSuccess func(),
	onCleanup func(),
) error {
	return RunPhases(
		[]lifecycle.Step{{Phase: phase, Run: operation}},
		emit,
		onError,
		onSuccess,
		onCleanup,
	)
}
