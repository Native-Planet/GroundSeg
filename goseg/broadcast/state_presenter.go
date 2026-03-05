package broadcast

import (
	"fmt"

	"groundseg/broadcast/presenter"
	"groundseg/structs"
	"groundseg/transition"
)

// GetStateJson returns a serialized broadcast envelope with an explicit auth level.
func GetStateJson(bState structs.AuthBroadcast, authLevel transition.BroadcastAuthLevel) ([]byte, error) {
	broadcastJSON, err := presenter.MarshalEnvelope(presenter.NewEnvelope(string(authLevel), bState))
	if err != nil {
		return nil, fmt.Errorf("marshalling broadcast state payload: %w", err)
	}
	return broadcastJSON, nil
}
