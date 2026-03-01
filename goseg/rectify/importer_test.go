package rectify

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"groundseg/broadcast"
	"groundseg/docker"
	"groundseg/structs"
	"groundseg/testutil"
)

var importShipTransitionHandlerOnce sync.Once

func startImportShipTransitionHandler() {
	importShipTransitionHandlerOnce.Do(func() {
		go ImportShipTransitionHandler()
	})
}

func TestImportShipTransitionHandlerAppliesUploadTransitions(t *testing.T) {
	initializeRectifyTestEnv()
	startImportShipTransitionHandler()

	broadcast.UpdateBroadcast(structs.AuthBroadcast{})
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	status := "creating-" + suffix
	patp := "~zod-" + suffix
	errMsg := "error-" + suffix
	extracted := 77

	docker.PublishImportShipTransition(structs.UploadTransition{Type: "status", Event: status})
	docker.PublishImportShipTransition(structs.UploadTransition{Type: "patp", Event: patp})
	docker.PublishImportShipTransition(structs.UploadTransition{Type: "error", Event: errMsg})
	docker.PublishImportShipTransition(structs.UploadTransition{Type: "extracted", Value: extracted})

	testutil.WaitForCondition(t, func() bool {
		state := broadcast.GetState()
		return state.Upload.Status == status &&
			state.Upload.Patp == patp &&
			state.Upload.Error == errMsg &&
			state.Upload.Extracted == int64(extracted)
	}, "import ship transitions were not applied to broadcast upload state")
}

func TestImportShipTransitionHandlerIgnoresUnknownTransitionTypes(t *testing.T) {
	initializeRectifyTestEnv()
	startImportShipTransitionHandler()

	initial := structs.AuthBroadcast{}
	initial.Upload.Status = "steady"
	initial.Upload.Patp = "~bus"
	initial.Upload.Error = "none"
	initial.Upload.Extracted = 42
	broadcast.UpdateBroadcast(initial)

	docker.PublishImportShipTransition(structs.UploadTransition{Type: "unknown", Event: "ignored"})
	time.Sleep(100 * time.Millisecond)

	state := broadcast.GetState()
	if state.Upload != initial.Upload {
		t.Fatalf("unexpected upload state change for unknown transition: got %+v want %+v", state.Upload, initial.Upload)
	}
}
