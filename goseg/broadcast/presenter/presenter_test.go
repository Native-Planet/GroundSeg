package presenter

import (
	"encoding/json"
	"testing"

	"groundseg/structs"
)

func sampleBroadcastState() structs.AuthBroadcast {
	state := structs.AuthBroadcast{}
	state.Urbits = map[string]structs.Urbit{
		"zod": {},
	}
	state.Profile.Startram.Info.Registered = true
	state.Profile.Startram.Info.Running = true
	return state
}

func TestMarshalEnvelopeIncludesEnvelopeMetadata(t *testing.T) {
	state := sampleBroadcastState()

	payload, err := MarshalEnvelope(NewEnvelope("authorized", state))
	if err != nil {
		t.Fatalf("MarshalEnvelope returned error: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal envelope payload: %v", err)
	}
	if _, ok := decoded["type"]; !ok {
		t.Fatal("expected envelope type field")
	}
	if _, ok := decoded["auth_level"]; !ok {
		t.Fatal("expected envelope auth_level field")
	}
	if _, ok := decoded["payload"]; !ok {
		t.Fatal("expected envelope payload field")
	}
}

func TestMarshalAuthorizedUsesLegacyBroadcastShape(t *testing.T) {
	state := sampleBroadcastState()

	payload, err := MarshalAuthorized(state)
	if err != nil {
		t.Fatalf("MarshalAuthorized returned error: %v", err)
	}

	var decoded structs.AuthBroadcast
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal authorized payload: %v", err)
	}
	if len(decoded.Urbits) != 1 {
		t.Fatalf("expected authorized payload to preserve urbit map, got %d entries", len(decoded.Urbits))
	}
}

func TestMarshalScopedProducesScopedPayload(t *testing.T) {
	state := sampleBroadcastState()

	payload, exists, err := MarshalScoped(state, "zod")
	if err != nil {
		t.Fatalf("MarshalScoped returned error: %v", err)
	}
	if !exists {
		t.Fatal("expected scoped payload for known patp")
	}

	var decoded structs.AuthBroadcast
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal scoped payload: %v", err)
	}
	if decoded.Type != "structure" {
		t.Fatalf("expected scoped type structure, got %q", decoded.Type)
	}
	if decoded.AuthLevel != "zod" {
		t.Fatalf("expected scoped auth level zod, got %q", decoded.AuthLevel)
	}
	if len(decoded.Urbits) != 1 {
		t.Fatalf("expected exactly one urbit in scoped payload, got %d", len(decoded.Urbits))
	}
	if !decoded.Profile.Startram.Info.Registered || !decoded.Profile.Startram.Info.Running {
		t.Fatal("expected scoped payload to preserve startram registration/running flags")
	}
}

func TestMarshalScopedReturnsNotFoundForUnknownPatp(t *testing.T) {
	state := sampleBroadcastState()

	payload, exists, err := MarshalScoped(state, "nec")
	if err != nil {
		t.Fatalf("MarshalScoped returned error for unknown patp: %v", err)
	}
	if exists {
		t.Fatal("expected exists=false for unknown patp")
	}
	if payload != nil {
		t.Fatalf("expected nil payload for unknown patp, got %d bytes", len(payload))
	}
}
