package rectify

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"groundseg/broadcast"
	"groundseg/docker/events"
	"groundseg/shipworkflow"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/testutil"
	"groundseg/transition"
)

var (
	rectifyTestEnvOnce         sync.Once
	urbitTransitionHandlerOnce sync.Once
	newShipTransitionOnce      sync.Once
	systemTransitionOnce       sync.Once
	rectifyUrbitOnce           sync.Once
	urbitTransitionErrCh       = make(chan error, 1)
	newShipTransitionErrCh     = make(chan error, 1)
	systemTransitionErrCh      = make(chan error, 1)
	rectifyUrbitErrCh          = make(chan error, 1)
	rectifyTestRuntime         = RectifyRuntime{
		EventRuntime: events.NewEventRuntime(),
	}
)

func initializeRectifyTestEnv() {
	rectifyTestEnvOnce.Do(func() {
		// BroadcastToClients can be triggered during transition updates.
		// The loop now publishes to leak asynchronously to avoid test deadlocks.
	})
}

func assertNoAsyncRectifyError(t *testing.T, name string, errCh <-chan error) {
	t.Helper()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("%s returned error: %v", name, err)
		}
	default:
	}
}

func TestUrbitTransitionHandlerUpdatesBroadcastState(t *testing.T) {
	initializeRectifyTestEnv()
	urbitTransitionHandlerOnce.Do(func() {
		go func() {
			urbitTransitionErrCh <- UrbitTransitionHandlerWithContextAndRuntime(context.Background(), rectifyTestRuntime)
		}()
	})

	current := structs.AuthBroadcast{
		Urbits: map[string]structs.Urbit{
			"zod": {},
		},
	}
	broadcast.DefaultBroadcastStateRuntime().UpdateBroadcast(current)

	eventValue := "pack-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	rectifyTestRuntime.EventRuntime.PublishUrbitTransition(context.Background(), structs.UrbitTransition{
		Patp:  "zod",
		Type:  "pack",
		Event: eventValue,
	})

	testutil.WaitForCondition(t, func() bool {
		state := broadcast.DefaultBroadcastStateRuntime().GetState()
		return state.Urbits["zod"].Transition.Pack == eventValue
	}, "urbit pack transition was not applied")
	state := broadcast.DefaultBroadcastStateRuntime().GetState()
	if got := state.Urbits["zod"].Transition.Pack; got != eventValue {
		t.Fatalf("unexpected urbit pack transition value: got %q want %q", got, eventValue)
	}
	assertNoAsyncRectifyError(t, "UrbitTransitionHandlerWithContextAndRuntime", urbitTransitionErrCh)
}

func TestNewShipTransitionHandlerUpdatesBroadcastState(t *testing.T) {
	initializeRectifyTestEnv()
	newShipTransitionOnce.Do(func() {
		go func() {
			newShipTransitionErrCh <- NewShipTransitionHandlerWithContextAndRuntime(context.Background(), rectifyTestRuntime)
		}()
	})

	broadcast.DefaultBroadcastStateRuntime().UpdateBroadcast(structs.AuthBroadcast{})
	bootStage := "booting-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	patp := "~zod-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	rectifyTestRuntime.EventRuntime.PublishNewShipTransition(context.Background(), structs.NewShipTransition{Type: "bootStage", Event: bootStage})
	rectifyTestRuntime.EventRuntime.PublishNewShipTransition(context.Background(), structs.NewShipTransition{Type: "patp", Event: patp})

	testutil.WaitForCondition(t, func() bool {
		state := broadcast.DefaultBroadcastStateRuntime().GetState()
		return state.NewShip.Transition.BootStage == bootStage && state.NewShip.Transition.Patp == patp
	}, "new ship transitions were not applied")
	state := broadcast.DefaultBroadcastStateRuntime().GetState()
	if got := state.NewShip.Transition.BootStage; got != bootStage {
		t.Fatalf("unexpected new-ship boot stage: got %q want %q", got, bootStage)
	}
	if got := state.NewShip.Transition.Patp; got != patp {
		t.Fatalf("unexpected new-ship patp: got %q want %q", got, patp)
	}
	assertNoAsyncRectifyError(t, "NewShipTransitionHandlerWithContextAndRuntime", newShipTransitionErrCh)
}

