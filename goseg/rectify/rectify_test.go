package rectify

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"groundseg/broadcast"
	"groundseg/docker"
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
		go UrbitTransitionHandler()
	})

	current := structs.AuthBroadcast{
		Urbits: map[string]structs.Urbit{
			"zod": {},
		},
	}
	broadcast.UpdateBroadcast(current)

	eventValue := "pack-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	docker.PublishUrbitTransition(structs.UrbitTransition{
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
		go NewShipTransitionHandler()
	})

	broadcast.UpdateBroadcast(structs.AuthBroadcast{})
	bootStage := "booting-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	patp := "~zod-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	docker.PublishNewShipTransition(structs.NewShipTransition{Type: "bootStage", Event: bootStage})
	docker.PublishNewShipTransition(structs.NewShipTransition{Type: "patp", Event: patp})

	testutil.WaitForCondition(t, func() bool {
		state := broadcast.GetState()
		return state.NewShip.Transition.BootStage == bootStage && state.NewShip.Transition.Patp == patp
	}, "new ship transitions were not applied")
}

func TestSystemTransitionHandlerUpdatesBroadcastState(t *testing.T) {
	initializeRectifyTestEnv()
	systemTransitionOnce.Do(func() {
		go SystemTransitionHandler()
	})

	broadcast.UpdateBroadcast(structs.AuthBroadcast{})
	event := "bug-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	docker.PublishSystemTransition(structs.SystemTransition{Type: "bugReportError", Event: event})

	testutil.WaitForCondition(t, func() bool {
		state := broadcast.GetState()
		return state.System.Transition.BugReportError == event
	}, "system transition bugReportError was not applied")
}

func TestRectifyUrbitAppliesStartramEvents(t *testing.T) {
	initializeRectifyTestEnv()
	rectifyUrbitOnce.Do(func() {
		go RectifyUrbit()
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
