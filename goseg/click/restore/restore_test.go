package restore

import (
	"errors"
	"strings"
	"testing"
)

func resetRestoreSeams() {
	executeClickCommandForRestore = executeRestoreCommand
	restoreAgentFn = restoreAgent
}

func TestRestoreAgentExecFailure(t *testing.T) {
	t.Cleanup(resetRestoreSeams)
	executeClickCommandForRestore = func(_, _, _, _, _, _ string, _ func(string)) (string, error) {
		return "", errors.New("exec failed")
	}

	err := RestoreAgent("~zod", "groups")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "exec failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRestoreAgentFilterFailure(t *testing.T) {
	t.Cleanup(resetRestoreSeams)
	executeClickCommandForRestore = func(_, _, _, _, _, _ string, _ func(string)) (string, error) {
		return "", errors.New("parse failed")
	}

	err := RestoreAgent("~zod", "groups")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "parse failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRestoreAgentPokeFailure(t *testing.T) {
	t.Cleanup(resetRestoreSeams)
	executeClickCommandForRestore = func(_, _, _, _, _, _ string, _ func(string)) (string, error) {
		return "", errors.New("failed poke")
	}

	err := RestoreAgent("~zod", "groups")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "failed poke") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRestoreAgentBuildsExpectedCommand(t *testing.T) {
	t.Cleanup(resetRestoreSeams)

	var gotPatp, gotFile, gotHoon, gotOperation string
	executeClickCommandForRestore = func(
		patp, file, hoon, sourcePath, successToken, operation string,
		_clearLus func(string),
	) (string, error) {
		gotPatp = patp
		gotFile = file
		gotHoon = hoon
		gotOperation = operation
		return "ok", nil
	}

	if err := RestoreAgent("~nec", "profile"); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if gotPatp != "~nec" {
		t.Fatalf("unexpected patp: %s", gotPatp)
	}
	if gotFile != "restore-profile" {
		t.Fatalf("unexpected file: %s", gotFile)
	}
	if gotOperation != "Click restore-profile" {
		t.Fatalf("unexpected operation: %s", gotOperation)
	}
	if !strings.Contains(gotHoon, "backup-profile/jam") {
		t.Fatalf("expected backup scry path in hoon: %s", gotHoon)
	}
	if !strings.Contains(gotHoon, "send-raw-card") {
		t.Fatalf("expected send-raw-card in hoon: %s", gotHoon)
	}
}

func TestRestoreTlonAggregatesErrors(t *testing.T) {
	t.Cleanup(resetRestoreSeams)
	restoreAgentFn = func(_ string, agent string) error {
		if agent == "channels" || agent == "profile" {
			return errors.New("boom")
		}
		return nil
	}

	err := RestoreTlon("~zod")
	if err == nil {
		t.Fatalf("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "channels: boom") || !strings.Contains(msg, "profile: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func executeRestoreCommand(_, _, _, _, _, _ string, _ func(string)) (string, error) {
	return "ok", nil
}
