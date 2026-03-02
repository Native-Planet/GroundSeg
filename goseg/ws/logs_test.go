package ws

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"groundseg/session"
	"groundseg/structs"
	"groundseg/testutil"

	"github.com/docker/docker/api/types/container"
	"github.com/gorilla/websocket"
)

func resetLogsHandlerSeamsForTest(t *testing.T) {
	t.Helper()
	origLogCheck := logTokenCheckForLogs
	origRetrieve := retrieveSysLogHistoryForLogs
	origFind := findContainerForLogs
	origSysSessions := session.LogstreamRuntimeState().SysLogSessions()
	origDockerSessions := session.LogstreamRuntimeState().DockerLogSessions()
	t.Cleanup(func() {
		logTokenCheckForLogs = origLogCheck
		retrieveSysLogHistoryForLogs = origRetrieve
		findContainerForLogs = origFind
		session.LogstreamRuntimeState().SetSysLogSessions(origSysSessions)
		session.LogstreamRuntimeState().SetDockerLogSessions(origDockerSessions)
	})

	session.LogstreamRuntimeState().SetSysLogSessions([]*websocket.Conn{})
	session.LogstreamRuntimeState().SetDockerLogSessions(map[string]map[*websocket.Conn]bool{})
}

func wsURLFromHTTP(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}

func TestLogsHandlerSystemRequestReturnsHistoryAndRegistersSession(t *testing.T) {
	resetLogsHandlerSeamsForTest(t)
	logTokenCheckForLogs = func(token structs.WsTokenStruct, r *http.Request) bool {
		return true
	}
	retrieveSysLogHistoryForLogs = func() ([]byte, error) {
		return []byte(`{"type":"system","history":true,"log":[]}`), nil
	}

	server := httptest.NewServer(http.HandlerFunc(LogsHandler))
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURLFromHTTP(server.URL), nil)
	if err != nil {
		t.Fatalf("failed to dial logs websocket: %v", err)
	}
	defer conn.Close()

	payload := LogPayload{Type: "system", Token: structs.WsTokenStruct{ID: "id", Token: "tok"}}
	msg, _ := json.Marshal(payload)
	if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.Fatalf("failed to write logs request: %v", err)
	}

	_, historyMsg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read history response: %v", err)
	}
	if string(historyMsg) != `{"type":"system","history":true,"log":[]}` {
		t.Fatalf("unexpected history response: %s", historyMsg)
	}

	testutil.WaitForCondition(t, func() bool {
		return len(session.LogstreamRuntimeState().SysLogSessions()) == 1
	}, "expected websocket session to be registered for system logs")
}

func TestLogsHandlerContainerRequestTracksDockerSession(t *testing.T) {
	resetLogsHandlerSeamsForTest(t)
	logTokenCheckForLogs = func(token structs.WsTokenStruct, r *http.Request) bool {
		return true
	}
	findContainerForLogs = func(name string) (*container.Summary, error) {
		if name != "vere" {
			t.Fatalf("unexpected container lookup: %s", name)
		}
		return nil, nil
	}

	server := httptest.NewServer(http.HandlerFunc(LogsHandler))
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURLFromHTTP(server.URL), nil)
	if err != nil {
		t.Fatalf("failed to dial logs websocket: %v", err)
	}
	defer conn.Close()

	payload := LogPayload{Type: "vere", Token: structs.WsTokenStruct{ID: "id", Token: "tok"}}
	msg, _ := json.Marshal(payload)
	if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.Fatalf("failed to write logs request: %v", err)
	}

	testutil.WaitForCondition(t, func() bool {
		sessions, exists := session.LogstreamRuntimeState().DockerLogSessions()["vere"]
		return exists && len(sessions) == 1
	}, "expected docker log session to be tracked for container request")
}

func TestLogsHandlerClosesConnectionForUnauthenticatedRequest(t *testing.T) {
	resetLogsHandlerSeamsForTest(t)
	logTokenCheckForLogs = func(token structs.WsTokenStruct, r *http.Request) bool {
		return false
	}
	retrieveSysLogHistoryForLogs = func() ([]byte, error) {
		return nil, errors.New("should not be called")
	}

	server := httptest.NewServer(http.HandlerFunc(LogsHandler))
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURLFromHTTP(server.URL), nil)
	if err != nil {
		t.Fatalf("failed to dial logs websocket: %v", err)
	}
	defer conn.Close()

	payload := LogPayload{Type: "system", Token: structs.WsTokenStruct{ID: "id", Token: "tok"}}
	msg, _ := json.Marshal(payload)
	if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.Fatalf("failed to write logs request: %v", err)
	}

	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("expected websocket read to fail after unauthenticated request")
	}
}
