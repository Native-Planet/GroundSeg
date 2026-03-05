package broadcast

import (
	"strings"
	"testing"

	"groundseg/structs"
	"groundseg/transition"
)

func TestGetStateJsonIncludesAuthLevelEnvelope(t *testing.T) {
	payload, err := GetStateJson(structs.AuthBroadcast{}, transition.BroadcastAuthLevelAuthorized)
	if err != nil {
		t.Fatalf("GetStateJson returned error: %v", err)
	}
	encoded := string(payload)
	if !strings.Contains(encoded, `"auth_level":"authorized"`) {
		t.Fatalf("expected auth-level envelope in serialized payload, got %s", encoded)
	}
}
