package router

import (
	"encoding/json"
	"testing"

	"groundseg/handler/api"
	"groundseg/handler/ship"
	"groundseg/handler/system"
	"groundseg/leakchannel"
	"groundseg/structs"
)

func resetLeakSeams() {
	urbitHandlerForLeak = ship.UrbitHandler
	penpaiHandlerForLeak = system.PenpaiHandler
	newShipHandlerForLeak = api.NewShipHandler
	systemHandlerForLeak = system.SystemHandler
	startramHandlerForLeak = system.StartramHandler
	supportHandlerForLeak = system.SupportHandler
	pwHandlerForLeak = api.PwHandler
}

func TestGallsegUnauthHandlerRoutesOnlyMatchingUrbitPayload(t *testing.T) {
	t.Cleanup(resetLeakSeams)
	calls := 0
	urbitHandlerForLeak = func([]byte) error {
		calls++
		return nil
	}

	gallsegUnauthHandler(leakchannel.ActionChannel{Patp: "~zod", Content: []byte("{invalid")})
	gallsegUnauthHandler(leakchannel.ActionChannel{
		Patp: "~zod",
		Content: mustJSON(t, structs.WsUrbitPayload{
			Payload: structs.WsUrbitAction{Type: "system", Patp: "~zod"},
		}),
	})
	gallsegUnauthHandler(leakchannel.ActionChannel{
		Patp: "~zod",
		Content: mustJSON(t, structs.WsUrbitPayload{
			Payload: structs.WsUrbitAction{Type: "urbit", Patp: "~bus"},
		}),
	})
	if calls != 0 {
		t.Fatalf("expected no calls for invalid/mismatched payloads, got %d", calls)
	}

	gallsegUnauthHandler(leakchannel.ActionChannel{
		Patp: "~zod",
		Content: mustJSON(t, structs.WsUrbitPayload{
			Payload: structs.WsUrbitAction{Type: "urbit", Patp: "~zod"},
		}),
	})
	if calls != 1 {
		t.Fatalf("expected one urbit handler call, got %d", calls)
	}
}

func TestGallsegAuthedHandlerDispatchesByActionType(t *testing.T) {
	t.Cleanup(resetLeakSeams)
	counts := map[string]int{}
	urbitHandlerForLeak = func([]byte) error { counts["urbit"]++; return nil }
	penpaiHandlerForLeak = func([]byte) error { counts["penpai"]++; return nil }
	newShipHandlerForLeak = func([]byte) error { counts["new_ship"]++; return nil }
	systemHandlerForLeak = func([]byte) error { counts["system"]++; return nil }
	startramHandlerForLeak = func([]byte) error { counts["startram"]++; return nil }
	supportHandlerForLeak = func([]byte) error { counts["support"]++; return nil }
	pwHandlerForLeak = func([]byte, bool) error {
		counts["password"]++
		return nil
	}

	for _, actionType := range []string{"urbit", "penpai", "new_ship", "system", "startram", "support", "password"} {
		gallsegAuthedHandler(leakchannel.ActionChannel{Type: actionType, Content: []byte("{}")})
	}
	gallsegAuthedHandler(leakchannel.ActionChannel{Type: "unknown", Content: []byte("{}")})

	for _, actionType := range []string{"urbit", "penpai", "new_ship", "system", "startram", "support", "password"} {
		if counts[actionType] != 1 {
			t.Fatalf("expected %s handler call once, got %d", actionType, counts[actionType])
		}
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	return data
}
