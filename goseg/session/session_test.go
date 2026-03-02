package session

import (
	"reflect"
	"testing"

	"github.com/gorilla/websocket"
)

func TestRemoveSysLogSessionsDropsQueuedConnections(t *testing.T) {
	originalSessions := SysLogSessions()
	originalToRemove := SysSessionsToRemove()
	t.Cleanup(func() {
		SetSysLogSessions(originalSessions)
		SetSysSessionsToRemove(originalToRemove)
	})

	a := &websocket.Conn{}
	b := &websocket.Conn{}
	c := &websocket.Conn{}
	SetSysLogSessions([]*websocket.Conn{a, b, c})
	SetSysSessionsToRemove([]*websocket.Conn{b})

	RemoveSysLogSessions()

	if !reflect.DeepEqual(SysLogSessions(), []*websocket.Conn{a, c}) {
		t.Fatalf("unexpected remaining sessions: %+v", SysLogSessions())
	}
	if len(SysSessionsToRemove()) != 0 {
		t.Fatalf("expected SysSessionsToRemove to be cleared, got %+v", SysSessionsToRemove())
	}
}
