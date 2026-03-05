package main

import (
	"context"
	"errors"
	"testing"

	"groundseg/startuporchestrator"
)

func TestBootstrapSubsystemsForwardsCallbacks(t *testing.T) {
	const expectedPort = 61234
	var gotPort int
	gotServer := false
	gotC2C := false
	runtime := bootstrapRuntimeWith(
		func(_ context.Context, opts startuporchestrator.StartupOptions) error {
			if opts.HTTPPort != expectedPort {
				t.Fatalf("expected startup port %d, got %d", expectedPort, opts.HTTPPort)
			}
			if opts.StartServer == nil {
				t.Fatal("expected StartServer callback")
			}
			if opts.StartC2CCheck == nil {
				t.Fatal("expected StartC2CCheck callback")
			}
			if err := opts.StartServer(context.Background(), opts.HTTPPort, 3000); err != nil {
				return err
			}
			if err := opts.StartC2CCheck(context.Background()); err != nil {
				return err
			}
			return nil
		},
		func(_ context.Context, httpPort int, websocketPort int) error {
			gotPort = httpPort
			if websocketPort != 3000 {
				t.Fatalf("expected websocket port %d, got %d", 3000, websocketPort)
			}
			gotServer = true
			return nil
		},
		func(context.Context) error {
			gotC2C = true
			return nil
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
	if !gotC2C {
		t.Fatal("expected c2c callback to be exercised")
	}
}

func TestBootstrapSubsystemsForwardsStartupRuntime(t *testing.T) {
	startupRuntime := startuporchestrator.StartupRuntime{}
	startupRuntime.StartConfigEventLoopFn = func(context.Context) error { return nil }
	gotStartupRuntime := false

	runtime := bootstrapRuntimeWith(
		func(_ context.Context, opts startuporchestrator.StartupOptions) error {
			if opts.StartupRuntime.StartConfigEventLoopFn == nil {
				t.Fatal("expected startup runtime to be forwarded into startup options")
			}
			gotStartupRuntime = true
			return opts.StartupRuntime.StartConfigEventLoopFn(context.Background())
		},
		func(_ context.Context, _ int, _ int) error { return nil },
		func(context.Context) error { return nil },
		startupRuntime,
	)

	if err := runBootstrapSubsystems(context.Background(), appStartupOptions{}, runtime); err != nil {
		t.Fatalf("bootstrapSubsystems returned error: %v", err)
	}
	if !gotStartupRuntime {
		t.Fatal("expected startup runtime callback to be invoked")
	}
}

func TestBootstrapSubsystemsPropagatesBootstrapError(t *testing.T) {
	expectedErr := errors.New("bootstrap failed")
	runtime := bootstrapRuntimeWith(
		func(context.Context, startuporchestrator.StartupOptions) error { return expectedErr },
		func(context.Context, int, int) error { return nil },
		func(context.Context) error { return nil },
	)

	err := runBootstrapSubsystems(context.Background(), appStartupOptions{}, runtime)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected bootstrap error %v, got %v", expectedErr, err)
	}
}
