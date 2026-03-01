package lifecycle

import (
	"errors"
	"strings"
	"testing"
)

func resetExitState() {
	createHoonForExit = createHoon
	deleteHoonForExit = deleteHoon
	clickExecForExit = clickExec
	filterResponseForExit = filterResponse
}

func TestBarExitCreateFailureClearsCode(t *testing.T) {
	t.Cleanup(resetExitState)
	createHoonForExit = func(_, _, _ string) error {
		return errors.New("write failed")
	}
	if err := BarExit("~zod"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestBarExitExecFailure(t *testing.T) {
	t.Cleanup(resetExitState)
	createHoonForExit = func(_, _, _ string) error { return nil }
	deleted := false
	deleteHoonForExit = func(_, _ string) { deleted = true }
	clickExecForExit = func(_, _, _ string) (string, error) {
		return "", errors.New("exec failed")
	}

	err := BarExit("~bus")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !deleted {
		t.Fatalf("expected deferred delete")
	}
	if !strings.Contains(err.Error(), "failed to get exec") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBarExitPokeFailure(t *testing.T) {
	t.Cleanup(resetExitState)
	createHoonForExit = func(_, _, _ string) error { return nil }
	deleteHoonForExit = func(_, _ string) {}
	clickExecForExit = func(_, _, _ string) (string, error) { return "response", nil }
	filterResponseForExit = func(_, _ string) (string, bool, error) {
		return "", false, nil
	}

	err := BarExit("~nec")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "poke failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBarExitSuccess(t *testing.T) {
	t.Cleanup(resetExitState)
	createHoonForExit = func(_, _, _ string) error { return nil }
	deleteHoonForExit = func(_, _ string) {}
	clickExecForExit = func(_, _, _ string) (string, error) { return "response", nil }
	filterResponseForExit = func(_, _ string) (string, bool, error) {
		return "", true, nil
	}

	if err := BarExit("~pal"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func createHoon(_, _, _ string) error { return nil }
func deleteHoon(_, _ string)          {}
func clickExec(_, _, _ string) (string, error) {
	return "ok", nil
}
func filterResponse(_, _ string) (string, bool, error) {
	return "", true, nil
}
