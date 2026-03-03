package lifecycle

import (
	"errors"
	"strings"
	"testing"
)

func resetExitState() {
	executeClickCommandForExit = executeClickCommand
}

func TestBarExitCreateFailureClearsCode(t *testing.T) {
	t.Cleanup(resetExitState)
	executeClickCommandForExit = func(_, _, _, _, _, _ string, _ func(string)) (string, error) {
		return "", errors.New("write failed")
	}
	if err := BarExit("~zod"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestBarExitExecFailure(t *testing.T) {
	t.Cleanup(resetExitState)
	executeClickCommandForExit = func(_, _, _, _, _, _ string, _ func(string)) (string, error) {
		return "", errors.New("exec failed")
	}

	err := BarExit("~bus")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "failed to execute") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBarExitPokeFailure(t *testing.T) {
	t.Cleanup(resetExitState)
	executeClickCommandForExit = func(_, _, _, _, _, _ string, _ func(string)) (string, error) {
		return "", errors.New("poke failed")
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
	executeClickCommandForExit = func(_, _, _, _, _, _ string, _ func(string)) (string, error) {
		return "", nil
	}

	if err := BarExit("~pal"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func executeClickCommand(_, _, _, _, _, _ string, _ func(string)) (string, error) {
	return "ok", nil
}
