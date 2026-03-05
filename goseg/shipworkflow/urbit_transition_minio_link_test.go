package shipworkflow

import (
	"testing"

	"groundseg/structs"
)

func TestLoadMinIOLinkStateDefaultsEndpointFromWgURL(t *testing.T) {
	withRuntimeUrbitConfig(t, structs.UrbitDocker{
		UrbitNetworkConfig: structs.UrbitNetworkConfig{
			WgURL: "example.groundseg.net",
		},
		UrbitFeatureConfig: structs.UrbitFeatureConfig{
			MinIOLinked: true,
		},
	})

	state, err := loadMinIOLinkState("~zod")
	if err != nil {
		t.Fatalf("loadMinIOLinkState returned error: %v", err)
	}
	if !state.IsLinked {
		t.Fatal("expected linked flag to be preserved")
	}
	if state.Endpoint != "s3.example.groundseg.net" {
		t.Fatalf("unexpected endpoint: got %q", state.Endpoint)
	}
}

func TestMinIOLinkCoordinatorStepSequence(t *testing.T) {
	steps := buildToggleMinIOLinkSteps("~zod", structs.WsUrbitPayload{})
	if len(steps) != 5 {
		t.Fatalf("expected 5 steps, got %d", len(steps))
	}
	if steps[0].Run == nil {
		t.Fatal("expected load-current-state step run function")
	}
	if steps[1].Event != "unlinking" || steps[1].EmitWhen == nil {
		t.Fatal("expected unlinking step with EmitWhen")
	}
	if steps[3].Event != "linking" || steps[3].EmitWhen == nil {
		t.Fatal("expected linking step with EmitWhen")
	}
	if steps[4].Event != "success" || steps[4].EmitWhen == nil {
		t.Fatal("expected success step with EmitWhen")
	}
}
