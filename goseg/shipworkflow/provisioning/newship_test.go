package provisioning

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"groundseg/shipcleanup"
	"groundseg/structs"
)

func TestHandleNewShipErrorCleanupPublishesTransitionsAndRollsBack(t *testing.T) {
	var events []structs.NewShipTransition
	var rollbackPatp string
	var rollbackOptions shipcleanup.RollbackOptions

	runtime := Runtime{
		PublishTransitionFn: func(_ context.Context, transition structs.NewShipTransition) {
			events = append(events, transition)
		},
		RollbackProvisioningFn: func(patp string, options shipcleanup.RollbackOptions) error {
			rollbackPatp = patp
			rollbackOptions = options
			return nil
		},
	}

	err := handleNewShipErrorCleanup(runtime, "~zod", "boom", "/tmp/custom")
	if err == nil {
		t.Fatal("expected cleanup error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected cleanup error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 transition events, got %d", len(events))
	}
	if events[0].Type != "bootStage" || events[0].Event != "aborted" {
		t.Fatalf("unexpected first transition: %+v", events[0])
	}
	if events[1].Type != "error" || !strings.Contains(events[1].Event, "boom") {
		t.Fatalf("unexpected second transition: %+v", events[1])
	}
	if rollbackPatp != "~zod" {
		t.Fatalf("unexpected rollback patp: %q", rollbackPatp)
	}
	if rollbackOptions.CustomPierPath == "" || !rollbackOptions.RemoveContainer || !rollbackOptions.RemoveContainerState {
		t.Fatalf("unexpected rollback options: %+v", rollbackOptions)
	}
}

func TestWaitForNewShipReadyBootPathPublishesCompletedEvent(t *testing.T) {
	var events []structs.NewShipTransition
	bootWaitCalled := false
	syncCalled := false

	runtime := Runtime{
		PublishTransitionFn: func(_ context.Context, transition structs.NewShipTransition) {
			events = append(events, transition)
		},
		WaitForBootCodeFn: func(_ string, _ time.Duration) {
			bootWaitCalled = true
		},
		WaitForRemoteReadyFn: func(_ string, _ time.Duration) {},
		SwitchShipToWireguardFn: func(_ string, _ bool) error {
			return nil
		},
		SyncRetrieveFn: func() error {
			syncCalled = true
			return nil
		},
		ConfigFn: func() structs.SysConfig {
			return structs.SysConfig{}
		},
		StartLlamaAPIFn: func() {},
	}

	waitForNewShipReady(runtime, structs.WsNewShipPayload{
		Payload: structs.WsNewShipAction{
			Patp:   "~zod",
			Remote: false,
		},
	}, "")

	if !bootWaitCalled {
		t.Fatal("expected wait for boot code to be called")
	}
	if !syncCalled {
		t.Fatal("expected sync retrieve to be called")
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 transition events, got %d", len(events))
	}
	if events[0].Event != "booting" || events[1].Event != "completed" {
		t.Fatalf("unexpected transitions: %+v", events)
	}
}

func TestProvisionShipReturnsCleanupErrorWhenCreateConfigFails(t *testing.T) {
	expected := errors.New("create config failed")
	var rollbackCalled bool

	runtime := Runtime{
		CreateUrbitConfigFn: func(string, string) error { return expected },
		PublishTransitionFn: func(context.Context, structs.NewShipTransition) {},
		RollbackProvisioningFn: func(string, shipcleanup.RollbackOptions) error {
			rollbackCalled = true
			return nil
		},
	}

	err := ProvisionShip(runtime, "~zod", structs.WsNewShipPayload{
		Payload: structs.WsNewShipAction{
			Patp: "~zod",
			Key:  "sample-key",
		},
	}, "")
	if err == nil {
		t.Fatal("expected provisioning failure")
	}
	if !strings.Contains(err.Error(), "failed to create urbit config") {
		t.Fatalf("unexpected provisioning error: %v", err)
	}
	if !rollbackCalled {
		t.Fatal("expected rollback to be invoked on provisioning failure")
	}
}
