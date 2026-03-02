package shipworkflow

import (
	"testing"
	"time"

	"groundseg/config"
	"groundseg/structs"
	"groundseg/transition"
)

func withRuntimeUrbitConfig(t *testing.T, conf structs.UrbitDocker) {
	t.Helper()
	previous := getUrbitConfigFn
	getUrbitConfigFn = func(_ string) structs.UrbitDocker {
		return conf
	}
	t.Cleanup(func() {
		getUrbitConfigFn = previous
	})
}

func withTransitionTemplateStub(t *testing.T, fn func(string, urbitTransitionTemplate, ...transitionStep[string]) error) {
	t.Helper()
	previous := runUrbitTransitionTemplateFn
	runUrbitTransitionTemplateFn = fn
	t.Cleanup(func() {
		runUrbitTransitionTemplateFn = previous
	})
}

func withTransitionedOperationStub(t *testing.T, fn func(string, string, string, string, time.Duration, func() error) error) {
	t.Helper()
	previous := runTransitionedOperationFn
	runTransitionedOperationFn = func(patp, transitionType, startEvent, successEvent string, clearDelay time.Duration, op func() error) error {
		return fn(patp, transitionType, startEvent, successEvent, clearDelay, op)
	}
	t.Cleanup(func() {
		runTransitionedOperationFn = previous
	})
}

func withLoadUrbitConfigStub(t *testing.T, fn func(string) error) {
	t.Helper()
	previous := loadUrbitConfigFn
	loadUrbitConfigFn = fn
	t.Cleanup(func() {
		loadUrbitConfigFn = previous
	})
}

func TestSetUrbitDomainUsesUrbitDomainTransition(t *testing.T) {
	patp := "~zod"
	var called bool
	withRuntimeUrbitConfig(t, structs.UrbitDocker{
		UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
			PierName: patp,
		},
		UrbitNetworkConfig: structs.UrbitNetworkConfig{
			WgURL: "groundseg.net",
		},
	})
	withTransitionTemplateStub(t, func(gotPatp string, template urbitTransitionTemplate, steps ...transitionStep[string]) error {
		called = true
		if gotPatp != patp {
			t.Fatalf("expected patp %q, got %q", patp, gotPatp)
		}
		if template.transitionType != string(transition.UrbitTransitionUrbitDomain) {
			t.Fatalf("unexpected transition type %q", template.transitionType)
		}
		if got, want := len(steps), 1; got != want {
			t.Fatalf("expected %d transition step, got %d", want, got)
		}
		return nil
	})

	err := SetUrbitDomain(patp, structs.WsUrbitPayload{Payload: structs.WsUrbitAction{Domain: "ship.groundseg.net"}})
	if !called {
		t.Fatal("expected transition template to be invoked")
	}
	if err != nil {
		t.Fatalf("SetUrbitDomain returned error: %v", err)
	}
}

func TestSetMinIODomainUsesMinIODomainTransition(t *testing.T) {
	patp := "~zod"
	var called bool
	withRuntimeUrbitConfig(t, structs.UrbitDocker{
		UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
			PierName: patp,
		},
		UrbitNetworkConfig: structs.UrbitNetworkConfig{
			WgURL: "groundseg.net",
		},
	})
	withTransitionTemplateStub(t, func(gotPatp string, template urbitTransitionTemplate, steps ...transitionStep[string]) error {
		called = true
		if gotPatp != patp {
			t.Fatalf("expected patp %q, got %q", patp, gotPatp)
		}
		if template.transitionType != string(transition.UrbitTransitionMinIODomain) {
			t.Fatalf("unexpected transition type %q", template.transitionType)
		}
		if len(steps) != 1 {
			t.Fatalf("expected 1 transition step, got %d", len(steps))
		}
		return nil
	})
	err := SetMinIODomain(patp, structs.WsUrbitPayload{Payload: structs.WsUrbitAction{Domain: "s3.example.groundseg.net"}})
	if !called {
		t.Fatal("expected transition template to be invoked")
	}
	if err != nil {
		t.Fatalf("SetMinIODomain returned error: %v", err)
	}
}

func TestTogglePowerUsesTogglePowerTransition(t *testing.T) {
	patp := "~zod"
	withRuntimeUrbitConfig(t, structs.UrbitDocker{
		UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
			PierName:   patp,
			BootStatus: "boot",
		},
	})
	var called bool
	withTransitionTemplateStub(t, func(gotPatp string, template urbitTransitionTemplate, _ ...transitionStep[string]) error {
		called = true
		if template.transitionType != string(transition.UrbitTransitionTogglePower) {
			t.Fatalf("unexpected transition type %q", template.transitionType)
		}
		return nil
	})
	if err := TogglePower(patp); err != nil {
		t.Fatalf("TogglePower returned error: %v", err)
	}
	if !called {
		t.Fatal("expected transition template to be invoked")
	}
}

func TestToggleDevModeUsesTransitionTemplate(t *testing.T) {
	patp := "~zod"
	withRuntimeUrbitConfig(t, structs.UrbitDocker{
		UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
			PierName:   patp,
			BootStatus: "boot",
		},
	})
	var called bool
	withTransitionTemplateStub(t, func(gotPatp string, template urbitTransitionTemplate, _ ...transitionStep[string]) error {
		called = true
		if template.transitionType != string(transition.UrbitTransitionToggleDevMode) {
			t.Fatalf("unexpected transition type %q", template.transitionType)
		}
		return nil
	})
	if err := ToggleDevMode(patp); err != nil {
		t.Fatalf("ToggleDevMode returned error: %v", err)
	}
	if !called {
		t.Fatal("expected transition template to be invoked")
	}
}

