package session

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestMuConnGuards(t *testing.T) {
	inactiveReader := &MuConn{}
	_, _, err := inactiveReader.Read(nil)
	if err == nil {
		t.Fatal("expected read on inactive connection to return error")
	}

	inactiveWriter := &MuConn{}
	if err := inactiveWriter.Write([]byte("payload")); err != nil {
		t.Fatalf("expected write on inactive connection to be a no-op, got %v", err)
	}
}

func TestAppendAndRemoveConnHelpers(t *testing.T) {
	connA := &websocket.Conn{}
	connB := &websocket.Conn{}

	existing := appendMuConnIfMissing(nil, &MuConn{Conn: connA})
	existing = appendMuConnIfMissing(existing, &MuConn{Conn: connA})
	if len(existing) != 1 {
		t.Fatalf("expected duplicate connection to be ignored, got %d", len(existing))
	}

	existing = appendMuConnIfMissing(existing, &MuConn{Conn: connB})
	if len(existing) != 2 {
		t.Fatalf("expected second unique connection to be added, got %d", len(existing))
	}

	filtered, removed := removeConnFromBucket(existing, connA)
	if !removed || len(filtered) != 1 {
		t.Fatalf("expected connA to be removed, removed=%v len=%d", removed, len(filtered))
	}
}

func TestClientManagerConnectionLifecycle(t *testing.T) {
	cm := NewClientManager()
	if cm == nil {
		t.Fatal("expected NewClientManager to initialize")
	}

	conn := &websocket.Conn{}
	muConn := cm.GetMuConn(conn, "token")
	if muConn == nil || muConn.Conn != conn {
		t.Fatalf("expected active muConn for new websocket connection")
	}
	if cm.GetMuConn(conn, "token") != muConn {
		t.Fatalf("expected repeated GetMuConn to return same object")
	}

	cm.AddUnauthClient("token", muConn)
	if got := cm.AuthClientCount("token"); got != 0 {
		t.Fatalf("expected auth bucket to start empty, got %d", got)
	}
	if got := cm.UnauthClientCount("token"); got != 1 {
		t.Fatalf("expected unauth bucket to contain one client, got %d", got)
	}

	cm.AddAuthClient("token", muConn)
	if got := cm.AuthClientCount("token"); got != 1 {
		t.Fatalf("expected auth transition to add client, got %d", got)
	}
	if got := cm.UnauthClientCount("token"); got != 0 {
		t.Fatalf("expected auth transition to remove client from unauth bucket, got %d", got)
	}

	if !cm.HasAuthConnection("token", conn) {
		t.Fatalf("expected connection to be marked authenticated")
	}
	if !cm.HasAnyAuthConnection(conn) {
		t.Fatalf("expected connection to be discoverable as an auth connection")
	}
}

func TestClientManagerTransitionAndCleanup(t *testing.T) {
	cm := NewClientManager()

	stale := &MuConn{Conn: &websocket.Conn{}, Active: true, LastActive: time.Now().Add(-20 * time.Minute)}
	fresh := &MuConn{Conn: &websocket.Conn{}, Active: true, LastActive: time.Now().Add(-10 * time.Second)}
	queued := &MuConn{Conn: nil, Active: true, LastActive: time.Now().Add(-20 * time.Minute)}

	cm.transitionClientState("stale", stale, true)
	cm.transitionClientState("fresh", fresh, true)
	cm.transitionClientState("queued", queued, false)

	if got := cm.AuthClientCount("stale"); got != 1 {
		t.Fatalf("expected stale auth client to be tracked, got %d", got)
	}
	if got := cm.UnauthClientCount("queued"); got != 0 {
		t.Fatalf("expected queued unauth token to have no live socket bucket entries, got %d", got)
	}
	if _, queuedTracked := cm.store.unauthTokens["queued"]; !queuedTracked {
		t.Fatal("expected queued unauth token to remain in session bookkeeping")
	}

	cm.CleanupStaleSessions(time.Minute)
	if got := cm.AuthClientCount("stale"); got != 0 {
		t.Fatalf("expected stale auth client to be cleaned up, got %d", got)
	}
	if got := cm.AuthClientCount("fresh"); got != 1 {
		t.Fatalf("expected fresh auth client to remain, got %d", got)
	}
	if got := cm.UnauthClientCount("queued"); got != 0 {
		t.Fatalf("expected queued unauth bucket to remain nil/empty with no live conn, got %d", got)
	}
	if _, queuedTracked := cm.store.unauthTokens["queued"]; !queuedTracked {
		t.Fatal("expected queued token bookkeeping to persist without live connection")
	}
}

func TestClientManagerDeactivateAndBroadcastNoop(t *testing.T) {
	cm := NewClientManager()
	conn := &websocket.Conn{}
	cm.AddUnauthClient("token", &MuConn{Conn: conn, Active: true})

	if !cm.DeactivateConnection(conn) {
		t.Fatalf("expected deactivate to indicate tracked connection")
	}
	if !cm.DeactivateConnection(conn) {
		t.Fatalf("expected deactivate to remain idempotent on already inactive connection")
	}

	deactivated := cm.FindMuConn(conn)
	if deactivated == nil || deactivated.Active {
		t.Fatalf("expected deactivated client to be inactive")
	}

	// broadcast should ignore inactive and avoid calling websocket internals in tests.
	cm.BroadcastAuth([]byte("auth"))
	cm.BroadcastUnauth([]byte("unauth"))
}

func TestHandleReadPathWithoutConnection(t *testing.T) {
	cm := NewClientManager()
	if _, _, err := (&MuConn{Active: false}).Read(nil); err == nil {
		t.Fatalf("expected read failure for inactive connection")
	}

	cm.handleWriteFailure(nil, "scope", "token", nil)
}
