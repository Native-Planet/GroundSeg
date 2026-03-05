package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

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

	if err := runServer(context.Background(), 8123, 3000, runtime); err != nil {
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

	err := runServer(context.Background(), 8124, 3000, runtime)
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

	err := runServer(context.Background(), 8126, 3000, runtime)
	if err == nil {
		t.Fatal("expected listener failure to be surfaced")
	}
	if !strings.Contains(err.Error(), "unexpected listener failure") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartServerReturnsAllUnexpectedListenerErrors(t *testing.T) {
	runtime := defaultServerRuntime()

	runtime.listenAndServe = func(server *http.Server) error {
		if server.Addr == ":3000" {
			return errors.New("websocket listener failed")
		}
		return errors.New("http listener failed")
	}
	runtime.shutdown = func(_ context.Context, _ *http.Server) error {
		return nil
	}

	err := runServer(context.Background(), 8127, 3000, runtime)
	if err == nil {
		t.Fatal("expected listener failures to be surfaced")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "http listener failed") || !strings.Contains(errMsg, "websocket listener failed") {
		t.Fatalf("expected both listener errors in joined result, got: %v", err)
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
		done <- runServer(ctx, 8125, 3000, runtime)
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
