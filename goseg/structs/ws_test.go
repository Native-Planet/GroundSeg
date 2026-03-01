package structs

import (
	"encoding/json"
	"testing"
)

func TestWsUrbitActionValueJSONTag(t *testing.T) {
	input := []byte(`{"type":"urbit","action":"loom","patp":"~zod","value":42}`)
	var action WsUrbitAction
	if err := json.Unmarshal(input, &action); err != nil {
		t.Fatalf("failed to unmarshal WsUrbitAction: %v", err)
	}
	if action.Value != 42 {
		t.Fatalf("expected value=42, got %d", action.Value)
	}
}
