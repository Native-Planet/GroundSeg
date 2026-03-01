package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"groundseg/config"
	"groundseg/system"
)

func resetMainTestSeams() func() {
	originalNetCheckFn := netCheckFn
	originalConnCheckFn := connCheckFn
	originalConnTimeout := connTimeout
	originalConnInterval := connInterval
	originalIsNPBoxFn := isNPBoxFn
	originalC2CModeFn := c2cModeFn
	originalSetC2CModeFn := setC2CModeFn
	originalKillSwitchFn := killSwitchFn
	return func() {
		netCheckFn = originalNetCheckFn
		connCheckFn = originalConnCheckFn
		connTimeout = originalConnTimeout
		connInterval = originalConnInterval
		isNPBoxFn = originalIsNPBoxFn
		c2cModeFn = originalC2CModeFn
		setC2CModeFn = originalSetC2CModeFn
		killSwitchFn = originalKillSwitchFn
	}
}

func TestLoadServiceRunsFunction(t *testing.T) {
	done := make(chan struct{}, 1)
	loadService(func() error {
		done <- struct{}{}
		return nil
	}, "unused")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loadService did not execute function")
	}
}

func TestContentTypeSetterAppliesMimeType(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	recorder := httptest.NewRecorder()

	ContentTypeSetter(next).ServeHTTP(recorder, req)

	contentType := recorder.Header().Get("Content-Type")
	if !strings.Contains(contentType, "javascript") {
		t.Fatalf("expected javascript mime type, got %q", contentType)
	}
}

func TestFallbackToIndexServesIndexForMissingPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("INDEX"), 0o644); err != nil {
		t.Fatalf("write index file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "about.txt"), []byte("ABOUT"), 0o644); err != nil {
		t.Fatalf("write about file: %v", err)
	}

	handler := fallbackToIndex(http.FS(os.DirFS(dir)))

	missingReq := httptest.NewRequest(http.MethodGet, "/missing-page", nil)
	missingRes := httptest.NewRecorder()
	handler(missingRes, missingReq)
	if missingReq.URL.Path != "/index.html" {
		t.Fatalf("expected request path rewrite to /index.html, got %q", missingReq.URL.Path)
	}
	if missingRes.Code == http.StatusNotFound {
		t.Fatalf("unexpected 404 for fallback response: code=%d", missingRes.Code)
	}

	existingReq := httptest.NewRequest(http.MethodGet, "/about.txt", nil)
	existingRes := httptest.NewRecorder()
	handler(existingRes, existingReq)
	if !strings.Contains(existingRes.Body.String(), "ABOUT") {
		t.Fatalf("expected existing file body, got %q", existingRes.Body.String())
	}
}

func TestConnCheckImmediateSuccess(t *testing.T) {
	restore := resetMainTestSeams()
	t.Cleanup(restore)

	callCount := 0
	netCheckFn = func(string) bool {
		callCount++
		return true
	}

	if !connCheck() {
		t.Fatal("expected connCheck to return true on first success")
	}
	if callCount != 1 {
		t.Fatalf("expected one netcheck call, got %d", callCount)
	}
}

func TestConnCheckRetriesThenSucceeds(t *testing.T) {
	restore := resetMainTestSeams()
	t.Cleanup(restore)

	connTimeout = 40 * time.Millisecond
	connInterval = 1 * time.Millisecond
	callCount := 0
	netCheckFn = func(string) bool {
		callCount++
		return callCount >= 3
	}

	if !connCheck() {
		t.Fatal("expected connCheck to return true after retries")
	}
	if callCount < 3 {
		t.Fatalf("expected at least three netcheck calls, got %d", callCount)
	}
}

func TestConnCheckTimeout(t *testing.T) {
	restore := resetMainTestSeams()
	t.Cleanup(restore)

	connTimeout = 5 * time.Millisecond
	connInterval = 1 * time.Millisecond
	netCheckFn = func(string) bool {
		return false
	}

	if connCheck() {
		t.Fatal("expected connCheck to return false on timeout")
	}
}

func TestC2cCheckActivatesModeWhenOfflineOnNPBox(t *testing.T) {
	restore := resetMainTestSeams()
	t.Cleanup(restore)

	conf := config.Conf()

	originalDevice := system.Device
	system.Device = "wlan0"
	t.Cleanup(func() {
		system.Device = originalDevice
	})

	connCheckFn = func() bool { return false }
	isNPBoxFn = func(string) bool { return true }
	activated := false
	c2cModeFn = func() error {
		activated = true
		return nil
	}
	setC2CModeCalled := false
	setC2CModeValue := false
	setC2CModeFn = func(enabled bool) error {
		setC2CModeCalled = true
		setC2CModeValue = enabled
		return nil
	}
	killStarted := make(chan struct{}, 1)
	killSwitchFn = func() {
		killStarted <- struct{}{}
	}

	C2cCheck()

	if !activated {
		t.Fatal("expected C2C mode activation call")
	}
	if !setC2CModeCalled || !setC2CModeValue {
		t.Fatalf("expected setC2CMode(true), called=%v value=%v", setC2CModeCalled, setC2CModeValue)
	}
	if conf.C2cInterval > 0 {
		select {
		case <-killStarted:
		case <-time.After(2 * time.Second):
			t.Fatal("expected killSwitch to be started")
		}
	}
}

func TestC2cCheckSkipsWhenOnline(t *testing.T) {
	restore := resetMainTestSeams()
	t.Cleanup(restore)

	connCheckFn = func() bool { return true }
	isNPBoxFn = func(string) bool { return true }
	activated := false
	c2cModeFn = func() error {
		activated = true
		return nil
	}

	C2cCheck()
	if activated {
		t.Fatal("C2cCheck should not activate C2C mode when internet is available")
	}
}
