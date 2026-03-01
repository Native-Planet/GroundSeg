package auth

import (
	"testing"
)

func TestNewClientManagerInitializesMaps(t *testing.T) {
	manager := NewClientManager()
	if manager == nil {
		t.Fatal("expected manager instance")
	}
	if manager.AuthClients == nil {
		t.Fatal("expected auth clients map to be initialized")
	}
	if manager.UnauthClients == nil {
		t.Fatal("expected unauth clients map to be initialized")
	}
}

func TestTokenIdAuthed(t *testing.T) {
	manager := NewClientManager()
	manager.AuthClients["token-1"] = nil

	if !TokenIdAuthed(manager, "token-1") {
		t.Fatal("expected existing token to be authed")
	}
	if TokenIdAuthed(manager, "missing-token") {
		t.Fatal("expected missing token to be unauthed")
	}
}

func TestWsAuthCheckNilConn(t *testing.T) {
	if WsAuthCheck(nil) {
		t.Fatal("nil connection should not be treated as authenticated")
	}
}

func TestAddToAuthMapRejectsNilConn(t *testing.T) {
	token := map[string]string{
		"id":    "id-1",
		"token": "token-1",
	}
	if err := AddToAuthMap(nil, token, true); err == nil {
		t.Fatal("expected AddToAuthMap to reject nil connections")
	}
}
