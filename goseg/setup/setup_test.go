package setup

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"groundseg/config"
	"groundseg/structs"

	"github.com/gorilla/websocket"
)

func resetSetupSeamsForTest(t *testing.T) {
	t.Helper()
	origUpdate := updateConfTypedForSetup
	origHasher := hasherForSetup
	origCycle := cycleWgKeyForSetup
	origRegister := startramRegisterForSetup
	origAddAuth := addToAuthMapForSetup
	t.Cleanup(func() {
		updateConfTypedForSetup = origUpdate
		hasherForSetup = origHasher
		cycleWgKeyForSetup = origCycle
		startramRegisterForSetup = origRegister
		addToAuthMapForSetup = origAddAuth
	})
}

func mustSetupMessage(t *testing.T, action, password, key, region string) []byte {
	t.Helper()
	payload := structs.WsSetupPayload{}
	payload.Type = "setup"
	payload.Payload.Action = action
	payload.Payload.Password = password
	payload.Payload.Key = key
	payload.Payload.Region = region
	msg, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal setup payload: %v", err)
	}
	return msg
}

func TestSetupRejectsInvalidJSON(t *testing.T) {
	if err := Setup([]byte("{"), &structs.MuConn{}, map[string]string{}); err == nil {
		t.Fatal("expected setup unmarshal error")
	}
}

func TestSetupBeginUpdatesProfileStage(t *testing.T) {
	resetSetupSeamsForTest(t)

	var patch config.ConfPatch
	updateConfTypedForSetup = func(opts ...config.ConfigUpdateOption) error {
		if len(opts) != 1 {
			t.Fatalf("expected 1 update option, got %d", len(opts))
		}
		opts[0](&patch)
		return nil
	}

	err := Setup(mustSetupMessage(t, "begin", "", "", ""), &structs.MuConn{}, map[string]string{})
	if err != nil {
		t.Fatalf("Setup(begin) returned error: %v", err)
	}
	if patch.Setup == nil || *patch.Setup != "profile" {
		t.Fatalf("expected setup stage profile, got %+v", patch)
	}
}

func TestSetupPasswordHashesAndPersistsPassword(t *testing.T) {
	resetSetupSeamsForTest(t)

	hasherInput := ""
	hasherForSetup = func(password string) string {
		hasherInput = password
		return "hashed-password"
	}

	var patch config.ConfPatch
	updateConfTypedForSetup = func(opts ...config.ConfigUpdateOption) error {
		for _, opt := range opts {
			opt(&patch)
		}
		return nil
	}

	err := Setup(mustSetupMessage(t, "password", "plain-secret", "", ""), &structs.MuConn{}, map[string]string{})
	if err != nil {
		t.Fatalf("Setup(password) returned error: %v", err)
	}
	if hasherInput != "plain-secret" {
		t.Fatalf("expected hasher to receive plain-secret, got %q", hasherInput)
	}
	if patch.Setup == nil || *patch.Setup != "startram" {
		t.Fatalf("expected setup stage startram, got %+v", patch)
	}
	if patch.PwHash == nil || *patch.PwHash != "hashed-password" {
		t.Fatalf("expected hashed password update, got %+v", patch)
	}
}

func TestSetupStartramRunsRegistrationAndAuthFlow(t *testing.T) {
	resetSetupSeamsForTest(t)

	steps := []string{}
	cycleWgKeyForSetup = func() error {
		steps = append(steps, "cycle")
		return nil
	}
	startramRegisterForSetup = func(key, region string) error {
		steps = append(steps, "register")
		if key != "reg-key" || region != "us-east" {
			t.Fatalf("unexpected startram registration input: key=%q region=%q", key, region)
		}
		return nil
	}
	updateConfTypedForSetup = func(opts ...config.ConfigUpdateOption) error {
		steps = append(steps, "update")
		patch := &config.ConfPatch{}
		for _, opt := range opts {
			opt(patch)
		}
		if patch.Setup == nil || *patch.Setup != "complete" {
			t.Fatalf("expected setup stage complete, got %+v", patch)
		}
		return nil
	}
	addToAuthMapForSetup = func(_ *websocket.Conn, token map[string]string, authed bool) error {
		steps = append(steps, "auth")
		if !authed {
			t.Fatal("expected auth map move to authorized")
		}
		if token["id"] != "tok-1" {
			t.Fatalf("unexpected token: %+v", token)
		}
		return nil
	}

	conn := &structs.MuConn{}
	token := map[string]string{"id": "tok-1"}
	err := Setup(mustSetupMessage(t, "startram", "", "reg-key", "us-east"), conn, token)
	if err != nil {
		t.Fatalf("Setup(startram) returned error: %v", err)
	}
	if strings.Join(steps, ",") != "cycle,register,update,auth" {
		t.Fatalf("unexpected step order: %v", steps)
	}
}

func TestSetupSkipCompletesAndAddsAuth(t *testing.T) {
	resetSetupSeamsForTest(t)

	updated := false
	updateConfTypedForSetup = func(opts ...config.ConfigUpdateOption) error {
		patch := &config.ConfPatch{}
		for _, opt := range opts {
			opt(patch)
		}
		updated = patch.Setup != nil && *patch.Setup == "complete"
		return nil
	}
	authed := false
	addToAuthMapForSetup = func(_ *websocket.Conn, _ map[string]string, authorized bool) error {
		authed = authorized
		return nil
	}

	err := Setup(mustSetupMessage(t, "skip", "", "", ""), &structs.MuConn{}, map[string]string{})
	if err != nil {
		t.Fatalf("Setup(skip) returned error: %v", err)
	}
	if !updated || !authed {
		t.Fatalf("expected skip to update setup completion and auth: updated=%v authed=%v", updated, authed)
	}
}

func TestSetupReturnsActionErrors(t *testing.T) {
	resetSetupSeamsForTest(t)

	startramRegisterForSetup = func(string, string) error { return errors.New("register failed") }
	if err := Setup(mustSetupMessage(t, "startram", "", "k", "r"), &structs.MuConn{}, map[string]string{}); err == nil {
		t.Fatal("expected startram setup error when registration fails")
	}

	if err := Setup(mustSetupMessage(t, "unknown", "", "", ""), &structs.MuConn{}, map[string]string{}); err == nil {
		t.Fatal("expected invalid action error")
	}
}
