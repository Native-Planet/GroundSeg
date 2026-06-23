package docker

import "testing"

func TestPlanPowerTransitionStartsFromStaleMaintenanceStatus(t *testing.T) {
	transition, err := PlanPowerTransition("pack", false)
	if err != nil {
		t.Fatalf("PlanPowerTransition returned error: %v", err)
	}
	if !transition.Start {
		t.Fatal("expected stale stopped maintenance status to start normally")
	}
	if !transition.UpdateBootStatus || transition.BootStatus != "boot" {
		t.Fatalf("expected boot status recovery to boot, got update=%v status=%q", transition.UpdateBootStatus, transition.BootStatus)
	}
	if transition.DesiredStatus != "running" {
		t.Fatalf("expected desired status running, got %q", transition.DesiredStatus)
	}
}

func TestPlanPowerTransitionRejectsRunningMaintenanceStatus(t *testing.T) {
	if _, err := PlanPowerTransition("pack", true); err == nil {
		t.Fatal("expected running maintenance status to reject power toggle")
	}
}

func TestPlanPowerTransitionAllowsIgnoreToStartAndStop(t *testing.T) {
	startTransition, err := PlanPowerTransition("ignore", false)
	if err != nil {
		t.Fatalf("PlanPowerTransition start returned error: %v", err)
	}
	if !startTransition.Start || startTransition.UpdateBootStatus {
		t.Fatalf("expected ignore start without boot status update, got %+v", startTransition)
	}
	if startTransition.DesiredStatus != "running" {
		t.Fatalf("expected desired status running, got %q", startTransition.DesiredStatus)
	}

	stopTransition, err := PlanPowerTransition("ignore", true)
	if err != nil {
		t.Fatalf("PlanPowerTransition stop returned error: %v", err)
	}
	if !stopTransition.Stop || stopTransition.UpdateBootStatus {
		t.Fatalf("expected ignore stop without boot status update, got %+v", stopTransition)
	}
	if stopTransition.DesiredStatus != "stopped" {
		t.Fatalf("expected desired status stopped, got %q", stopTransition.DesiredStatus)
	}
}

func TestPlanPowerTransitionRememberedBootStates(t *testing.T) {
	startTransition, err := PlanPowerTransition("noboot", false)
	if err != nil {
		t.Fatalf("PlanPowerTransition start returned error: %v", err)
	}
	if !startTransition.Start || !startTransition.UpdateBootStatus || startTransition.BootStatus != "boot" {
		t.Fatalf("expected noboot to boot start transition, got %+v", startTransition)
	}

	stopTransition, err := PlanPowerTransition("boot", true)
	if err != nil {
		t.Fatalf("PlanPowerTransition stop returned error: %v", err)
	}
	if !stopTransition.Stop || !stopTransition.UpdateBootStatus || stopTransition.BootStatus != "noboot" {
		t.Fatalf("expected boot to noboot stop transition, got %+v", stopTransition)
	}
}
