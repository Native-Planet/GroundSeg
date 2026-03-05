package shipworkflow

import (
	"errors"
	"strings"
	"testing"

	"groundseg/config"
	"groundseg/structs"
)

func TestBuildStartramRestartTransitionStepsWhenDisabledReturnsGateOnly(t *testing.T) {
	t.Parallel()

	steps := buildStartramRestartTransitionSteps(defaultStartramRuntime(), config.StartramSettings{WgOn: false})
	if len(steps) != 1 {
		t.Fatalf("expected one gate step when wireguard disabled, got %d", len(steps))
	}
	if err := steps[0].Run(); err == nil || !strings.Contains(err.Error(), "startram is disabled") {
		t.Fatalf("expected disabled gate error, got %v", err)
	}
}

func TestBuildStartramWireguardRevertStepsDispatchesForWireguardShips(t *testing.T) {
	t.Parallel()

	var dispatched []string
	runtime := defaultStartramRuntime()
	runtime.UrbitConfFn = func(string) structs.UrbitDocker {
		return structs.UrbitDocker{
			UrbitNetworkConfig: structs.UrbitNetworkConfig{
				Network: "wireguard",
			},
		}
	}
	runtime.DispatchUrbitPayloadFn = func(payload structs.WsUrbitPayload) error {
		dispatched = append(dispatched, payload.Payload.Patp)
		return nil
	}

	step := buildStartramWireguardRevertSteps(runtime, "~zod")
	if step == nil {
		t.Fatal("expected wireguard revert step for wireguard-connected ship")
	}
	if err := step.Run(); err != nil {
		t.Fatalf("wireguard revert step returned error: %v", err)
	}
	if len(dispatched) != 1 || dispatched[0] != "~zod" {
		t.Fatalf("unexpected dispatched payloads: %+v", dispatched)
	}
}

func TestBuildStartramWireguardRevertStepsSkipsNonWireguardShips(t *testing.T) {
	t.Parallel()

	runtime := defaultStartramRuntime()
	runtime.UrbitConfFn = func(string) structs.UrbitDocker {
		return structs.UrbitDocker{
			UrbitNetworkConfig: structs.UrbitNetworkConfig{
				Network: "bridge",
			},
		}
	}

	if step := buildStartramWireguardRevertSteps(runtime, "~bus"); step != nil {
		t.Fatalf("expected no revert step for non-wireguard network, got %+v", step)
	}
}

func TestBuildStartramToggleWireguardDisableStepsRunsCoreActions(t *testing.T) {
	t.Parallel()

	var updatedConfig bool
	var stoppedContainer bool
	var dispatched []string

	runtime := defaultStartramRuntime()
	runtime.UpdateConfigTypedFn = func(...config.ConfigUpdateOption) error {
		updatedConfig = true
		return nil
	}
	runtime.StopContainerByNameFn = func(name string) error {
		if name != "wireguard" {
			t.Fatalf("unexpected container stop target: %s", name)
		}
		stoppedContainer = true
		return nil
	}
	runtime.UrbitConfFn = func(patp string) structs.UrbitDocker {
		network := "bridge"
		if patp == "~zod" {
			network = "wireguard"
		}
		return structs.UrbitDocker{
			UrbitNetworkConfig: structs.UrbitNetworkConfig{
				Network: network,
			},
		}
	}
	runtime.DispatchUrbitPayloadFn = func(payload structs.WsUrbitPayload) error {
		dispatched = append(dispatched, payload.Payload.Patp)
		return nil
	}

	steps, err := buildStartramToggleWireguardDisableSteps(runtime, config.StartramSettings{
		WgOn:  true,
		Piers: []string{"~zod", "~bus"},
	})
	if err != nil {
		t.Fatalf("buildStartramToggleWireguardDisableSteps returned error: %v", err)
	}
	for _, step := range steps {
		if err := step.Run(); err != nil {
			t.Fatalf("disable step %q returned error: %v", step.Name, err)
		}
	}

	if !updatedConfig {
		t.Fatal("expected wireguard disable config update")
	}
	if !stoppedContainer {
		t.Fatal("expected wireguard container stop")
	}
	if len(dispatched) != 1 || dispatched[0] != "~zod" {
		t.Fatalf("expected only wireguard ship dispatch, got %+v", dispatched)
	}
}

func TestStartramEndpointTransitionCoordinatorHelpers(t *testing.T) {
	t.Parallel()

	var stopCalls int
	var svcDeletes []string
	var cycled bool
	var persistedEndpoint string
	var persistedRegistered *bool

	runtime := defaultStartramRuntime()
	runtime.StopContainerByNameFn = func(name string) error {
		if name != "wireguard" {
			t.Fatalf("unexpected stop target: %s", name)
		}
		stopCalls++
		return nil
	}
	runtime.SvcDeleteFn = func(url, svcType string) error {
		svcDeletes = append(svcDeletes, url+"|"+svcType)
		return nil
	}
	runtime.CycleWgKeyFn = func() error {
		cycled = true
		return nil
	}
	runtime.UpdateConfigTypedFn = func(opts ...config.ConfigUpdateOption) error {
		patch := &config.ConfPatch{}
		for _, opt := range opts {
			opt(patch)
		}
		if patch.EndpointURL != nil {
			persistedEndpoint = *patch.EndpointURL
		}
		persistedRegistered = patch.WgRegistered
		return nil
	}

	coordinator := startramEndpointTransitionCoordinator{
		runtime: runtime,
		settings: config.StartramSettings{
			WgOn:         true,
			WgRegistered: true,
			Piers:        []string{"~zod"},
		},
		endpoint: "new.endpoint.example",
	}

	if err := coordinator.stopWireguard(); err != nil {
		t.Fatalf("stopWireguard returned error: %v", err)
	}
	if err := coordinator.unregisterAnchors(); err != nil {
		t.Fatalf("unregisterAnchors returned error: %v", err)
	}
	if err := coordinator.cycleWireguardKey(); err != nil {
		t.Fatalf("cycleWireguardKey returned error: %v", err)
	}
	if err := coordinator.persistNewEndpoint(); err != nil {
		t.Fatalf("persistNewEndpoint returned error: %v", err)
	}

	if stopCalls != 1 {
		t.Fatalf("expected one wireguard stop call, got %d", stopCalls)
	}
	if len(svcDeletes) != 2 {
		t.Fatalf("expected urbit+s3 service deletions, got %+v", svcDeletes)
	}
	if !cycled {
		t.Fatal("expected wireguard key cycling")
	}
	if persistedEndpoint != "new.endpoint.example" {
		t.Fatalf("unexpected persisted endpoint: %q", persistedEndpoint)
	}
	if persistedRegistered == nil || *persistedRegistered {
		t.Fatalf("expected WgRegistered patch to be false, got %+v", persistedRegistered)
	}
}

func TestStartramEndpointTransitionCoordinatorAggregatesDeleteErrors(t *testing.T) {
	t.Parallel()

	runtime := defaultStartramRuntime()
	runtime.SvcDeleteFn = func(string, string) error { return errors.New("delete failed") }
	coordinator := startramEndpointTransitionCoordinator{
		runtime: runtime,
		settings: config.StartramSettings{
			WgRegistered: true,
			Piers:        []string{"~zod"},
		},
	}

	err := coordinator.unregisterAnchors()
	if err == nil {
		t.Fatal("expected aggregate error when service delete fails")
	}
}
