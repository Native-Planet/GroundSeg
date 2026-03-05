package main

import (
	"context"
	"testing"
	"time"

	"groundseg/config"
)

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

func TestC2CCheckActivatesModeWhenOfflineOnNPBox(t *testing.T) {
	activated := false
	setModeCalled := false
	setModeValue := false
	killStarted := make(chan struct{}, 1)

	runtime := c2cTestRuntime(
		withC2CState(
			func() bool { return false },
			func() config.ConnectivitySettings { return config.ConnectivitySettings{C2CInterval: 42} },
			func() bool { return true },
			func() bool { return true },
			nil,
			func() error {
				activated = true
				return nil
			},
			func(enabled bool) error {
				setModeCalled = true
				setModeValue = enabled
				return nil
			},
			func(_ context.Context, _ func() config.ConnectivitySettings, _ func() error) {
				killStarted <- struct{}{}
			},
		),
	)

	if err := C2CCheckWith(context.Background(), runtime); err != nil {
		t.Fatalf("expected no error when enabling C2C mode, got %v", err)
	}

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

func TestC2CCheckSkipsWhenOnline(t *testing.T) {
	activated := false
	runtime := c2cTestRuntime(
		withC2CState(
			func() bool { return true },
			func() config.ConnectivitySettings { return config.ConnectivitySettings{} },
			func() bool { return true },
			func() bool { return true },
			nil,
			func() error {
				activated = true
				return nil
			},
			func(enabled bool) error { return nil },
			func(_ context.Context, _ func() config.ConnectivitySettings, _ func() error) {
				t.Fatal("killSwitch should not be started when internet is available")
			},
		),
	)

	if err := C2CCheckWith(context.Background(), runtime); err != nil {
		t.Fatalf("expected online checks to skip C2C mode with no error, got %v", err)
	}
	if activated {
		t.Fatal("C2CCheck should not activate C2C mode when internet is available")
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
