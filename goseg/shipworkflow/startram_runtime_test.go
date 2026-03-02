package shipworkflow

import (
	"strings"
	"testing"

	"groundseg/config"
	"groundseg/docker/orchestration"
	"groundseg/structs"
)

func TestDefaultDispatchUrbitPayloadRoutesToggleNetwork(t *testing.T) {
	t.Parallel()

	originalDispatch := dispatchStartramToggleNetworkFn
	defer func() {
		dispatchStartramToggleNetworkFn = originalDispatch
	}()

	var gotPatp string
	dispatchStartramToggleNetworkFn = func(patp string) error {
		gotPatp = patp
		return nil
	}

	payload := structs.WsUrbitPayload{
		Payload: structs.WsUrbitAction{
			Action: "toggle-network",
			Patp:   "~zod",
		},
	}
	if err := defaultDispatchUrbitPayload(payload); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if gotPatp != "~zod" {
		t.Fatalf("expected patp %q, got %q", "~zod", gotPatp)
	}
}

func TestDefaultDispatchUrbitPayloadValidatesAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pay  structs.WsUrbitPayload
		sub  string
	}{
		{
			name: "missing action",
			pay: structs.WsUrbitPayload{
				Payload: structs.WsUrbitAction{
					Patp: "~zod",
				},
			},
			sub: "no urbit action",
		},
		{
			name: "missing patp",
			pay: structs.WsUrbitPayload{
				Payload: structs.WsUrbitAction{
					Action: "toggle-network",
				},
			},
			sub: "no patp provided",
		},
		{
			name: "unsupported action",
			pay: structs.WsUrbitPayload{
				Payload: structs.WsUrbitAction{
					Action: "unsupported",
					Patp:   "~zod",
				},
			},
			sub: "unsupported urbit action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := defaultDispatchUrbitPayload(tt.pay)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.sub) {
				t.Fatalf("expected error containing %q, got %q", tt.sub, err.Error())
			}
		})
	}
}

func TestHandleStartramToggleDispatchesPerShipWithLocalPayload(t *testing.T) {
	originalRuntimeFn := startramRuntimeFn
	originalToggleNetworkFn := dispatchStartramToggleNetworkFn
	originalDispatchUrbitPayloadFn := startramDispatchUrbitPayloadFn

	var dispatched []string
	var updateConfCalled bool

	dispatchStartramToggleNetworkFn = func(patp string) error {
		dispatched = append(dispatched, patp)
		return nil
	}
	startramDispatchUrbitPayloadFn = func(payload structs.WsUrbitPayload) error {
		return dispatchStartramToggleNetworkFn(payload.Payload.Patp)
	}

	runtime := orchestration.NewRuntime(
		orchestration.WithSnapshotOps(orchestration.RuntimeSnapshotOps{
			StartramSettingsSnapshotFn: func() config.StartramSettings {
				return config.StartramSettings{
					WgOn:  true,
					Piers: []string{"~zod", "~bus", "~nec"},
				}
			},
		}),
		orchestration.WithUrbitOps(orchestration.RuntimeUrbitOps{
			UrbitConfFn: func(patp string) structs.UrbitDocker {
				return structs.UrbitDocker{Network: "wireguard"}
			},
		}),
		orchestration.WithConfigOps(orchestration.RuntimeConfigOps{
			UpdateConfTypedFn: func(...config.ConfUpdateOption) error {
				updateConfCalled = true
				return nil
			},
		}),
		orchestration.WithContainerOps(orchestration.RuntimeContainerOps{
			StopContainerByNameFn: func(string) error { return nil },
			DeleteContainerFn:     func(string) error { return nil },
		}),
		orchestration.WithLoadOps(orchestration.RuntimeLoadOps{
			LoadMCFn:     func() error { return nil },
			LoadMinIOsFn: func() error { return nil },
		}),
	)

	startramRuntimeFn = func() orchestration.StartramRuntime {
		return orchestration.StartramRuntime{
			StartramSettingsSnapshotFn: runtime.StartramSettingsSnapshotFn,
			UrbitConfFn:                runtime.UrbitConfFn,
			UpdateConfTypedFn:          runtime.UpdateConfTypedFn,
			StopContainerByNameFn:      runtime.StopContainerByNameFn,
			DeleteContainerFn:          runtime.DeleteContainerFn,
			LoadMCFn:                   runtime.LoadMCFn,
			LoadMinIOsFn:               runtime.LoadMinIOsFn,
		}
	}
	defer func() {
		startramRuntimeFn = originalRuntimeFn
		startramDispatchUrbitPayloadFn = originalDispatchUrbitPayloadFn
		dispatchStartramToggleNetworkFn = originalToggleNetworkFn
	}()

	if err := HandleStartramToggle(); err != nil {
		t.Fatalf("HandleStartramToggle() returned error: %v", err)
	}

	if !updateConfCalled {
		t.Fatal("expected update conf to disable wireguard")
	}
	if len(dispatched) != 3 {
		t.Fatalf("expected 3 dispatches, got %d", len(dispatched))
	}
	expected := []string{"~zod", "~bus", "~nec"}
	for i := range expected {
		if dispatched[i] != expected[i] {
			t.Fatalf("dispatch order mismatch at index %d: expected %q, got %q", i, expected[i], dispatched[i])
		}
	}

	if len(dispatched) != len(expected) {
		t.Fatalf("expected dispatched count %d, got %d", len(expected), len(dispatched))
	}
}
