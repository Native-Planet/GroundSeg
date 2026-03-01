package click

import (
	"errors"
	"strings"
	"testing"
)

func resetRestoreSeams() {
	executeClickCommandForRestore = executeClickCommand
	filterResponseForRestore = filterResponse
}

func TestRestoreAgentExecFailure(t *testing.T) {
	t.Cleanup(resetRestoreSeams)
	executeClickCommandForRestore = func(_, _, _, _, _, _ string) (string, error) {
		return "", errors.New("exec failed")
	}

	err := restoreAgent("~zod", "groups")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "exec failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRestoreAgentFilterFailure(t *testing.T) {
	t.Cleanup(resetRestoreSeams)
	executeClickCommandForRestore = func(_, _, _, _, _, _ string) (string, error) {
		return "response", nil
	}
	filterResponseForRestore = func(_, _ string) (string, bool, error) {
		return "", false, errors.New("parse failed")
	}

	err := restoreAgent("~zod", "groups")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "failed to get exec") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRestoreAgentPokeFailure(t *testing.T) {
	t.Cleanup(resetRestoreSeams)
	executeClickCommandForRestore = func(_, _, _, _, _, _ string) (string, error) {
		return "response", nil
	}
	filterResponseForRestore = func(_, _ string) (string, bool, error) {
		return "", false, nil
	}

	err := restoreAgent("~zod", "groups")
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
	executeClickCommandForRestore = func(patp, file, hoon, _, _, operation string) (string, error) {
		gotPatp = patp
		gotFile = file
		gotHoon = hoon
		gotOperation = operation
		return "ok", nil
	}
	filterResponseForRestore = func(_, _ string) (string, bool, error) {
		return "", true, nil
	}

	if err := restoreAgent("~nec", "profile"); err != nil {
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
