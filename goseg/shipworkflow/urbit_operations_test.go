package shipworkflow

import (
	"errors"
	"testing"
)

func TestWrapLifecycleErrorPreservesDetailAndPatp(t *testing.T) {
	err := wrapLifecycleError("~zod", "could not start", errors.New("boom"))
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	if err.Error() != "could not start for ~zod: boom" {
		t.Fatalf("unexpected wrap message: %v", err)
	}
}

func TestRunDeskLifecycleRejectsUnsupportedAction(t *testing.T) {
	err := runDeskLifecycle("~zod", deskAction("invalid"), deskLifecycleSpec{desk: "groundseg"})
	if err == nil {
		t.Fatal("expected unsupported action error")
	}
	if err.Error() != "unsupported desk action \"invalid\" for groundseg" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUrbitDeleteStartramServiceRejectsUnknownService(t *testing.T) {
	err := urbitDeleteStartramService("~zod", "unknown")
	if err == nil {
		t.Fatal("expected unknown service error")
	}
}
