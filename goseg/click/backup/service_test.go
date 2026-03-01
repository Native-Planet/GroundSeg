package backup

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func resetBackupSeams() {
	executeClickCommandForBackup = func(_, _, _, _, _, _ string) (string, error) {
		return "", nil
	}
	filterResponseForBackup = func(_ string, _ string) (string, bool, error) {
		return "", true, nil
	}
	joinGapForBackup = func(parts []string) string {
		return strings.Join(parts, "  ")
	}
	backupAgentFn = backupAgent
}

func TestBackupAgentExecFailure(t *testing.T) {
	t.Cleanup(resetBackupSeams)
	executeClickCommandForBackup = func(_, _, _, _, _, _ string) (string, error) {
		return "", errors.New("exec failed")
	}

	err := backupAgent("~zod", "groups")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "exec failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackupAgentFilterFailure(t *testing.T) {
	t.Cleanup(resetBackupSeams)
	executeClickCommandForBackup = func(_, _, _, _, _, _ string) (string, error) {
		return "response", nil
	}
	filterResponseForBackup = func(_, _ string) (string, bool, error) {
		return "", false, errors.New("parse failed")
	}

	err := backupAgent("~zod", "groups")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "failed to get exec") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackupAgentPokeFailure(t *testing.T) {
	t.Cleanup(resetBackupSeams)
	executeClickCommandForBackup = func(_, _, _, _, _, _ string) (string, error) {
		return "response", nil
	}
	filterResponseForBackup = func(_, _ string) (string, bool, error) {
		return "", false, nil
	}

	err := backupAgent("~zod", "groups")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "failed poke") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackupAgentBuildsExpectedCommand(t *testing.T) {
	t.Cleanup(resetBackupSeams)

	var gotPatp, gotFile, gotHoon, gotOperation string
	executeClickCommandForBackup = func(patp, file, hoon, _, _, operation string) (string, error) {
		gotPatp = patp
		gotFile = file
		gotHoon = hoon
		gotOperation = operation
		return "[0 %avow 0 %noun %success]", nil
	}
	filterResponseForBackup = func(_, _ string) (string, bool, error) {
		return "", true, nil
	}
	joinGapForBackup = func(parts []string) string {
		return strings.Join(parts, "  ")
	}

	if err := backupAgent("~zod", "profile"); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if gotPatp != "~zod" {
		t.Fatalf("unexpected patp: %s", gotPatp)
	}
	if gotFile != "backup-profile" {
		t.Fatalf("unexpected file: %s", gotFile)
	}
	if gotOperation != "Click backup-profile" {
		t.Fatalf("unexpected operation: %s", gotOperation)
	}
	if !strings.Contains(gotHoon, "/gv/profile/$") {
		t.Fatalf("hoon did not include agent scry path: %s", gotHoon)
	}
	if !strings.Contains(gotHoon, "/backup-profile/jam") {
		t.Fatalf("hoon did not include backup path: %s", gotHoon)
	}
}

func TestBackupTlonRunsAllAgents(t *testing.T) {
	t.Cleanup(resetBackupSeams)
	var got []string
	backupAgentFn = func(_ string, agent string) error {
		got = append(got, agent)
		return nil
	}

	if err := BackupTlon("~zod"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"activity", "channels", "channels-server", "groups", "profile", "chat"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected agents: got %v want %v", got, want)
	}
}

func TestBackupTlonAggregatesErrors(t *testing.T) {
	t.Cleanup(resetBackupSeams)
	backupAgentFn = func(_ string, agent string) error {
		if agent == "channels" || agent == "profile" {
			return errors.New("boom")
		}
		return nil
	}

	err := BackupTlon("~zod")
	if err == nil {
		t.Fatalf("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "channels: boom") || !strings.Contains(msg, "profile: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}
