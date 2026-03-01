package handler

import (
	"context"
	"groundseg/docker"
	"groundseg/lifecycle"
	"groundseg/orchestration"
	"groundseg/structs"
	"time"
)

var (
	publishUrbitTransitionForWorkflow = docker.PublishUrbitTransition
	sleepForWorkflow                  = time.Sleep
	newTickerForWorkflow              = time.NewTicker
)

func emitUrbitTransition(patp, transitionType, event string) {
	publishUrbitTransitionForWorkflow(structs.UrbitTransition{Patp: patp, Type: transitionType, Event: event})
}

// publishTransitionWithPolicy centralizes delayed-clear semantics used by multiple handlers.
func publishTransitionWithPolicy[T any](publish func(T), event T, clear T, clearDelay time.Duration) {
	publish(event)
	if clearDelay > 0 {
		sleepForWorkflow(clearDelay)
	}
	publish(clear)
}

func runTransitionedOperation(patp, transitionType, startEvent, successEvent string, clearDelay time.Duration, operation func() error) error {
	policy := orchestration.NewTransitionPolicy(clearDelay, sleepForWorkflow)
	return orchestration.RunSinglePhase(
		lifecycle.Phase(startEvent),
		operation,
		func(phase lifecycle.Phase) {
			emitUrbitTransition(patp, transitionType, string(phase))
		},
		func(_ lifecycle.Phase, _ error) {
			emitUrbitTransition(patp, transitionType, "error")
		},
		func() {
			if successEvent != "" {
				emitUrbitTransition(patp, transitionType, successEvent)
			}
		},
		func() {
			policy.Cleanup(func() {
				emitUrbitTransition(patp, transitionType, "")
			})
		},
	)
}

func pollWithTimeout(ctx context.Context, interval time.Duration, condition func() (bool, error)) error {
	ticker := newTickerForWorkflow(interval)
	defer ticker.Stop()
	for {
		done, err := condition()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
