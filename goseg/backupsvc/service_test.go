package backupsvc

import (
	"fmt"
	"io/fs"
	"os"
	"testing"
	"time"
)

type fakeDirEntry struct {
	name  string
	isDir bool
}

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return f.isDir }
func (f fakeDirEntry) Type() fs.FileMode          { return 0 }
func (f fakeDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func TestResolveBackupRootReturnsMMCPathWhenAvailable(t *testing.T) {
	restore := resetBackupsvcDependencies()
	defer restore()

	isMountedMMCFn = func(path string) (bool, error) {
		if path != "/mnt/test" {
			t.Fatalf("expected mount check for /mnt/test, got %s", path)
		}
		return true, nil
	}

	if got := ResolveBackupRoot("/mnt/test"); got != "/media/data/backup" {
		t.Fatalf("unexpected backup root, got %q", got)
	}
}

func TestEnsureLocalDirsCreatesAllDirectories(t *testing.T) {
	restore := resetBackupsvcDependencies()
	defer restore()

	created := []string{}
	mkdirAllFn = func(path string, perm os.FileMode) error {
		created = append(created, path)
		return nil
	}

	_, err := EnsureLocalDirs("/tmp/base", "~zod")
	if err != nil {
		t.Fatalf("EnsureLocalDirs returned error: %v", err)
	}

	expected := []string{
		"/tmp/base/~zod",
		"/tmp/base/~zod/daily",
		"/tmp/base/~zod/weekly",
		"/tmp/base/~zod/monthly",
	}
	if len(created) != len(expected) {
		t.Fatalf("expected %d mkdir calls, got %d", len(expected), len(created))
	}
	for i, got := range created {
		if got != expected[i] {
			t.Fatalf("expected mkdir call %d to be %q, got %q", i, expected[i], got)
		}
	}
}

func TestLatestBackupFileSelectsNewestTimestamp(t *testing.T) {
	restore := resetBackupsvcDependencies()
	defer restore()

	readDirFn = func(path string) ([]fs.DirEntry, error) {
		switch path {
		case "/base/~zod/daily":
			return []fs.DirEntry{
				fakeDirEntry{name: "100"},
				fakeDirEntry{name: "200"},
				fakeDirEntry{name: "bad"},
				fakeDirEntry{name: "ignored", isDir: true},
			}, nil
		case "/base/~zod/weekly":
			return []fs.DirEntry{
				fakeDirEntry{name: "300"},
			}, nil
		default:
			return []fs.DirEntry{}, nil
		}
	}

	path, err := LatestBackupFile("/base", "~zod")
	if err != nil {
		t.Fatalf("LatestBackupFile returned error: %v", err)
	}
	if path != "/base/~zod/weekly/300" {
		t.Fatalf("expected newest backup path /base/~zod/weekly/300, got %q", path)
	}
}

func TestUploadLatestBackupCallsStartramUpload(t *testing.T) {
	restore := resetBackupsvcDependencies()
	defer restore()

	readDirFn = func(path string) ([]fs.DirEntry, error) {
		if path == "/base/~zod" {
			return []fs.DirEntry{
				fakeDirEntry{name: "100"},
			}, nil
		}
		if path == "/base/~zod/daily" || path == "/base/~zod/weekly" || path == "/base/~zod/monthly" {
			return []fs.DirEntry{}, nil
		}
		return []fs.DirEntry{}, nil
	}

	var uploaded struct {
		patp     string
		password string
		latest   string
	}
	uploadBackupFn = func(patp, password, latestBackup string) error {
		uploaded = struct {
			patp     string
			password string
			latest   string
		}{
			patp:     patp,
			password: password,
			latest:   latestBackup,
		}
		return nil
	}

	if err := UploadLatestBackup("~zod", "secret", "/base"); err != nil {
		t.Fatalf("UploadLatestBackup returned error: %v", err)
	}
	if uploaded.latest != "/base/~zod/100" || uploaded.patp != "~zod" || uploaded.password != "secret" {
		t.Fatalf("unexpected upload args: %+v", uploaded)
	}
}

func TestMostRecentDailyBackupTimeReturnsLatest(t *testing.T) {
	restore := resetBackupsvcDependencies()
	defer restore()

	readDirFn = func(path string) ([]fs.DirEntry, error) {
		if path == "/base/~zod/daily" {
			return []fs.DirEntry{
				fakeDirEntry{name: "100"},
				fakeDirEntry{name: "250"},
				fakeDirEntry{name: "bad", isDir: false},
			}, nil
		}
		return []fs.DirEntry{}, nil
	}

	got, err := MostRecentDailyBackupTime("/base", "~zod")
	if err != nil {
		t.Fatalf("MostRecentDailyBackupTime returned error: %v", err)
	}
	want := time.Unix(250, 0)
	if got.Unix() != want.Unix() {
		t.Fatalf("expected latest daily backup time %v, got %v", want, got)
	}
}

func TestCreateLocalBackupPropagatesCreateErrors(t *testing.T) {
	restore := resetBackupsvcDependencies()
	defer restore()

	mkdirAllFn = func(_ string, _ os.FileMode) error { return nil }
	expected := fmt.Errorf("create backup error")
	createBackupFn = func(_, _, _, _ string) error {
		return expected
	}

	err := CreateLocalBackup("~zod", "/base")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "create local backup for ~zod: create backup error" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func resetBackupsvcDependencies() func() {
	origIsMounted := isMountedMMCFn
	origMkdirAll := mkdirAllFn
	origReadDir := readDirFn
	origCreateBackup := createBackupFn
	origUploadBackup := uploadBackupFn

	return func() {
		isMountedMMCFn = origIsMounted
		mkdirAllFn = origMkdirAll
		readDirFn = origReadDir
		createBackupFn = origCreateBackup
		uploadBackupFn = origUploadBackup
	}
}
