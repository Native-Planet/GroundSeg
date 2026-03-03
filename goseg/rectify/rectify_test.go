package rectify

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"groundseg/broadcast"
	"groundseg/docker/events"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/testutil"
)

var (
	rectifyTestEnvOnce         sync.Once
	urbitTransitionHandlerOnce sync.Once
	newShipTransitionOnce      sync.Once
	systemTransitionOnce       sync.Once
	rectifyUrbitOnce           sync.Once
)

func initializeRectifyTestEnv() {
	rectifyTestEnvOnce.Do(func() {
		// BroadcastToClients can be triggered during transition updates.
		// The loop now publishes to leak asynchronously to avoid test deadlocks.
	})
}

func TestUrbitTransitionHandlerUpdatesBroadcastState(t *testing.T) {
	initializeRectifyTestEnv()
	urbitTransitionHandlerOnce.Do(func() {
		go func() {
			if err := UrbitTransitionHandlerWithContext(context.Background()); err != nil {
				t.Fatalf("UrbitTransitionHandlerWithContext failed: %v", err)
			}
		}()
	})

	current := structs.AuthBroadcast{
		Urbits: map[string]structs.Urbit{
			"zod": {},
		},
	}
	broadcast.UpdateBroadcast(current)

	eventValue := "pack-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	events.DefaultEventRuntime().PublishUrbitTransition(context.Background(), structs.UrbitTransition{
		Patp:  "zod",
		Type:  "pack",
		Event: eventValue,
	})

	testutil.WaitForCondition(t, func() bool {
		state := broadcast.GetState()
		return state.Urbits["zod"].Transition.Pack == eventValue
	}, "urbit pack transition was not applied")
}

func TestNewShipTransitionHandlerUpdatesBroadcastState(t *testing.T) {
	initializeRectifyTestEnv()
	newShipTransitionOnce.Do(func() {
		go func() {
			if err := NewShipTransitionHandlerWithContext(context.Background()); err != nil {
				t.Fatalf("NewShipTransitionHandlerWithContext failed: %v", err)
			}
		}()
	})

	broadcast.UpdateBroadcast(structs.AuthBroadcast{})
	bootStage := "booting-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	patp := "~zod-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	events.DefaultEventRuntime().PublishNewShipTransition(context.Background(), structs.NewShipTransition{Type: "bootStage", Event: bootStage})
	events.DefaultEventRuntime().PublishNewShipTransition(context.Background(), structs.NewShipTransition{Type: "patp", Event: patp})

	testutil.WaitForCondition(t, func() bool {
		state := broadcast.GetState()
		return state.NewShip.Transition.BootStage == bootStage && state.NewShip.Transition.Patp == patp
	}, "new ship transitions were not applied")
}

func TestSystemTransitionHandlerUpdatesBroadcastState(t *testing.T) {
	initializeRectifyTestEnv()
	systemTransitionOnce.Do(func() {
		go func() {
			if err := SystemTransitionHandlerWithContext(context.Background()); err != nil {
				t.Fatalf("SystemTransitionHandlerWithContext failed: %v", err)
			}
		}()
	})

	broadcast.UpdateBroadcast(structs.AuthBroadcast{})
	event := "bug-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	events.DefaultEventRuntime().PublishSystemTransition(context.Background(), structs.SystemTransition{Type: "bugReportError", Event: event})

	testutil.WaitForCondition(t, func() bool {
		state := broadcast.GetState()
		return state.System.Transition.BugReportError == event
	}, "system transition bugReportError was not applied")
}

func TestRectifyUrbitAppliesStartramEvents(t *testing.T) {
	initializeRectifyTestEnv()
	rectifyUrbitOnce.Do(func() {
		go func() {
			if err := RectifyUrbitWithContext(context.Background()); err != nil {
				t.Fatalf("RectifyUrbitWithContext failed: %v", err)
			}
		}()
	})

	broadcast.UpdateBroadcast(structs.AuthBroadcast{})
	restartValue := "restart-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	startram.PublishEvent(structs.Event{Type: "restart", Data: restartValue})
	startram.PublishEvent(structs.Event{Type: "toggle", Data: "loading"})
	startram.PublishEvent(structs.Event{Type: "endpoint", Data: nil})

	testutil.WaitForCondition(t, func() bool {
		state := broadcast.GetState()
		return state.Profile.Startram.Transition.Restart == restartValue &&
			state.Profile.Startram.Transition.Toggle == "loading" &&
			state.Profile.Startram.Transition.Endpoint == ""
	}, "rectify startram transitions were not applied")
}
