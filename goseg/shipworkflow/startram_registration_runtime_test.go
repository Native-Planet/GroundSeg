package shipworkflow

import (
	"errors"
	"strings"
	"testing"

	"groundseg/config"
	"groundseg/structs"
)

func TestRunStartramSetBackupPasswordWithRuntimeUpdatesConfigPatch(t *testing.T) {
	t.Parallel()

	var gotPassword string
	runtime := defaultStartramRuntime()
	runtime.UpdateConfigTypedFn = func(opts ...config.ConfigUpdateOption) error {
		if len(opts) != 1 {
			t.Fatalf("expected one config update option, got %d", len(opts))
		}
		patch := &config.ConfPatch{}
		for _, opt := range opts {
			opt(patch)
		}
		if patch.RemoteBackupPassword == nil {
			t.Fatal("expected remote backup password patch to be set")
		}
		gotPassword = *patch.RemoteBackupPassword
		return nil
	}
	runtime.PublishEventFn = func(structs.Event) {}

	if err := runStartramSetBackupPasswordWithRuntime(runtime, "backup-secret"); err != nil {
		t.Fatalf("runStartramSetBackupPasswordWithRuntime returned error: %v", err)
	}
	if gotPassword != "backup-secret" {
		t.Fatalf("unexpected backup password patch value: %q", gotPassword)
	}
}

func TestRunStartramRegisterWithRuntimeReturnsCycleKeyError(t *testing.T) {
	t.Parallel()

	runtime := defaultStartramRuntime()
	runtime.CycleWgKeyFn = func() error { return errors.New("cycle key failed") }
	runtime.PublishEventFn = func(structs.Event) {}

	err := runStartramRegisterWithRuntime(runtime, "reg-code", "ap-southeast")
	if err == nil {
		t.Fatal("expected cycle key failure to abort register flow")
	}
	if !strings.Contains(err.Error(), "cycle wireguard key") {
		t.Fatalf("unexpected register error: %v", err)
	}
}
