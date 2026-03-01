package desk

import (
	"errors"
	"strings"
	"testing"
	"time"

	"groundseg/structs"
)

func resetDeskState() {
	desksMutex.Lock()
	shipDesks = make(map[string]map[string]structs.ClickDesks)
	desksMutex.Unlock()
	executeClickCommandForDesk = executeCommandForDesk
	createHoonForDesk = createHoon
	deleteHoonForDesk = deleteHoon
	clickExecForDesk = clickExec
	filterResponseForDesk = filterResponse
}

func TestAllowDeskRequestFlowControl(t *testing.T) {
	t.Cleanup(resetDeskState)
	if !allowDeskRequest("~zod", "base") {
		t.Fatalf("expected unknown desk request to be allowed")
	}

	desksMutex.Lock()
	shipDesks["~zod"] = map[string]structs.ClickDesks{
		"base": {LastError: time.Now()},
	}
	desksMutex.Unlock()
	if allowDeskRequest("~zod", "base") {
		t.Fatalf("expected recent error to deny request")
	}

	desksMutex.Lock()
	shipDesks["~zod"] = map[string]structs.ClickDesks{
		"base": {LastFetch: time.Now().Add(-3 * time.Minute), Status: "ok"},
	}
	desksMutex.Unlock()
	if !allowDeskRequest("~zod", "base") {
		t.Fatalf("expected stale cache to allow request")
	}

	desksMutex.Lock()
	shipDesks["~zod"] = map[string]structs.ClickDesks{
		"base": {LastFetch: time.Now(), Status: "ok"},
	}
	desksMutex.Unlock()
	if allowDeskRequest("~zod", "base") {
		t.Fatalf("expected fresh cache to deny request")
	}
}

func TestFetchDeskFromMemoryErrors(t *testing.T) {
	t.Cleanup(resetDeskState)
	if _, err := fetchDeskFromMemory("~zod", "base"); err == nil {
		t.Fatalf("expected missing patp error")
	}

	desksMutex.Lock()
	shipDesks["~zod"] = map[string]structs.ClickDesks{}
	desksMutex.Unlock()
	if _, err := fetchDeskFromMemory("~zod", "base"); err == nil {
		t.Fatalf("expected missing desk error")
	}
}

func TestGetDeskUsesCachedStatus(t *testing.T) {
	t.Cleanup(resetDeskState)

	desksMutex.Lock()
	shipDesks["~nec"] = map[string]structs.ClickDesks{
		"garden": {LastFetch: time.Now(), Status: "running"},
	}
	desksMutex.Unlock()

	createCalled := false
	createHoonForDesk = func(_, _, _ string) error {
		createCalled = true
		return nil
	}

	status, err := getDesk("~nec", "garden", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "running" {
		t.Fatalf("unexpected status: %s", status)
	}
	if createCalled {
		t.Fatalf("expected cached read without command execution")
	}
}

func TestGetDeskBypassFetchesFreshStatus(t *testing.T) {
	t.Cleanup(resetDeskState)
	createHoonForDesk = func(_, _, _ string) error { return nil }
	deleteHoonForDesk = func(_, _ string) {}
	clickExecForDesk = func(_, _, _ string) (string, error) { return "response", nil }
	filterResponseForDesk = func(_, _ string) (string, bool, error) {
		return "mounted", false, nil
	}

	got, err := getDesk("~bus", "base", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "mounted" {
		t.Fatalf("unexpected status: %s", got)
	}
}

func TestGetDeskExecFailureStoresError(t *testing.T) {
	t.Cleanup(resetDeskState)
	createHoonForDesk = func(_, _, _ string) error { return nil }
	deleted := false
	deleteHoonForDesk = func(_, _ string) { deleted = true }
	clickExecForDesk = func(_, _, _ string) (string, error) {
		return "", errors.New("exec failed")
	}

	_, err := getDesk("~zod", "base", true)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !deleted {
		t.Fatalf("expected deferred delete to run")
	}
	if !strings.Contains(err.Error(), "failed to get exec") {
		t.Fatalf("unexpected error: %v", err)
	}

	desksMutex.Lock()
	record := shipDesks["~zod"]["base"]
	desksMutex.Unlock()
	if record.LastError.IsZero() {
		t.Fatalf("expected LastError to be recorded")
	}
}

func TestMountDeskAndCommitDeskPokeFailures(t *testing.T) {
	t.Cleanup(resetDeskState)
	executeClickCommandForDesk = func(_, _, _, _, _, _ string) (string, error) {
		return "response", nil
	}
	filterResponseForDesk = func(_, _ string) (string, bool, error) {
		return "", false, nil
	}

	if err := mountDesk("~zod", "base"); err == nil || !strings.Contains(err.Error(), "poke failed") {
		t.Fatalf("expected mount poke failure, got: %v", err)
	}
	if err := commitDesk("~zod", "base"); err == nil || !strings.Contains(err.Error(), "poke failed") {
		t.Fatalf("expected commit poke failure, got: %v", err)
	}
}

func TestDeskActionCommandsBubbleErrors(t *testing.T) {
	t.Cleanup(resetDeskState)
	executeClickCommandForDesk = func(_, _, _, _, _, _ string) (string, error) {
		return "", errors.New("command failed")
	}

	if err := ReviveDesk("~zod", "base"); err == nil {
		t.Fatalf("expected reviveDesk to fail")
	}
	if err := UninstallDesk("~zod", "base"); err == nil {
		t.Fatalf("expected uninstallDesk to fail")
	}
	if err := InstallDesk("~zod", "~bus", "base"); err == nil {
		t.Fatalf("expected installDesk to fail")
	}
}

func executeCommandForDesk(_, _, _, _, _, _ string) (string, error) {
	return "ok", nil
}

func createHoon(_, _, _ string) error { return nil }
func deleteHoon(_, _ string)          {}
func clickExec(_, _, _ string) (string, error) {
	return "ok", nil
}
func filterResponse(_, _ string) (string, bool, error) { return "", true, nil }
