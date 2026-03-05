package shipworkflow

import (
	"testing"

	"groundseg/structs"
)

func TestBuildUrbitDomainStepsReturnsSingleSuccessStep(t *testing.T) {
	steps := buildUrbitDomainSteps("~zod", structs.WsUrbitPayload{
		Payload: structs.WsUrbitAction{Domain: "ship.example.com"},
	})
	if len(steps) != 1 {
		t.Fatalf("expected one transition step, got %d", len(steps))
	}
	if steps[0].Event != "success" {
		t.Fatalf("expected success event, got %q", steps[0].Event)
	}
	if steps[0].Run == nil {
		t.Fatal("expected transition step run function")
	}
}

func TestBuildMinIODomainStepsReturnsSingleSuccessStep(t *testing.T) {
	steps := buildMinIODomainSteps("~zod", structs.WsUrbitPayload{
		Payload: structs.WsUrbitAction{Domain: "s3.ship.example.com"},
	})
	if len(steps) != 1 {
		t.Fatalf("expected one transition step, got %d", len(steps))
	}
	if steps[0].Event != "success" {
		t.Fatalf("expected success event, got %q", steps[0].Event)
	}
	if steps[0].Run == nil {
		t.Fatal("expected transition step run function")
	}
}
