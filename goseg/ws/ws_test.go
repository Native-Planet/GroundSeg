//go:build integration

package ws

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWsHandlerRejectsNonWebSocketUpgrade(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.local/ws", nil)
	rr := httptest.NewRecorder()

	WsHandler(rr, req)

	if rr.Code == http.StatusSwitchingProtocols {
		t.Fatalf("expected non-upgrade request to be rejected, got %d", rr.Code)
	}
}
