package auth

import (
	"groundseg/structs"
	"testing"
)

func TestNewClientManagerInitializesMaps(t *testing.T) {
	manager := NewClientManager()
	if manager == nil {
		t.Fatal("expected manager instance")
	}
	if count := manager.AuthClientCount("test-token"); count != 0 {
		t.Fatalf("expected auth clients bucket for unknown token to be empty, got %d", count)
	}
	if count := manager.UnauthClientCount("test-token"); count != 0 {
		t.Fatalf("expected unauth clients bucket for unknown token to be empty, got %d", count)
	}
}

func TestTokenIdAuthed(t *testing.T) {
	manager := NewClientManager()
	manager.AddAuthClient("token-1", &structs.MuConn{})

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
