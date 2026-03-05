package backup

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func resetBackupSeams() {
	SetRuntime(nil)
}

func withBackupRuntime(mutator func(*backupRuntime)) {
	runtime := defaultBackupRuntime()
	if mutator != nil {
		mutator(&runtime)
	}
	SetRuntime(runtime)
}

func TestBackupAgentExecFailure(t *testing.T) {
	t.Cleanup(resetBackupSeams)
	withBackupRuntime(func(runtime *backupRuntime) {
		runtime.executeClickCommandForBackup = func(_, _, _, _, _, _ string, _ func(string)) (string, error) {
			return "", errors.New("exec failed")
		}
	})

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
	withBackupRuntime(func(runtime *backupRuntime) {
		runtime.executeClickCommandForBackup = func(_, _, _, _, _, _ string, _ func(string)) (string, error) {
			return "", errors.New("parse failed")
		}
	})

	err := backupAgent("~zod", "groups")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "parse failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackupAgentPokeFailure(t *testing.T) {
	t.Cleanup(resetBackupSeams)
	withBackupRuntime(func(runtime *backupRuntime) {
		runtime.executeClickCommandForBackup = func(_, _, _, _, _, _ string, _ func(string)) (string, error) {
			return "", errors.New("click command failed")
		}
	})

	err := backupAgent("~zod", "groups")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "click command failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackupAgentBuildsExpectedCommand(t *testing.T) {
	t.Cleanup(resetBackupSeams)

	var gotPatp, gotFile, gotHoon, gotOperation string
	withBackupRuntime(func(runtime *backupRuntime) {
		runtime.executeClickCommandForBackup = func(
			patp, file, hoon, sourcePath, successToken, operation string,
			_clearLusCode func(string),
		) (string, error) {
			gotPatp = patp
			gotFile = file
			gotHoon = hoon
			gotOperation = operation
			return "ok", nil
		}
		runtime.joinGapForBackup = func(parts []string) string {
			return strings.Join(parts, "  ")
		}
	})

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
	withBackupRuntime(func(runtime *backupRuntime) {
		runtime.executeClickCommandForBackup = func(patp, file, _, _, _, _ string, _ func(string)) (string, error) {
			got = append(got, strings.TrimPrefix(file, "backup-"))
			return "", nil
		}
		runtime.joinGapForBackup = func(parts []string) string { return strings.Join(parts, "  ") }
	})

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
	withBackupRuntime(func(runtime *backupRuntime) {
		runtime.executeClickCommandForBackup = func(_, _, _, _, _, _ string, _ func(string)) (string, error) {
			return "", nil
		}
		runtime.joinGapForBackup = func(parts []string) string { return strings.Join(parts, "  ") }
	})

	withBackupRuntime(func(runtime *backupRuntime) {
		runtime.executeClickCommandForBackup = func(_, file, _, _, _, _ string, _ func(string)) (string, error) {
			if file == "backup-channels" || file == "backup-profile" {
				return "", errors.New("boom")
			}
			return "", nil
		}
		runtime.joinGapForBackup = func(parts []string) string { return strings.Join(parts, "  ") }
	})
	err := BackupTlon("~zod")
	if err == nil {
		t.Fatalf("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "channels: click command failed for backup-channels") ||
		!strings.Contains(msg, "profile: click command failed for backup-profile") {
		t.Fatalf("unexpected error: %v", err)
	}
}
