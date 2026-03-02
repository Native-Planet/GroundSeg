package session

import (
	"reflect"
	"testing"

	"github.com/gorilla/websocket"
)

func TestRemoveSysLogSessionsDropsQueuedConnections(t *testing.T) {
	systemRuntime := LogstreamRuntimeState()
	originalSessions := systemRuntime.SysLogSessions()
	originalToRemove := systemRuntime.SysSessionsToRemove()
	t.Cleanup(func() {
		systemRuntime.SetSysLogSessions(originalSessions)
		systemRuntime.SetSysSessionsToRemove(originalToRemove)
	})

	a := &websocket.Conn{}
	b := &websocket.Conn{}
	c := &websocket.Conn{}
	systemRuntime.SetSysLogSessions([]*websocket.Conn{a, b, c})
	systemRuntime.SetSysSessionsToRemove([]*websocket.Conn{b})

	systemRuntime.RemoveSysLogSessions()

	if !reflect.DeepEqual(systemRuntime.SysLogSessions(), []*websocket.Conn{a, c}) {
		t.Fatalf("unexpected remaining sessions: %+v", systemRuntime.SysLogSessions())
	}
	if len(systemRuntime.SysSessionsToRemove()) != 0 {
		t.Fatalf("expected SysSessionsToRemove to be cleared, got %+v", systemRuntime.SysSessionsToRemove())
	}
}
