package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"groundseg/config"
	"groundseg/session"
	"groundseg/structs"

	"github.com/gorilla/websocket"
)

func TestMain(m *testing.M) {
	tmpRoot, err := os.MkdirTemp("", "groundseg-auth-*")
	if err != nil {
		panic(err)
	}
	basePath := filepath.Join(tmpRoot, "groundseg")
	if err := os.MkdirAll(filepath.Join(basePath, "settings"), 0o755); err != nil {
		panic(err)
	}
	config.SetBasePath(basePath)
	_ = os.Setenv("GS_BASE_PATH", basePath)
	os.Exit(m.Run())
}

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

func TestGetMuConnAndReadMuConnHandleNilInputs(t *testing.T) {
	if GetMuConn(nil, "token-id") != nil {
		t.Fatal("expected nil manager when websocket connection is nil")
	}
	_, _, err := ReadMuConn(nil)
	if err == nil || !strings.Contains(err.Error(), "invalid websocket session") {
		t.Fatalf("expected invalid websocket session error, got %v", err)
	}
}

func TestWsAuthCheckRejectsNilConn(t *testing.T) {
	if WsAuthCheck(nil) {
		t.Fatal("nil connection should not be treated as authenticated")
	}
}

func TestWsAuthCheckRequiresTrackedSession(t *testing.T) {
	conn := &websocket.Conn{}
	if WsAuthCheck(conn) {
		t.Fatal("untracked websocket should not be authenticated")
	}
	if err := WsNilSession(conn); err == nil {
		t.Fatal("expected WsNilSession to fail for untracked connection")
	}
	if WsNilSession(&websocket.Conn{}) == nil {
		// A fresh connection with no authenticated session should always fail.
		t.Fatal("expected WsNilSession to fail when no session exists")
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

func TestTokenIdAuthedValidatesWhitespace(t *testing.T) {
	if IsTokenIdAuthed("  ") {
		t.Fatal("whitespace token should never be authenticated")
	}
}

func TestSetClientManagerRecoversNilAndRestoreExisting(t *testing.T) {
	original := GetClientManager()
	t.Cleanup(func() {
		SetClientManager(original)
	})

	SetClientManager(nil)
	if GetClientManager() == nil {
		t.Fatal("expected non-nil client manager after nil assignment")
	}

	custom := session.NewClientManager()
	SetClientManager(custom)
	if GetClientManager() != custom {
		t.Fatal("expected custom client manager restored")
	}
}

func TestAddToAuthMapAndRemoval(t *testing.T) {
	if err := config.Initialize(); err != nil {
		t.Fatalf("failed to initialize config for test: %v", err)
	}
	SetClientManager(NewClientManager())
	token := map[string]string{
		"id":    "test-id",
		"token": "test-token",
	}
	conn := &websocket.Conn{}
	if err := AddToAuthMap(conn, token, true); err != nil {
		t.Fatalf("expected token to be added: %v", err)
	}

	if !GetClientManager().HasAuthConnection("test-id", conn) {
		t.Fatal("expected websocket to be marked authed")
	}

	manager := GetClientManager()
	RemoveFromAuthMap("", true)
	if !manager.HasAuthToken("test-id") {
		t.Fatal("expected token to remain in auth map")
	}

	RemoveFromAuthMap("test-id", true)
	if manager.HasAuthToken("test-id") {
		t.Fatal("expected token to be removed from auth map")
	}

	if err := AddToAuthMap(conn, map[string]string{"id": "test-id", "token": "ignored"}, true); err != nil {
		t.Fatal("expected overwrite path to be accepted")
	}

	if err := AddToAuthMap(conn, map[string]string{"token": "missing-id"}, true); err == nil {
		t.Fatal(fmt.Sprintf("expected missing id to fail: %v", err))
	}
}
