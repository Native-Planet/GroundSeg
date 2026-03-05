package shipworkflow

import (
	"strings"
	"testing"

	"groundseg/structs"
)

func TestSchedulePackRejectsNonPositiveFrequency(t *testing.T) {
	err := schedulePack("~zod", structs.WsUrbitPayload{
		Payload: structs.WsUrbitAction{
			Frequency:    0,
			IntervalType: "day",
		},
	})
	if err == nil {
		t.Fatal("expected non-positive frequency validation error")
	}
	if !strings.Contains(err.Error(), "greater than zero") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSchedulePackRejectsUnknownInterval(t *testing.T) {
	err := schedulePack("~zod", structs.WsUrbitPayload{
		Payload: structs.WsUrbitAction{
			Frequency:    1,
			IntervalType: "year",
		},
	})
	if err == nil {
		t.Fatal("expected unknown interval validation error")
	}
	if !strings.Contains(err.Error(), "unknown pack schedule interval type") {
		t.Fatalf("unexpected error: %v", err)
	}
}
