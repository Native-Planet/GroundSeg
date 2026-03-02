package ship

import (
	"context"
	"encoding/json"
	"errors"
	"groundseg/shipworkflow"
	"strings"
	"testing"
	"time"

	"groundseg/config"
	"groundseg/structs"
)

func buildUrbitPayload(t *testing.T, action, patp string) []byte {
	t.Helper()
	payload := structs.WsUrbitPayload{
		Payload: structs.WsUrbitAction{
			Action: action,
			Patp:   patp,
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal urbit payload: %v", err)
	}
	return data
}

func TestUrbitHandlerDispatchesRegisteredAction(t *testing.T) {
	originalCommands := urbitCommands
	t.Cleanup(func() {
		urbitCommands = originalCommands
	})

	var gotPatp string
	var gotAction string
	urbitCommands = map[string]urbitCommand{
		"test-action": func(patp string, payload structs.WsUrbitPayload) error {
			gotPatp = patp
			gotAction = payload.Payload.Action
			return nil
		},
	}

	if err := UrbitHandler(buildUrbitPayload(t, "test-action", "~zod")); err != nil {
		t.Fatalf("UrbitHandler returned error: %v", err)
	}
	if gotPatp != "~zod" {
		t.Fatalf("unexpected patp: got %q", gotPatp)
	}
	if gotAction != "test-action" {
		t.Fatalf("unexpected action: got %q", gotAction)
	}
}

func TestUrbitHandlerPropagatesCommandError(t *testing.T) {
	originalCommands := urbitCommands
	t.Cleanup(func() {
		urbitCommands = originalCommands
	})

	urbitCommands = map[string]urbitCommand{
		"test-action": func(string, structs.WsUrbitPayload) error {
			return errors.New("boom")
		},
	}

	err := UrbitHandler(buildUrbitPayload(t, "test-action", "~nec"))
	if err == nil {
		t.Fatal("expected UrbitHandler to return command error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUrbitHandlerRejectsUnknownAction(t *testing.T) {
	originalCommands := urbitCommands
	t.Cleanup(func() {
		urbitCommands = originalCommands
	})
	urbitCommands = map[string]urbitCommand{}

	err := UrbitHandler(buildUrbitPayload(t, "missing-action", "~bus"))
	if err == nil {
		t.Fatal("expected unknown action to fail")
	}
	if !strings.Contains(err.Error(), "unrecognized urbit action") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUrbitHandlerRejectsMalformedPayload(t *testing.T) {
	if err := UrbitHandler([]byte("{not-json")); err == nil {
		t.Fatal("expected malformed payload error")
	}
}

func TestUrbitHandlerRecoversFromPanicInAction(t *testing.T) {
	originalCommands := urbitCommands
	t.Cleanup(func() {
		urbitCommands = originalCommands
	})

	urbitCommands = map[string]urbitCommand{
		"panic-action": func(string, structs.WsUrbitPayload) error {
			panic("boom")
		},
	}
	err := UrbitHandler(buildUrbitPayload(t, "panic-action", "~zod"))
	if err == nil {
		t.Fatal("expected panic to be recovered and returned as error")
	}
	if !strings.Contains(err.Error(), "panic handling urbit action panic-action") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func resetUrbitTestSeams() func() {
	originalStatus := urbitGetShipStatus
	originalCleanDelete := urbitCleanDeleteFn
	originalPoller := waitCompletePoller
	originalWaitCompleteFn := waitCompleteFn
	originalAliasFn := areSubdomainsAliasesFn
	return func() {
		urbitGetShipStatus = originalStatus
		waitCompletePoller = originalPoller
		waitCompleteFn = originalWaitCompleteFn
		urbitCleanDeleteFn = originalCleanDelete
		areSubdomainsAliasesFn = originalAliasFn
	}
}

func TestCutSliceRemovesMatch(t *testing.T) {
	got := cutSlice([]string{"alpha", "beta", "gamma"}, "beta")
	want := []string{"alpha", "gamma"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected slice after cut: got %v want %v", got, want)
	}

	unchanged := cutSlice([]string{"alpha", "beta"}, "delta")
	if len(unchanged) != 2 || unchanged[0] != "alpha" || unchanged[1] != "beta" {
		t.Fatalf("slice should be unchanged when key is missing: %v", unchanged)
	}
}

func TestAreSubdomainsAliasesUsesAltCnameBypass(t *testing.T) {
	restoreSeams := resetUrbitTestSeams()
	t.Cleanup(restoreSeams)
	originalRetrieve := config.GetStartramConfig()
	t.Cleanup(func() {
		config.SetStartramConfig(originalRetrieve)
	})

	config.SetStartramConfig(structs.StartramRetrieve{Cname: "alt.example"})
	areSubdomainsAliasesFn = func(domain1, domain2 string) (bool, error) {
		return shipworkflow.AreSubdomainsAliasesWithLookup(func(string) (string, error) {
			t.Fatal("lookupCNAME should not be called when cname bypass matches")
			return "", nil
		}, domain1, domain2)
	}

	isAlias, err := AreSubdomainsAliases("ship.alt.example", "ignored.example")
	if err != nil {
		t.Fatalf("AreSubdomainsAliases returned error: %v", err)
	}
	if !isAlias {
		t.Fatal("expected cname bypass to mark domains as aliases")
	}
}

func TestAreSubdomainsAliasesComparesResolvedCnames(t *testing.T) {
	restoreSeams := resetUrbitTestSeams()
	t.Cleanup(restoreSeams)
	originalRetrieve := config.GetStartramConfig()
	t.Cleanup(func() {
		config.SetStartramConfig(originalRetrieve)
	})
	config.SetStartramConfig(structs.StartramRetrieve{})

	lookupResults := map[string]string{
		"a.example": "shared.target.",
		"b.example": "shared.target.",
		"c.example": "other.target.",
	}
	areSubdomainsAliasesFn = func(domain1, domain2 string) (bool, error) {
		return shipworkflow.AreSubdomainsAliasesWithLookup(func(domain string) (string, error) {
			resolved, ok := lookupResults[domain]
			if !ok {
				return "", errors.New("missing lookup")
			}
			return resolved, nil
		}, domain1, domain2)
	}

	matched, err := AreSubdomainsAliases("a.example", "b.example")
	if err != nil {
		t.Fatalf("unexpected lookup error: %v", err)
	}
	if !matched {
		t.Fatal("expected equal CNAMEs to match")
	}

	matched, err = AreSubdomainsAliases("a.example", "c.example")
	if err != nil {
		t.Fatalf("unexpected lookup error: %v", err)
	}
	if matched {
		t.Fatal("expected different CNAMEs to not match")
	}
}

func TestAreSubdomainsAliasesRejectsInvalidSubdomain(t *testing.T) {
	isAlias, err := AreSubdomainsAliases("invalid-domain", "other.example")
	if err == nil {
		t.Fatal("expected invalid domain error")
	}
	if isAlias {
		t.Fatal("invalid domain should never be considered alias")
	}
}

func TestWaitCompleteReturnsWhenShipStops(t *testing.T) {
	restoreSeams := resetUrbitTestSeams()
	t.Cleanup(restoreSeams)

	waitCompletePoller = func(_ context.Context, _ time.Duration, condition func() (bool, error)) error {
		for i := 0; i < 5; i++ {
			done, err := condition()
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
		return errors.New("condition never completed")
	}
	urbitGetShipStatus = func([]string) (map[string]string, error) {
		return map[string]string{"zod": "Exited (0)"}, nil
	}

	if err := WaitComplete("zod"); err != nil {
		t.Fatalf("WaitComplete returned error: %v", err)
	}
}

func TestWaitCompleteFailsAfterStatusRetriesExhausted(t *testing.T) {
	restoreSeams := resetUrbitTestSeams()
	t.Cleanup(restoreSeams)

	waitCompletePoller = func(_ context.Context, _ time.Duration, condition func() (bool, error)) error {
		for i := 0; i < 10; i++ {
			done, err := condition()
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
		return errors.New("condition never completed")
	}
	urbitGetShipStatus = func([]string) (map[string]string, error) {
		return nil, errors.New("docker unavailable")
	}

	err := WaitComplete("zod")
	if err == nil {
		t.Fatal("expected WaitComplete to fail when status fetch keeps failing")
	}
	if !strings.Contains(err.Error(), "retrieve ship status for zod") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUrbitCleanDeleteStopsRunningShipAndDeletesContainer(t *testing.T) {
	restoreSeams := resetUrbitTestSeams()
	t.Cleanup(restoreSeams)

	calls := 0
	urbitCleanDeleteFn = func(patp string) error {
		calls++
		if patp != "zod" {
			t.Fatalf("unexpected patp: %q", patp)
		}
		return nil
	}

	if err := urbitCleanDelete("zod"); err != nil {
		t.Fatalf("urbitCleanDelete returned error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected shipworkflow clean delete to be called once, got %d", calls)
	}
}

func TestUrbitCleanDeletePropagatesDeleteContainerError(t *testing.T) {
	restoreSeams := resetUrbitTestSeams()
	t.Cleanup(restoreSeams)

	urbitCleanDeleteFn = func(string) error {
		return errors.New("delete failed")
	}

	err := urbitCleanDelete("zod")
	if err == nil {
		t.Fatal("expected delete error")
	}
	if !strings.Contains(err.Error(), "delete failed") {
		t.Fatalf("expected wrapped delete error, got %v", err)
	}
}
