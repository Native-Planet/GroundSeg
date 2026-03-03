package chopsvc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"groundseg/shipworkflow"
	"groundseg/structs"
)

func resetChopSvcSeams() {
	publishUrbitTransitionFn = func(_ context.Context, _ structs.UrbitTransition) error { return nil }
	getShipStatusFn = func([]string) (map[string]string, error) {
		return map[string]string{}, nil
	}
	barExitFn = func(string) error { return nil }
	stopContainerByNameFn = func(string) error { return nil }
	persistShipRuntimeConfigFn = func(_ string, update func(*structs.UrbitRuntimeConfig) error) error {
		conf := structs.UrbitRuntimeConfig{}
		return update(&conf)
	}
	startContainerFn = func(string, string) (structs.ContainerState, error) {
		return structs.ContainerState{}, nil
	}
	forceUpdateContainerStatsFn = func(string) structs.ContainerStats {
		return structs.ContainerStats{}
	}
	waitCompleteFn = WaitComplete
	sleepFn = time.Sleep
	waitCompletePollerFn = shipworkflow.PollWithTimeout
}

func TestChopPierReturnsErrorWhenStatusFetchFails(t *testing.T) {
	t.Cleanup(resetChopSvcSeams)
	getShipStatusFn = func([]string) (map[string]string, error) {
		return nil, errors.New("docker down")
	}
	waitCompleteFn = func(string) error { return nil }
	sleepFn = func(time.Duration) {}

	err := ChopPier("~zod")
	if err == nil {
		t.Fatal("expected status failure")
	}
	if !strings.Contains(err.Error(), "Failed to get ship status for ~zod") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChopPierRunningShipTransitionsToChopAndBackToBoot(t *testing.T) {
	t.Cleanup(resetChopSvcSeams)
	sleepFn = func(time.Duration) {}
	getShipStatusFn = func([]string) (map[string]string, error) {
		return map[string]string{"~zod": "Up 2 minutes"}, nil
	}
	waitCompleteCalls := 0
	waitCompleteFn = func(string) error {
		waitCompleteCalls++
		return nil
	}
	var savedBootStates []string
	persistShipRuntimeConfigFn = func(_ string, update func(*structs.UrbitRuntimeConfig) error) error {
		conf := structs.UrbitRuntimeConfig{}
		if err := update(&conf); err != nil {
			return err
		}
		savedBootStates = append(savedBootStates, conf.BootStatus)
		return nil
	}
	startCalls := 0
	startContainerFn = func(ship, kind string) (structs.ContainerState, error) {
		if ship != "~zod" || kind != "vere" {
			t.Fatalf("unexpected container start args: %s %s", ship, kind)
		}
		startCalls++
		return structs.ContainerState{}, nil
	}
	var events []string
	publishUrbitTransitionFn = func(_ context.Context, t structs.UrbitTransition) error {
		if t.Type == "chop" {
			events = append(events, t.Event)
		}
		return nil
	}

	if err := ChopPier("~zod"); err != nil {
		t.Fatalf("ChopPier failed: %v", err)
	}
	if waitCompleteCalls != 2 {
		t.Fatalf("expected two wait-complete calls, got %d", waitCompleteCalls)
	}
	if startCalls != 2 {
		t.Fatalf("expected two container starts, got %d", startCalls)
	}
	if len(savedBootStates) != 2 || savedBootStates[0] != "chop" || savedBootStates[1] != "boot" {
		t.Fatalf("unexpected boot state persistence: %+v", savedBootStates)
	}
	if !strings.Contains(strings.Join(events, ","), "success") {
		t.Fatalf("expected success event in transitions, got %v", events)
	}
}

func TestWaitCompleteRetriesStatusFailureBeforeSuccess(t *testing.T) {
	t.Cleanup(resetChopSvcSeams)
	polls := 0
	waitCompletePollerFn = func(ctx context.Context, _ time.Duration, condition func() (bool, error)) error {
		for {
			done, err := condition()
			if err != nil {
				return err
			}
			if done {
				return nil
			}
			polls++
			if polls > 5 {
				return ctx.Err()
			}
		}
	}
	statusCalls := 0
	getShipStatusFn = func([]string) (map[string]string, error) {
		statusCalls++
		switch statusCalls {
		case 1:
			return nil, errors.New("temporary")
		case 2:
			return map[string]string{"~bus": "Up"}, nil
		default:
			return map[string]string{"~bus": "Exited"}, nil
		}
	}
	if err := WaitComplete("~bus"); err != nil {
		t.Fatalf("WaitComplete failed: %v", err)
	}
	if statusCalls < 3 {
		t.Fatalf("expected retries before completion, got %d status calls", statusCalls)
	}
}