func TestSystemTransitionHandlerUpdatesBroadcastState(t *testing.T) {
	initializeRectifyTestEnv()
	systemTransitionOnce.Do(func() {
		go func() {
			systemTransitionErrCh <- SystemTransitionHandlerWithContextAndRuntime(context.Background(), rectifyTestRuntime)
		}()
	})

	broadcast.DefaultBroadcastStateRuntime().UpdateBroadcast(structs.AuthBroadcast{})
	event := "bug-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	rectifyTestRuntime.EventRuntime.PublishSystemTransition(context.Background(), structs.SystemTransition{Type: "bugReportError", Event: event})

	testutil.WaitForCondition(t, func() bool {
		state := broadcast.DefaultBroadcastStateRuntime().GetState()
		return state.System.Transition.BugReportError == event
	}, "system transition bugReportError was not applied")
	state := broadcast.DefaultBroadcastStateRuntime().GetState()
	if got := state.System.Transition.BugReportError; got != event {
		t.Fatalf("unexpected system transition bug report error: got %q want %q", got, event)
	}
	assertNoAsyncRectifyError(t, "SystemTransitionHandlerWithContextAndRuntime", systemTransitionErrCh)
}

func TestRectifyUrbitAppliesStartramEvents(t *testing.T) {
	initializeRectifyTestEnv()
	rectifyUrbitOnce.Do(func() {
		go func() {
			rectifyUrbitErrCh <- RectifyUrbitWithContext(context.Background())
		}()
	})

	broadcast.DefaultBroadcastStateRuntime().UpdateBroadcast(structs.AuthBroadcast{})
	restartValue := "restart-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	startram.PublishEvent(structs.Event{Type: "restart", Data: restartValue})
	startram.PublishEvent(structs.Event{Type: "toggle", Data: "loading"})
	startram.PublishEvent(structs.Event{Type: "endpoint", Data: nil})

	testutil.WaitForCondition(t, func() bool {
		state := broadcast.DefaultBroadcastStateRuntime().GetState()
		return state.Profile.Startram.Transition.Restart == restartValue &&
			state.Profile.Startram.Transition.Toggle == "loading" &&
			state.Profile.Startram.Transition.Endpoint == ""
	}, "rectify startram transitions were not applied")
	state := broadcast.DefaultBroadcastStateRuntime().GetState()
	if got := state.Profile.Startram.Transition.Restart; got != restartValue {
		t.Fatalf("unexpected startram restart transition: got %q want %q", got, restartValue)
	}
	if got := state.Profile.Startram.Transition.Toggle; got != "loading" {
		t.Fatalf("unexpected startram toggle transition: got %q want %q", got, "loading")
	}
	if got := state.Profile.Startram.Transition.Endpoint; got != "" {
		t.Fatalf("unexpected startram endpoint transition: got %q want empty string", got)
	}
	assertNoAsyncRectifyError(t, "RectifyUrbitWithContext", rectifyUrbitErrCh)
}

func TestSetUrbitTransitionSupportsWorkflowRegistryAndTelemetry(t *testing.T) {
	for transitionType := range shipworkflow.UrbitTransitionReducerMap() {
		var event structs.UrbitTransition
		event.Type = string(transitionType)
		event.Event = "ok"
		event.Value = 1

		state := structs.UrbitTransitionBroadcast{}
		if !setUrbitTransition(&state, event) {
			t.Fatalf("expected urbit transition from workflow registry to be recognized: %s", transitionType)
		}
	}

	for _, transitionType := range []transition.UrbitTransitionType{
		transition.UrbitTransitionChop,
		transition.UrbitTransitionShipCompressed,
		transition.UrbitTransitionBucketCompressed,
		transition.UrbitTransitionPenpaiCompanion,
		transition.UrbitTransitionGallseg,
		transition.UrbitTransitionDeleteService,
		transition.UrbitTransitionLocalTlonBackupsEnabled,
		transition.UrbitTransitionRemoteTlonBackupsEnabled,
		transition.UrbitTransitionLocalTlonBackup,
		transition.UrbitTransitionLocalTlonBackupSchedule,
		transition.UrbitTransitionHandleRestoreTlonBackup,
		transition.UrbitTransitionServiceRegistrationStatus,
	} {
		event := structs.UrbitTransition{Type: string(transitionType), Event: "ok", Value: 1}
		state := structs.UrbitTransitionBroadcast{}
		if !setUrbitTransition(&state, event) {
			t.Fatalf("expected urbit transition to be recognized: %s", transitionType)
		}
	}
}
