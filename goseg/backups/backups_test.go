package backups

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func resetBackupSeams() func() {
	originalBackupTlon := backupTlonFn
	originalNowFn := nowFn
	originalGetVolume := getVolumeFn
	return func() {
		backupTlonFn = originalBackupTlon
		nowFn = originalNowFn
		getVolumeFn = originalGetVolume
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func TestCreateBackupCreatesCompressedBackupsAndPrunesOldFiles(t *testing.T) {
	restore := resetBackupSeams()
	t.Cleanup(restore)

	backupTlonFn = func(string) error { return nil }
	nowFn = func() time.Time {
		// Sunday and first day of month to exercise daily+weekly+monthly writes.
		return time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
	}

	volumeRoot := t.TempDir()
	patp := "zod"
	putDir := filepath.Join(volumeRoot, patp, ".urb", "put")
	if err := os.MkdirAll(putDir, 0o755); err != nil {
		t.Fatalf("mkdir put dir: %v", err)
	}
	writeFile(t, filepath.Join(putDir, "one.jam"), "jam-content-1")
	writeFile(t, filepath.Join(putDir, "two.jam"), "jam-content-2")
	getVolumeFn = func(string) (string, error) {
		return volumeRoot, nil
	}

	dailyDir := t.TempDir()
	weeklyDir := t.TempDir()
	monthlyDir := t.TempDir()
	for i := 1; i <= 4; i++ {
		ts := strconv.Itoa(i)
		writeFile(t, filepath.Join(dailyDir, ts), "old")
		writeFile(t, filepath.Join(weeklyDir, ts), "old")
		writeFile(t, filepath.Join(monthlyDir, ts), "old")
	}

	if err := CreateBackup(patp, dailyDir, weeklyDir, monthlyDir); err != nil {
		t.Fatalf("CreateBackup returned error: %v", err)
	}

	timestamp := strconv.FormatInt(nowFn().Unix(), 10)
	for _, dir := range []string{dailyDir, weeklyDir, monthlyDir} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read backup dir %s: %v", dir, err)
		}
		if len(entries) != 3 {
			t.Fatalf("expected pruned backups to 3 files in %s, got %d", dir, len(entries))
		}
		if _, err := os.Stat(filepath.Join(dir, timestamp)); err != nil {
			t.Fatalf("expected new timestamp backup in %s: %v", dir, err)
		}
	}
}

func TestCreateBackupReturnsErrorWhenBackupRoutineFails(t *testing.T) {
	restore := resetBackupSeams()
	t.Cleanup(restore)

	backupTlonFn = func(string) error { return errors.New("tlon failure") }
	err := CreateBackup("zod", t.TempDir(), t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatal("expected backup routine failure")
	}
}

func TestCreateBackupReturnsErrorWhenNoJamFilesPresent(t *testing.T) {
	restore := resetBackupSeams()
	t.Cleanup(restore)

	backupTlonFn = func(string) error { return nil }
	volumeRoot := t.TempDir()
	patp := "zod"
	putDir := filepath.Join(volumeRoot, patp, ".urb", "put")
	if err := os.MkdirAll(putDir, 0o755); err != nil {
		t.Fatalf("mkdir put dir: %v", err)
	}
	getVolumeFn = func(string) (string, error) {
		return volumeRoot, nil
	}

	err := CreateBackup(patp, t.TempDir(), t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatal("expected error when no .jam files are present")
	}
}

func TestCreateBackupReturnsErrorOnMissingVolume(t *testing.T) {
	restore := resetBackupSeams()
	t.Cleanup(restore)

	backupTlonFn = func(string) error { return nil }
	getVolumeFn = func(string) (string, error) {
		return "", nil
	}

	err := CreateBackup("zod", t.TempDir(), t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatal("expected error when volume path is empty")
	}
}
