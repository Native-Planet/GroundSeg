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
	parseClickResponseForDesk = parseResponseForDesk
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

	commandCalled := false
	executeClickCommandForDesk = func(_, _, _, _, _, _ string, _ func(string)) (string, error) {
		commandCalled = true
		return "", nil
	}

	status, err := getDesk("~nec", "garden", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "running" {
		t.Fatalf("unexpected status: %s", status)
	}
	if commandCalled {
		t.Fatalf("expected cached read without command execution")
	}
}

func TestGetDeskBypassFetchesFreshStatus(t *testing.T) {
	t.Cleanup(resetDeskState)
	executeClickCommandForDesk = func(_, _, _, _, _, _ string, _ func(string)) (string, error) { return "response", nil }
	parseClickResponseForDesk = func(
		patp, file, hoon, sourcePath, responseToken, operation string,
		clearLusCode func(string),
	) (string, string, bool, error) {
		return "", "mounted", true, nil
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
	executeClickCommandForDesk = func(_, _, _, _, _, _ string, _ func(string)) (string, error) {
		return "", errors.New("exec failed")
	}
	parseClickResponseForDesk = func(
		_, _, _, _, _, _ string, _ func(string),
	) (string, string, bool, error) {
		return "", "", false, errors.New("exec failed")
	}

	_, err := getDesk("~zod", "base", true)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "exec failed") {
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
	executeClickCommandForDesk = func(_, _, _, _, _, _ string, _ func(string)) (string, error) {
		return "", errors.New("failed poke")
	}
	parseClickResponseForDesk = parseResponseForDeskError

	if err := mountDesk("~zod", "base"); err == nil || !strings.Contains(err.Error(), "failed poke") {
		t.Fatalf("expected mount poke failure, got: %v", err)
	}
	if err := commitDesk("~zod", "base"); err == nil || !strings.Contains(err.Error(), "failed poke") {
		t.Fatalf("expected commit poke failure, got: %v", err)
	}
}

func TestDeskActionCommandsBubbleErrors(t *testing.T) {
	t.Cleanup(resetDeskState)
	executeClickCommandForDesk = func(_, _, _, _, _, _ string, _ func(string)) (string, error) {
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

func executeCommandForDesk(_, _, _, _, _, _ string, _ func(string)) (string, error) {
	return "ok", nil
}
func parseResponseForDesk(_, _, _, _, _, _ string, _ func(string)) (string, string, bool, error) {
	return "", "", true, nil
}
func parseResponseForDeskError(
	_, _, _, _, _, _ string,
	_ func(string),
) (string, string, bool, error) {
	return "", "", false, nil
}
