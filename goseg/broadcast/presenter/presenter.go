package presenter

import (
	"encoding/json"
	"fmt"

	"groundseg/structs"
	"groundseg/transition"
)

type Envelope struct {
	Type      string                `json:"type"`
	AuthLevel string                `json:"auth_level"`
	Payload   structs.AuthBroadcast `json:"payload"`
}

func NewEnvelope(authLevel string, payload structs.AuthBroadcast) Envelope {
	return Envelope{
		Type:      string(transition.BroadcastMessageTypeStructure),
		AuthLevel: authLevel,
		Payload:   payload,
	}
}

func MarshalEnvelope(envelope Envelope) ([]byte, error) {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal broadcast envelope (%s): %w", envelope.AuthLevel, err)
	}
	return payload, nil
}

func MarshalAuthorized(payload structs.AuthBroadcast) ([]byte, error) {
	serialized, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal authorized broadcast payload: %w", err)
	}
	return serialized, nil
}

func MarshalScoped(payload structs.AuthBroadcast, patp string) ([]byte, bool, error) {
	shipInfo, exists := payload.Urbits[patp]
	if !exists {
		return nil, false, nil
	}
	scopedPayload := structs.AuthBroadcast{
		Type:      string(transition.BroadcastMessageTypeStructure),
		AuthLevel: patp,
		Urbits: map[string]structs.Urbit{
			patp: shipInfo,
		},
	}
	scopedPayload.Profile.Startram.Info.Registered = payload.Profile.Startram.Info.Registered
	scopedPayload.Profile.Startram.Info.Running = payload.Profile.Startram.Info.Running
	serialized, err := json.Marshal(scopedPayload)
	if err != nil {
		return nil, false, fmt.Errorf("marshal scoped broadcast payload (%s): %w", patp, err)
	}
	return serialized, true, nil
}
