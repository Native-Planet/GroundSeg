package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"groundseg/config"
)

type c2cRuntimeOption func(*c2cRuntime)

func c2cTestRuntime(overrides ...c2cRuntimeOption) c2cRuntime {
	runtime := defaultC2CRuntime()
	for _, apply := range overrides {
		if apply == nil {
			continue
		}
		apply(&runtime)
	}
	return runtime
}

func withC2CState(
	connCheck func() bool,
	settings func() config.ConnectivitySettings,
	isNPBoxFn func() bool,
	hasDeviceFn func() bool,
	wifiInfoFn func() (bool, error),
	isC2CModeFn func() error,
	setC2CModeFn func(bool) error,
	startKillSwitchFn func(context.Context, func() config.ConnectivitySettings, func() error),
) c2cRuntimeOption {
	return func(runtime *c2cRuntime) {
		runtime.connectivity = c2cConnectivityRuntime{
			connCheck:    connCheck,
			settingsSnap: settings,
		}
		runtime.device = c2cDeviceRuntime{
			isNPBox:   isNPBoxFn,
			hasDevice: hasDeviceFn,
			wifiInfo:  wifiInfoFn,
		}
		runtime.mode = c2cModeRuntime{
			isC2CMode:       isC2CModeFn,
			setC2CMode:      setC2CModeFn,
			startKillSwitch: startKillSwitchFn,
			restartFn:       func() error { return nil },
		}
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

func TestParseStartupOptionsReturnsErrorOnInvalidPort(t *testing.T) {
	if _, err := parseStartupOptions([]string{"--http-port=not-a-number"}); err == nil {
		t.Fatal("expected invalid http port to error")
	}

	if _, err := parseStartupOptions([]string{"--ws-port=90000"}); err == nil {
		t.Fatal("expected out-of-range websocket port to error")
	}
}

func TestParseStartupOptionsParsesValidPortsAndDevMode(t *testing.T) {
	opts, err := parseStartupOptions([]string{"dev", "--http-port=8080", "--ws-port=3001"})
	if err != nil {
		t.Fatalf("parseStartupOptions returned unexpected error: %v", err)
	}
	if !opts.devMode || opts.httpPort != 8080 || opts.websocketPort != 3001 {
		t.Fatalf("unexpected parsed options: %#v", opts)
	}
}
