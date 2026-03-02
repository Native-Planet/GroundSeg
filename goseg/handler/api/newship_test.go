package api

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"groundseg/docker/events"
	"groundseg/structs"
)

func drainNewShipTransitions() {
	for {
		select {
		case <-events.NewShipTransitions():
		default:
			return
		}
	}
}

func readNewShipTransitions(t *testing.T, count int) []structs.NewShipTransition {
	t.Helper()
	capturedEvents := make([]structs.NewShipTransition, 0, count)
	for i := 0; i < count; i++ {
		select {
		case evt := <-events.NewShipTransitions():
			capturedEvents = append(capturedEvents, evt)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for new-ship transition %d", i+1)
		}
	}
	return capturedEvents
}

func buildNewShipMessage(t *testing.T, action, patp string) []byte {
	t.Helper()
	payload := structs.WsNewShipPayload{
		Payload: structs.WsNewShipAction{
			Action:        action,
			Patp:          patp,
			SelectedDrive: "system-drive",
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal new-ship payload: %v", err)
	}
	return data
}

func TestNewShipHandlerResetPublishesTransitions(t *testing.T) {
	drainNewShipTransitions()

	if err := NewShipHandler(buildNewShipMessage(t, "reset", "")); err != nil {
		t.Fatalf("NewShipHandler(reset) returned error: %v", err)
	}
	events := readNewShipTransitions(t, 3)
	expected := []structs.NewShipTransition{
		{Type: "bootStage", Event: ""},
		{Type: "patp", Event: ""},
		{Type: "error", Event: ""},
	}
	for i, want := range expected {
		if events[i] != want {
			t.Fatalf("unexpected event[%d]: want %+v got %+v", i, want, events[i])
		}
	}
}

func TestNewShipHandlerCancelRoutesToDeleteAndResets(t *testing.T) {
	originalCancel := handleNewShipCancel
	originalReset := handleNewShipReset
	t.Cleanup(func() {
		handleNewShipCancel = originalCancel
		handleNewShipReset = originalReset
	})

	var deletedPatp string
	handleNewShipCancel = func(patp string) error {
		deletedPatp = patp
		return nil
	}
	handleNewShipReset = func() error {
		return originalReset()
	}

	drainNewShipTransitions()
	if err := NewShipHandler(buildNewShipMessage(t, "cancel", "~zod")); err != nil {
		t.Fatalf("NewShipHandler(cancel) returned error: %v", err)
	}
	if deletedPatp != "~zod" {
		t.Fatalf("expected delete-ship to receive ~zod, got %q", deletedPatp)
	}
	events := readNewShipTransitions(t, 3)
	if events[0].Type != "bootStage" || events[1].Type != "patp" || events[2].Type != "error" {
		t.Fatalf("unexpected reset transition sequence: %+v", events)
	}
}

func TestNewShipHandlerBootRejectsInvalidPatp(t *testing.T) {
	err := NewShipHandler(buildNewShipMessage(t, "boot", "~notpatp"))
	if err == nil {
		t.Fatal("expected boot action to reject invalid patp")
	}
	if !strings.Contains(err.Error(), "Invalid @p provided") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewShipHandlerRejectsInvalidPayload(t *testing.T) {
	if err := NewShipHandler([]byte("{bad-json")); err == nil {
		t.Fatal("expected payload unmarshal error")
	}
}

func TestNewShipHandlerRejectsUnknownAction(t *testing.T) {
	err := NewShipHandler(buildNewShipMessage(t, "unknown-action", "~zod"))
	if err == nil {
		t.Fatal("expected unknown action error")
	}
	if !strings.Contains(err.Error(), "Unknown NewShip action") {
		t.Fatalf("unexpected error: %v", err)
	}
}
