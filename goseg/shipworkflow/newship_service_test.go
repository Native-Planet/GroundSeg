package shipworkflow

import (
	"errors"
	"strings"
	"testing"
	"time"

	"groundseg/docker/events"
	"groundseg/driveresolver"
	"groundseg/structs"
	"groundseg/transition"
)

func resetNewShipServiceSeamsForTest(t *testing.T) {
	t.Helper()
	origResolveDrive := resolveDriveFn
	origEnsureDriveReady := ensureDriveReadyFn
	origNormalizePatp := normalizePatpFn
	origValidatePatp := validatePatpFn
	origProvisionShip := provisionShipFn
	origSleep := newShipSleepFn
	origErrorDelay := newShipErrorDelay
	origRuntime := events.DefaultEventRuntime()
	t.Cleanup(func() {
		resolveDriveFn = origResolveDrive
		ensureDriveReadyFn = origEnsureDriveReady
		normalizePatpFn = origNormalizePatp
		validatePatpFn = origValidatePatp
		provisionShipFn = origProvisionShip
		newShipSleepFn = origSleep
		newShipErrorDelay = origErrorDelay
		events.SetDefaultEventRuntime(origRuntime)
	})
}

func testNewShipRuntimeWithBuffer(t *testing.T, size int) events.EventRuntime {
	t.Helper()
	runtime := events.NewEventRuntimeWithBuffer(size)
	events.SetDefaultEventRuntime(runtime)
	return runtime
}

func readNewShipTransition(t *testing.T, runtime events.EventRuntime) structs.NewShipTransition {
	t.Helper()
	select {
	case event := <-runtime.NewShipTransitions():
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for new-ship transition")
		return structs.NewShipTransition{}
	}
}

func TestResetNewShipUsesCanonicalTransitionKeys(t *testing.T) {
	resetNewShipServiceSeamsForTest(t)
	runtime := testNewShipRuntimeWithBuffer(t, 8)

	if err := ResetNewShip(); err != nil {
		t.Fatalf("ResetNewShip returned error: %v", err)
	}

	if got := readNewShipTransition(t, runtime).Type; got != string(transition.NewShipTransitionBootStage) {
		t.Fatalf("expected first reset event type %q, got %q", transition.NewShipTransitionBootStage, got)
	}
	if got := readNewShipTransition(t, runtime).Type; got != string(transition.NewShipTransitionPatp) {
		t.Fatalf("expected second reset event type %q, got %q", transition.NewShipTransitionPatp, got)
	}
	if got := readNewShipTransition(t, runtime).Type; got != string(transition.NewShipTransitionError) {
		t.Fatalf("expected third reset event type %q, got %q", transition.NewShipTransitionError, got)
	}
}

func TestHandleNewShipBootPreflightErrorUsesCanonicalErrorTransition(t *testing.T) {
	resetNewShipServiceSeamsForTest(t)
	runtime := testNewShipRuntimeWithBuffer(t, 8)
	newShipSleepFn = func(time.Duration) {}
	newShipErrorDelay = 0
	validatePatpFn = func(string) bool { return false }
	normalizePatpFn = func(value string) string { return value }

	err := HandleNewShipBoot(structs.WsNewShipPayload{
		Payload: structs.WsNewShipAction{
			Patp: "badpatp",
		},
	})
	if err == nil {
		t.Fatal("expected preflight validation failure")
	}

	first := readNewShipTransition(t, runtime)
	if first.Type != string(transition.NewShipTransitionError) {
		t.Fatalf("expected error transition type %q, got %q", transition.NewShipTransitionError, first.Type)
	}
	if !strings.Contains(first.Event, "Invalid @p provided") {
		t.Fatalf("unexpected validation event payload: %q", first.Event)
	}
	second := readNewShipTransition(t, runtime)
	if second.Type != string(transition.NewShipTransitionError) || second.Event != "" {
		t.Fatalf("expected error transition clear event, got %#v", second)
	}
}

func TestHandleNewShipBootAsyncProvisionFailureUsesCanonicalErrorTransition(t *testing.T) {
	resetNewShipServiceSeamsForTest(t)
	runtime := testNewShipRuntimeWithBuffer(t, 8)
	newShipSleepFn = func(time.Duration) {}
	newShipErrorDelay = 0
	normalizePatpFn = func(value string) string { return value }
	validatePatpFn = func(string) bool { return true }
	resolveDriveFn = func(string) (driveresolver.Resolution, error) { return driveresolver.Resolution{}, nil }
	ensureDriveReadyFn = func(resolution driveresolver.Resolution) (driveresolver.Resolution, error) { return resolution, nil }
	provisionShipFn = func(string, structs.WsNewShipPayload, string) error { return errors.New("provision failed") }

	err := HandleNewShipBoot(structs.WsNewShipPayload{
		Payload: structs.WsNewShipAction{
			Patp: "zod",
		},
	})
	if err != nil {
		t.Fatalf("expected boot preflight to succeed, got: %v", err)
	}

	first := readNewShipTransition(t, runtime)
	if first.Type != string(transition.NewShipTransitionError) {
		t.Fatalf("expected error transition type %q, got %q", transition.NewShipTransitionError, first.Type)
	}
	if !strings.Contains(first.Event, "provision failed") {
		t.Fatalf("unexpected async provisioning error payload: %q", first.Event)
	}
	second := readNewShipTransition(t, runtime)
	if second.Type != string(transition.NewShipTransitionError) || second.Event != "" {
		t.Fatalf("expected async error transition clear event, got %#v", second)
	}
}