func TestRebuildContainerUsesRunTransitionedOperation(t *testing.T) {
	patp := "~zod"
	withRuntimeUrbitConfig(t, structs.UrbitDocker{
		UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
			PierName:   patp,
			BootStatus: "boot",
		},
	})
	var called bool
	withTransitionTemplateStub(t, func(gotPatp string, template urbitTransitionTemplate, _ ...transitionStep[string]) error {
		called = true
		if gotPatp != patp {
			t.Fatalf("expected patp %q, got %q", patp, gotPatp)
		}
		if template.transitionType != string(transition.UrbitTransitionRebuildContainer) || template.startEvent != "loading" || template.successEvent != "success" {
			t.Fatalf("unexpected transition metadata: %q %q %q", template.transitionType, template.startEvent, template.successEvent)
		}
		if template.clearDelay != 3*time.Second {
			t.Fatalf("unexpected clear delay: %v", template.clearDelay)
		}
		return nil
	})
	if err := RebuildContainer(patp); err != nil {
		t.Fatalf("RebuildContainer returned error: %v", err)
	}
	if !called {
		t.Fatal("expected transition template to be invoked")
	}
}

func TestToggleNetworkUsesTransitionTemplate(t *testing.T) {
	patp := "~zod"
	withRuntimeUrbitConfig(t, structs.UrbitDocker{
		UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
			PierName:   patp,
			BootStatus: "boot",
		},
		UrbitNetworkConfig: structs.UrbitNetworkConfig{
			Network: "wireguard",
		},
	})
	var called bool
	withTransitionTemplateStub(t, func(_ string, template urbitTransitionTemplate, _ ...transitionStep[string]) error {
		called = true
		if template.transitionType != string(transition.UrbitTransitionToggleNetwork) {
			t.Fatalf("unexpected transition type %q", template.transitionType)
		}
		return nil
	})
	if err := ToggleNetwork(patp); err != nil {
		t.Fatalf("ToggleNetwork returned error: %v", err)
	}
	if !called {
		t.Fatal("expected transition template to be invoked")
	}
}

func TestToggleMinIOLinkUsesTransitionTemplate(t *testing.T) {
	patp := "~zod"
	withRuntimeUrbitConfig(t, structs.UrbitDocker{
		UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
			PierName: patp,
		},
		UrbitWebConfig: structs.UrbitWebConfig{
			CustomS3Web: "s3.example.com",
		},
		UrbitNetworkConfig: structs.UrbitNetworkConfig{
			WgURL: "groundseg.net",
		},
	})
	var called bool
	withTransitionTemplateStub(t, func(_ string, template urbitTransitionTemplate, _ ...transitionStep[string]) error {
		called = true
		if template.transitionType != string(transition.UrbitTransitionToggleMinIOLink) {
			t.Fatalf("unexpected transition type %q", template.transitionType)
		}
		return nil
	})
	if err := ToggleMinIOLink(patp); err != nil {
		t.Fatalf("ToggleMinIOLink returned error: %v", err)
	}
	if !called {
		t.Fatal("expected transition template to be invoked")
	}
}

func TestHandleLoomUsesTransitionTemplate(t *testing.T) {
	patp := "~zod"
	withRuntimeUrbitConfig(t, structs.UrbitDocker{
		UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
			PierName: patp,
		},
	})
	var called bool
	withTransitionTemplateStub(t, func(_ string, template urbitTransitionTemplate, _ ...transitionStep[string]) error {
		called = true
		if template.transitionType != string(transition.UrbitTransitionLoom) {
			t.Fatalf("unexpected transition type %q", template.transitionType)
		}
		return nil
	})
	if err := SetLoom(patp, structs.WsUrbitPayload{Payload: structs.WsUrbitAction{Value: 42}}); err != nil {
		t.Fatalf("SetLoom returned error: %v", err)
	}
	if !called {
		t.Fatal("expected transition template to be invoked")
	}
}

func TestHandleSnapTimeUsesTransitionTemplate(t *testing.T) {
	patp := "~zod"
	withRuntimeUrbitConfig(t, structs.UrbitDocker{
		UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
			PierName: patp,
		},
	})
	var called bool
	withTransitionTemplateStub(t, func(_ string, template urbitTransitionTemplate, _ ...transitionStep[string]) error {
		called = true
		if template.transitionType != string(transition.UrbitTransitionSnapTime) {
			t.Fatalf("unexpected transition type %q", template.transitionType)
		}
		return nil
	})
	if err := SetSnapTime(patp, structs.WsUrbitPayload{Payload: structs.WsUrbitAction{Value: 24}}); err != nil {
		t.Fatalf("SetSnapTime returned error: %v", err)
	}
	if !called {
		t.Fatal("expected transition template to be invoked")
	}
}

func TestToggleAutoRebootPersistsDisableShipRestarts(t *testing.T) {
	patp := "~zod"
	setupUrbitOperationsConfig(t, patp, structs.UrbitDocker{
		UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
			PierName: patp,
		},
		UrbitFeatureConfig: structs.UrbitFeatureConfig{
			DisableShipRestarts: false,
		},
	})
	withLoadUrbitConfigStub(t, func(_ string) error { return nil })
	if err := ToggleAutoReboot(patp); err != nil {
		t.Fatalf("ToggleAutoReboot returned error: %v", err)
	}
	conf := config.UrbitConf(patp)
	if got := conf.DisableShipRestarts; got != true {
		t.Fatalf("expected disable ship restarts to flip to true, got %v", got)
	}
}
