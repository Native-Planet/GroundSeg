package structs

import (
	"testing"

	"github.com/gorilla/websocket"
)

func TestGetMuConnReusesExistingSessionAcrossTokens(t *testing.T) {
	cm := &ClientManager{
		AuthClients:   make(map[string][]*MuConn),
		UnauthClients: make(map[string][]*MuConn),
	}

	conn := &websocket.Conn{}
	first := cm.GetMuConn(conn, "token-a")
	if first == nil {
		t.Fatal("expected first GetMuConn call to return a MuConn")
	}
	if len(cm.UnauthClients["token-a"]) != 1 {
		t.Fatalf("expected one unauth client for token-a, got %d", len(cm.UnauthClients["token-a"]))
	}
	if first != cm.UnauthClients["token-a"][0] {
		t.Fatal("expected stored unauth MuConn to match returned MuConn")
	}

	second := cm.GetMuConn(conn, "token-b")
	if second != first {
		t.Fatal("expected GetMuConn to reuse the managed connection object across token changes")
	}
	if len(cm.UnauthClients["token-a"]) != 1 {
		t.Fatalf("expected existing managed session to remain singular, got %d", len(cm.UnauthClients["token-a"]))
	}
}

func TestGetMuConnReturnsNilForNilConn(t *testing.T) {
	cm := &ClientManager{
		AuthClients:   make(map[string][]*MuConn),
		UnauthClients: make(map[string][]*MuConn),
	}

	if got := cm.GetMuConn(nil, "token"); got != nil {
		t.Fatalf("expected nil conn to return nil, got %#v", got)
	}
}
