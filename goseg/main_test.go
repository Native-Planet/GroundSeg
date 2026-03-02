package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"groundseg/config"
)

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
	callCount := 0
	if !connCheckWith(connectivityCheckRuntime{
		netCheck: func(string) bool {
			callCount++
			return true
		},
		timeout:  c2cCheckTimeout,
		interval: c2cCheckInterval,
	}) {
		t.Fatal("expected connCheck to return true on first success")
	}
	if callCount != 1 {
		t.Fatalf("expected one netcheck call, got %d", callCount)
	}
}

func TestConnCheckRetriesThenSucceeds(t *testing.T) {
	callCount := 0
	if !connCheckWith(connectivityCheckRuntime{
		netCheck: func(string) bool {
			callCount++
			return callCount >= 3
		},
		timeout:  40 * time.Millisecond,
		interval: 1 * time.Millisecond,
	}) {
		t.Fatal("expected connCheck to return true after retries")
	}
	if callCount < 3 {
		t.Fatalf("expected at least three netcheck calls, got %d", callCount)
	}
}

func TestConnCheckTimeout(t *testing.T) {
	if connCheckWith(connectivityCheckRuntime{
		netCheck: func(string) bool {
			return false
		},
		timeout:  5 * time.Millisecond,
		interval: 1 * time.Millisecond,
	}) {
		t.Fatal("expected connCheck to return false on timeout")
	}
}

func TestC2cCheckActivatesModeWhenOfflineOnNPBox(t *testing.T) {
	activated := false
	setModeCalled := false
	setModeValue := false
	killStarted := make(chan struct{}, 1)

	runtime := c2cRuntime{
		device: c2cDeviceRuntime{
			isNPBox:   func() bool { return true },
			hasDevice: func() bool { return true },
			wifiInfo:  nil,
		},
		connectivity: c2cConnectivityRuntime{
			connCheck: func() bool {
				return false
			},
			settingsSnap: func() config.ConnectivitySettings {
				return config.ConnectivitySettings{C2cInterval: 42}
			},
		},
		mode: c2cModeRuntime{
			isC2cMode: func() error {
			activated = true
			return nil
		},
			setC2cMode: func(enabled bool) error {
			setModeCalled = true
			setModeValue = enabled
			return nil
		},
			startKillSwitch: func(context.Context, func() config.ConnectivitySettings) {
			killStarted <- struct{}{}
		},
		},
	}

	C2cCheckWith(context.Background(), runtime)

	if !activated {
		t.Fatal("expected C2C mode activation call")
	}
	if !setModeCalled || !setModeValue {
		t.Fatalf("expected setC2CMode(true), called=%v value=%v", setModeCalled, setModeValue)
	}
	select {
	case <-killStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("expected killSwitch to be started")
	}
}

func TestC2cCheckSkipsWhenOnline(t *testing.T) {
	activated := false
		runtime := c2cRuntime{
			device: c2cDeviceRuntime{
				isNPBox:   func() bool { return true },
				hasDevice: func() bool { return true },
				wifiInfo:  nil,
			},
			connectivity: c2cConnectivityRuntime{
				connCheck: func() bool { return true },
				settingsSnap: func() config.ConnectivitySettings {
					return config.ConnectivitySettings{}
				},
			},
			mode: c2cModeRuntime{
				isC2cMode: func() error {
				activated = true
			return nil
		},
			setC2cMode: func(enabled bool) error { return nil },
			startKillSwitch: func(context.Context, func() config.ConnectivitySettings) {
			t.Fatal("killSwitch should not be started when internet is available")
		},
		},
	}

	C2cCheckWith(context.Background(), runtime)
	if activated {
		t.Fatal("C2cCheck should not activate C2C mode when internet is available")
	}
}

