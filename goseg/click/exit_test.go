package click

import (
	"errors"
	"strings"
	"testing"

	"groundseg/structs"
)

func resetExitState() {
	createHoonForExit = createHoon
	deleteHoonForExit = deleteHoon
	clickExecForExit = clickExec
	filterResponseForExit = filterResponse
	codeMutex.Lock()
	lusCodes = make(map[string]structs.ClickLusCode)
	codeMutex.Unlock()
}

func TestBarExitCreateFailureClearsCode(t *testing.T) {
	t.Cleanup(resetExitState)
	codeMutex.Lock()
	lusCodes["~zod"] = structs.ClickLusCode{LusCode: strings.Repeat("a", 27)}
	codeMutex.Unlock()
	createHoonForExit = func(_, _, _ string) error {
		return errors.New("write failed")
	}

	err := barExit("~zod")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "failed to create hoon") {
		t.Fatalf("unexpected error: %v", err)
	}
	codeMutex.Lock()
	_, exists := lusCodes["~zod"]
	codeMutex.Unlock()
	if exists {
		t.Fatalf("expected +code cache entry to be cleared")
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

	err := barExit("~bus")
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

	err := barExit("~nec")
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

	if err := barExit("~pal"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
