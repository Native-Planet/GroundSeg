package authsession

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"groundseg/session"
)

type countingBoundary struct {
	mu           sync.Mutex
	addCalled    int
	removeCalled int
	addErr       error
}

func (cb *countingBoundary) AddToAuthMap(_ *websocket.Conn, _ map[string]string, _ bool) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.addCalled++
	return cb.addErr
}

func (cb *countingBoundary) RemoveFromAuthMap(_ string, _ bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.removeCalled++
}

func (cb *countingBoundary) snapshot() (int, int) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.addCalled, cb.removeCalled
}

func TestSetSessionBoundaryIsSafeForConcurrentMutationAndCalls(t *testing.T) {
	boundaryA := &countingBoundary{}
	boundaryB := &countingBoundary{}
	origBoundary := &countingBoundary{addErr: errors.New("boundary replaced")}

	SetSessionBoundary(origBoundary)
	defer SetSessionBoundary(nil)

	conn := &websocket.Conn{}
	token := map[string]string{"id": "token-id", "token": "token-value"}

	const totalCalls = 160
	const totalAdds = totalCalls / 2
	const totalRemoves = totalCalls - totalAdds

	var wg sync.WaitGroup
	for i := 0; i < totalCalls; i++ {
		i := i
		wg.Add(2)
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				SetSessionBoundary(boundaryA)
			} else {
				SetSessionBoundary(boundaryB)
			}
		}()
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				if err := AddToAuthMap(conn, token, true); err != nil {
					t.Fatalf("unexpected add error: %v", err)
				}
				return
			}
			RemoveFromAuthMap(token["id"], true)
		}(i)
	}
	wg.Wait()

	boundaryAAdd, boundaryARemove := boundaryA.snapshot()
	boundaryBAdd, boundaryBRemove := boundaryB.snapshot()

	if boundaryAAdd+boundaryBAdd != totalAdds {
		t.Fatalf("expected %d add calls, got %d", totalAdds, boundaryAAdd+boundaryBAdd)
	}
	if boundaryARemove+boundaryBRemove != totalRemoves {
		t.Fatalf("expected %d remove calls, got %d", totalRemoves, boundaryARemove+boundaryBRemove)
	}
}

func TestSessionStoreAddToAuthMapRejectsBadInputs(t *testing.T) {
	conn := &websocket.Conn{}
	cm := session.NewClientManager()
	store := newSessionStore()
	store.getClientManager = func() *session.ClientManager { return cm }
	store.persistAuthorized = func(string, string, string) error { return nil }
	store.persistUnauthorized = func(string, string, string) error { return nil }
	store.now = func() time.Time { return time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC) }

	if err := store.AddToAuthMap(conn, nil, true); err == nil {
		t.Fatal("expected nil token to be rejected")
	}
	if err := store.AddToAuthMap(nil, map[string]string{"id": "t", "token": "v"}, true); err == nil {
		t.Fatal("expected nil websocket connection to be rejected")
	}
	if err := store.AddToAuthMap(conn, map[string]string{"id": "t"}, true); err == nil {
		t.Fatal("expected token without token string to be rejected")
	}
	if err := store.AddToAuthMap(conn, map[string]string{"token": "v"}, true); err == nil {
		t.Fatal("expected token without token ID to be rejected")
	}
}

func TestSessionStoreAddToAuthMapRollsBackOnPersistFailure(t *testing.T) {
	cm := session.NewClientManager()
	store := newSessionStore()
	store.getClientManager = func() *session.ClientManager { return cm }
	store.hashToken = func(token string) string { return "hash:" + token }
	store.now = func() time.Time { return time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC) }
	store.persistAuthorized = func(tokenID, hash, created string) error {
		if tokenID != "token-id" {
			t.Fatalf("unexpected token id, got %q", tokenID)
		}
		if hash != "hash:token-value" {
			t.Fatalf("unexpected token hash, got %q", hash)
		}
		if created != "2026-01-02_15:04:05" {
			t.Fatalf("unexpected created timestamp, got %q", created)
		}
		return errors.New("persist failed")
	}

	conn := &websocket.Conn{}
	err := store.AddToAuthMap(conn, map[string]string{"id": "token-id", "token": "token-value"}, true)
	if err == nil {
		t.Fatal("expected persist failure to be surfaced")
	}
	if len(cm.AuthClients["token-id"]) > 0 {
		t.Fatalf("expected rollback on persist failure, but auth map was mutated")
	}

	store.persistAuthorized = func(_ string, _ string, _ string) error { return nil }
	if err := store.AddToAuthMap(conn, map[string]string{"id": "token-id", "token": "token-value"}, true); err != nil {
		t.Fatalf("expected successful mutation after persist succeeds: %v", err)
	}
	if len(cm.AuthClients["token-id"]) != 1 {
		t.Fatalf("expected one auth client after successful persist")
	}
}

func TestSessionStoreAddToAuthMapConcurrentAddsAreDeduplicated(t *testing.T) {
	cm := session.NewClientManager()
	store := newSessionStore()
	store.getClientManager = func() *session.ClientManager { return cm }
	store.persistUnauthorized = func(_ string, _ string, _ string) error { return nil }
	store.persistAuthorized = func(_ string, _ string, _ string) error { return nil }
	conn := &websocket.Conn{}
	connToken := map[string]string{"id": "token-id", "token": "token-value"}

	const iterations = 200
	var wg sync.WaitGroup
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := store.AddToAuthMap(conn, connToken, false); err != nil {
				t.Fatalf("unexpected add error: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := len(cm.UnauthClients["token-id"]); got != 1 {
		t.Fatalf("expected single client entry after deduplicated concurrent adds, got %d", got)
	}
}