func TestKillSwitchStopsWhenContextIsDone(t *testing.T) {
	originalDebugMode := config.DebugMode()
	config.SetDebugMode(false)
	t.Cleanup(func() {
		config.SetDebugMode(originalDebugMode)
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		killSwitch(ctx, config.ConnectivitySettingsSnapshot)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected killSwitch to stop when context is done")
	}
}

func TestBootstrapSubsystemsForwardsCallbacks(t *testing.T) {
	const expectedPort = 61234
	var gotPort int
	gotServer := false
	gotC2c := false
	runtime := bootstrapRuntimeWith(
		func(_ context.Context, opts StartupOptions) error {
			if opts.HTTPPort != expectedPort {
				t.Fatalf("expected startup port %d, got %d", expectedPort, opts.HTTPPort)
			}
			if opts.StartServer == nil {
				t.Fatal("expected StartServer callback")
			}
			if opts.StartC2cCheck == nil {
				t.Fatal("expected StartC2cCheck callback")
			}
			if err := opts.StartServer(context.Background(), opts.HTTPPort); err != nil {
				return err
			}
			opts.StartC2cCheck(context.Background())
			return nil
		},
		func(_ context.Context, httpPort int) error {
			gotPort = httpPort
			gotServer = true
			return nil
		},
		func(context.Context) {
			gotC2c = true
		},
	)

	if err := runBootstrapSubsystems(context.Background(), appStartupOptions{httpPort: expectedPort}, runtime); err != nil {
		t.Fatalf("bootstrapSubsystems returned error: %v", err)
	}
	if !gotServer {
		t.Fatal("expected start server callback to be exercised")
	}
	if gotPort != expectedPort {
		t.Fatalf("expected server callback port %d, got %d", expectedPort, gotPort)
	}
	if !gotC2c {
		t.Fatal("expected c2c callback to be exercised")
	}
}

func TestBootstrapSubsystemsPropagatesBootstrapError(t *testing.T) {
	expectedErr := errors.New("bootstrap failed")
	runtime := bootstrapRuntimeWith(
		func(context.Context, StartupOptions) error { return expectedErr },
		func(context.Context, int) error { return nil },
		nil,
	)

	err := runBootstrapSubsystems(context.Background(), appStartupOptions{}, runtime)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected bootstrap error %v, got %v", expectedErr, err)
	}
}

func TestStartServerReturnsNilWhenServersExitNormally(t *testing.T) {
	runtime := defaultServerRuntime()

	var httpCalls int64
	var wsCalls int64
	runtime.listenAndServe = func(server *http.Server) error {
		switch server.Addr {
		case ":3000":
			atomic.AddInt64(&wsCalls, 1)
		default:
			atomic.AddInt64(&httpCalls, 1)
		}
		return nil
	}
	runtime.shutdown = func(_ context.Context, _ *http.Server) error {
		return nil
	}

	if err := runServer(context.Background(), 8123, runtime); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if atomic.LoadInt64(&httpCalls) == 0 || atomic.LoadInt64(&wsCalls) == 0 {
		t.Fatalf("expected both listeners to run, got http=%d ws=%d", httpCalls, wsCalls)
	}
}

func TestStartServerReturnsErrorWhenListenerFails(t *testing.T) {
	runtime := defaultServerRuntime()

	shutdownSig := make(chan struct{}, 1)
	var once sync.Once
	expectedErr := errors.New("http listener failure")
	runtime.listenAndServe = func(server *http.Server) error {
		if server.Addr == ":3000" {
			<-shutdownSig
			return http.ErrServerClosed
		}
		return expectedErr
	}
	runtime.shutdown = func(_ context.Context, _ *http.Server) error {
		once.Do(func() { close(shutdownSig) })
		return nil
	}

	err := runServer(context.Background(), 8124, runtime)
	if err == nil {
		t.Fatal("expected startServer to return listener failure")
	}
	if !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Fatalf("expected wrapped listener error, got %v", err)
	}
}

func TestStartServerReturnsUnexpectedErrorWhenOtherListenerShutsDownGracefully(t *testing.T) {
	runtime := defaultServerRuntime()

	runtime.listenAndServe = func(server *http.Server) error {
		if server.Addr == ":3000" {
			return http.ErrServerClosed
		}
		return errors.New("unexpected listener failure")
	}
	runtime.shutdown = func(_ context.Context, _ *http.Server) error {
		return nil
	}

	err := runServer(context.Background(), 8126, runtime)
	if err == nil {
		t.Fatal("expected listener failure to be surfaced")
	}
	if !strings.Contains(err.Error(), "unexpected listener failure") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartServerStopsWhenContextIsDone(t *testing.T) {
	runtime := defaultServerRuntime()

	started := make(chan struct{}, 2)
	stopCh := make(chan struct{})
	var shutdownOnce sync.Once
	runtime.listenAndServe = func(_ *http.Server) error {
		started <- struct{}{}
		<-stopCh
		return nil
	}
	runtime.shutdown = func(_ context.Context, _ *http.Server) error {
		shutdownOnce.Do(func() { close(stopCh) })
		return nil
	}

	done := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		done <- runServer(ctx, 8125, runtime)
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for server startup")
		}
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil on context cancellation, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for startServer to return")
	}
}
