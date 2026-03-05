package apt

import (
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"groundseg/structs"
)

func resetAptSeamsForTest(t *testing.T) {
	t.Helper()
	origExec := execCommandForAPT
	origHasUpdates := hasUpdatesForAPT
	origUpdateCheck := updateCheckForRunUpgrade
	origSystemUpdates := systemUpdates
	t.Cleanup(func() {
		execCommandForAPT = origExec
		hasUpdatesForAPT = origHasUpdates
		updateCheckForRunUpgrade = origUpdateCheck
		systemUpdates = origSystemUpdates
	})
}

func TestHasUpdatesParsesUpgradeSummary(t *testing.T) {
	resetAptSeamsForTest(t)

	execCommandForAPT = func(name string, args ...string) *exec.Cmd {
		if name != "apt" {
			t.Fatalf("unexpected command name: %s", name)
		}
		switch strings.Join(args, " ") {
		case "update":
			return exec.Command("true")
		case "upgrade -s":
			return exec.Command("sh", "-c", "printf '5 upgraded, 2 newly installed, 1 to remove and 3 not upgraded.\\n'")
		default:
			t.Fatalf("unexpected command args: %v", args)
			return nil
		}
	}

	updates, err := hasUpdates()
	if err != nil {
		t.Fatalf("hasUpdates returned error: %v", err)
	}
	if updates.Linux.Upgrade != 5 || updates.Linux.New != 2 || updates.Linux.Remove != 1 || updates.Linux.Ignore != 3 {
		t.Fatalf("unexpected parsed updates: %+v", updates.Linux)
	}
}

func TestHasUpdatesReturnsErrorsForCommandAndPatternFailures(t *testing.T) {
	resetAptSeamsForTest(t)

	execCommandForAPT = func(name string, args ...string) *exec.Cmd {
		switch strings.Join(args, " ") {
		case "update":
			return exec.Command("false")
		default:
			return exec.Command("true")
		}
	}
	if _, err := hasUpdates(); err == nil {
		t.Fatal("expected hasUpdates to fail when apt update fails")
	}

	execCommandForAPT = func(name string, args ...string) *exec.Cmd {
		switch strings.Join(args, " ") {
		case "update":
			return exec.Command("true")
		case "upgrade -s":
			return exec.Command("sh", "-c", "printf 'no updates available\\n'")
		default:
			return exec.Command("true")
		}
	}
	if _, err := hasUpdates(); err == nil || !strings.Contains(err.Error(), "Pattern not found") {
		t.Fatalf("expected pattern not found error, got %v", err)
	}
}

func TestRunUpgradeInvokesUpdateCheckEvenOnError(t *testing.T) {
	resetAptSeamsForTest(t)

	updateCalls := 0
	updateCheckForRunUpgrade = func() { updateCalls++ }

	execCommandForAPT = func(name string, args ...string) *exec.Cmd {
		if strings.Join(args, " ") != "upgrade -y" {
			t.Fatalf("unexpected command args: %v", args)
		}
		return exec.Command("false")
	}
	if err := RunUpgrade(); err == nil {
		t.Fatal("expected RunUpgrade to return command error")
	}
	if updateCalls != 1 {
		t.Fatalf("expected update check to run once after failed upgrade, got %d", updateCalls)
	}

	execCommandForAPT = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}
	if err := RunUpgrade(); err != nil {
		t.Fatalf("expected RunUpgrade success, got %v", err)
	}
	if updateCalls != 2 {
		t.Fatalf("expected update check to run after successful upgrade, got %d", updateCalls)
	}
}

func TestUpdateCheckUpdatesGlobalStateOnlyOnSuccess(t *testing.T) {
	resetAptSeamsForTest(t)

	success := structs.SystemUpdates{}
	success.Linux.Upgrade = 7
	hasUpdatesForAPT = func() (structs.SystemUpdates, error) {
		return success, nil
	}
	UpdateCheck()
	if !reflect.DeepEqual(systemUpdates, success) {
		t.Fatalf("expected updates to be overwritten on success: got %+v want %+v", systemUpdates, success)
	}

	existing := systemUpdates
	hasUpdatesForAPT = func() (structs.SystemUpdates, error) {
		return structs.SystemUpdates{}, errors.New("lookup failed")
	}
	UpdateCheck()
	if !reflect.DeepEqual(systemUpdates, existing) {
		t.Fatalf("expected updates to remain unchanged on error: got %+v want %+v", systemUpdates, existing)
	}
}
