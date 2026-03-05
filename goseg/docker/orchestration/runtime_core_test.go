package orchestration

import (
	"strings"
	"testing"
)

func TestStartupRuntimeInitializeRequiresCallback(t *testing.T) {
	var runtime StartupRuntime
	err := runtime.Initialize()
	if err == nil {
		t.Fatal("expected Initialize to fail when callback is unset")
	}
	if !strings.Contains(err.Error(), "startup initialize callback") {
		t.Fatalf("unexpected missing callback error: %v", err)
	}
}

func TestStartupRuntimeInitializeInvokesCallback(t *testing.T) {
	called := false
	runtime := StartupRuntime{
		StartupBootstrapOps: StartupBootstrapOps{
			InitializeFn: func() error {
				called = true
				return nil
			},
		},
	}
	if err := runtime.Initialize(); err != nil {
		t.Fatalf("expected Initialize to succeed: %v", err)
	}
	if !called {
		t.Fatal("expected Initialize callback to be invoked")
	}
}
