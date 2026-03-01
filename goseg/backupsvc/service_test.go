package backupsvc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func resetBackupSvcSeams() {
	isMountedMMCFn = func(basePath string) (bool, error) {
		return false, nil
	}
	mkdirAllFn = os.MkdirAll
	readDirFn = os.ReadDir
	createBackupFn = func(string, string, string, string) error {
		return nil
	}
	uploadBackupFn = func(string, string, string) error {
		return nil
	}
}

func TestResolveBackupRootUsesMMCWhenMounted(t *testing.T) {
	t.Cleanup(resetBackupSvcSeams)
	isMountedMMCFn = func(string) (bool, error) { return true, nil }
	if got := ResolveBackupRoot("/base"); got != "/media/data/backup" {
		t.Fatalf("expected mmc backup root, got %s", got)
	}
}

func TestResolveBackupRootFallsBackOnError(t *testing.T) {
	t.Cleanup(resetBackupSvcSeams)
	isMountedMMCFn = func(string) (bool, error) { return false, errors.New("probe failed") }
	if got := ResolveBackupRoot("/base"); got != filepath.Join("/base", "backup") {
		t.Fatalf("expected fallback backup root, got %s", got)
	}
}

func TestCreateLocalBackupEnsuresDirsAndDelegates(t *testing.T) {
	t.Cleanup(resetBackupSvcSeams)
	root := t.TempDir()
	patp := "~zod"
	createCalled := false
	createBackupFn = func(ship, daily, weekly, monthly string) error {
		createCalled = true
		if ship != patp {
			t.Fatalf("unexpected ship: %s", ship)
		}
		for _, dir := range []string{daily, weekly, monthly} {
			if _, err := os.Stat(dir); err != nil {
				t.Fatalf("expected backup dir to exist: %s (%v)", dir, err)
			}
		}
		return nil
	}
	if err := CreateLocalBackup(patp, root); err != nil {
		t.Fatalf("CreateLocalBackup failed: %v", err)
	}
	if !createCalled {
		t.Fatal("expected create backup delegate call")
	}
}

func TestLatestBackupFileChoosesMostRecentAcrossPeriods(t *testing.T) {
	t.Cleanup(resetBackupSvcSeams)
	root := t.TempDir()
	patp := "~zod"
	dirs, err := EnsureLocalDirs(root, patp)
	if err != nil {
		t.Fatalf("ensure local dirs failed: %v", err)
	}
	files := map[string]string{
		filepath.Join(dirs.Daily, "10"):    "daily",
		filepath.Join(dirs.Weekly, "500"):  "weekly",
		filepath.Join(dirs.Monthly, "300"): "monthly",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write file failed: %v", err)
		}
	}
	got, err := LatestBackupFile(root, patp)
	if err != nil {
		t.Fatalf("LatestBackupFile failed: %v", err)
	}
	want := filepath.Join(dirs.Weekly, "500")
	if got != want {
		t.Fatalf("expected newest backup %s, got %s", want, got)
	}
}

func TestUploadLatestBackupDelegatesWithNewestFile(t *testing.T) {
	t.Cleanup(resetBackupSvcSeams)
	root := t.TempDir()
	patp := "~nec"
	dirs, err := EnsureLocalDirs(root, patp)
	if err != nil {
		t.Fatalf("ensure local dirs failed: %v", err)
	}
	latest := filepath.Join(dirs.Daily, "200")
	if err := os.WriteFile(filepath.Join(dirs.Daily, "100"), []byte("old"), 0644); err != nil {
		t.Fatalf("write old backup failed: %v", err)
	}
	if err := os.WriteFile(latest, []byte("new"), 0644); err != nil {
		t.Fatalf("write latest backup failed: %v", err)
	}
	var gotPath string
	uploadBackupFn = func(ship, password, path string) error {
		if ship != patp || password != "pw" {
			t.Fatalf("unexpected upload args: %s %s %s", ship, password, path)
		}
		gotPath = path
		return nil
	}
	if err := UploadLatestBackup(patp, "pw", root); err != nil {
		t.Fatalf("UploadLatestBackup failed: %v", err)
	}
	if gotPath != latest {
		t.Fatalf("expected latest backup path %s, got %s", latest, gotPath)
	}
}

func TestMostRecentDailyBackupTimeReturnsZeroWhenMissing(t *testing.T) {
	t.Cleanup(resetBackupSvcSeams)
	got, err := MostRecentDailyBackupTime(t.TempDir(), "~zod")
	if err != nil {
		t.Fatalf("MostRecentDailyBackupTime failed: %v", err)
	}
	if !got.IsZero() {
		t.Fatalf("expected zero value time when no backups, got %v", got)
	}
}

func TestMostRecentDailyBackupTimeUsesNewestTimestamp(t *testing.T) {
	t.Cleanup(resetBackupSvcSeams)
	root := t.TempDir()
	patp := "~bus"
	dirs, err := EnsureLocalDirs(root, patp)
	if err != nil {
		t.Fatalf("ensure local dirs failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirs.Daily, "100"), []byte("older"), 0644); err != nil {
		t.Fatalf("write older backup failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirs.Daily, "200"), []byte("newer"), 0644); err != nil {
		t.Fatalf("write newer backup failed: %v", err)
	}
	got, err := MostRecentDailyBackupTime(root, patp)
	if err != nil {
		t.Fatalf("MostRecentDailyBackupTime failed: %v", err)
	}
	if got.Unix() != 200 {
		t.Fatalf("expected unix 200, got %d", got.Unix())
	}
	if got.Location() != time.Local && got.Location() != time.UTC {
		// keep assertion lightweight and robust across platforms.
		t.Fatalf("unexpected location on result: %v", got.Location())
	}
}
