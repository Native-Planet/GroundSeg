package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"groundseg/structs"

	"github.com/shirou/gopsutil/disk"
)

type stubFileInfo struct {
	mode os.FileMode
}

func (f stubFileInfo) Name() string       { return "stub" }
func (f stubFileInfo) Size() int64        { return 0 }
func (f stubFileInfo) Mode() os.FileMode  { return f.mode }
func (f stubFileInfo) ModTime() time.Time { return time.Time{} }
func (f stubFileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f stubFileInfo) Sys() interface{}   { return nil }

func TestSmartResultsSnapshotCopiesCurrentState(t *testing.T) {
	t.Parallel()

	SetSmartResults(map[string]bool{
		"dev-a": true,
		"dev-b": false,
	})

	snapshot := SmartResultsSnapshot()
	if got := snapshot["dev-a"]; !got {
		t.Fatalf("expected dev-a=true, got %v", got)
	}
	if got := snapshot["dev-b"]; got {
		t.Fatalf("expected dev-b=false, got %v", got)
	}

	snapshot["dev-a"] = false
	if got := SmartResultsSnapshot()["dev-a"]; !got {
		t.Fatalf("expected snapshot to be independent of internal state")
	}
}

func TestListHardDisks(t *testing.T) {
	t.Parallel()

	seams := DiskSeams{
		RunDiskCommandFn: func(string, ...string) (string, error) {
			return "", errors.New("lsblk failed")
		},
	}
	if _, err := ListHardDisks(seams); err == nil {
		t.Fatal("expected command failure from ListHardDisks")
	}

	seams.RunDiskCommandFn = func(string, ...string) (string, error) {
		return `{"invalid-json`, nil
	}
	if _, err := ListHardDisks(seams); err == nil {
		t.Fatal("expected json unmarshal failure from ListHardDisks")
	}

	seams.RunDiskCommandFn = func(string, ...string) (string, error) {
		return `{"blockdevices":[{"name":"sda","mountpoints":["/"]}]}`, nil
	}
	devices, err := ListHardDisks(seams)
	if err != nil {
		t.Fatalf("unexpected ListHardDisks error: %v", err)
	}
	if len(devices.BlockDevices) != 1 || devices.BlockDevices[0].Name != "sda" {
		t.Fatalf("unexpected parsed devices: %#v", devices.BlockDevices)
	}
}

func TestIsDevMounted(t *testing.T) {
	t.Parallel()
	if !IsDevMounted(structs.BlockDev{Mountpoints: []string{"", "/mnt"}}) {
		t.Fatal("expected mountpoint to be detected")
	}
	if IsDevMounted(structs.BlockDev{}) {
		t.Fatal("expected no mountpoint to report unmounted")
	}
}

func TestSmartCheckAllDrives(t *testing.T) {
	t.Parallel()

	calls := map[string]int{}
	seams := DiskSeams{
		CheckSataDriveFn: func(name string) (bool, error) {
			calls[name]++
			return name == "sda", nil
		},
		CheckNvmeDriveFn: func(name string) (bool, error) {
			calls[name]++
			return name != "nvme0n1", nil
		},
	}
	result := SmartCheckAllDrives(structs.LSBLKDevice{
		BlockDevices: []structs.BlockDev{
			{Name: "loop0"},
			{Name: "sda"},
			{Name: "nvme0n1"},
		},
	}, seams)
	if result["sda"] != true || result["nvme0n1"] != false {
		t.Fatalf("unexpected smart results: %#v", result)
	}
	if _, found := result["loop0"]; found {
		t.Fatalf("unsupported block device should not be checked: %#v", result)
	}
	if calls["sda"] != 1 || calls["nvme0n1"] != 1 {
		t.Fatalf("unexpected check calls: %#v", calls)
	}
}

func TestSetupTmpDirSkipsWhenNotMountedMMC(t *testing.T) {
	t.Parallel()

	var removeCalls int
	seams := DiskSeams{
		RemoveMultipartFn: func(path string) error {
			removeCalls++
			if path != "/tmp" {
				t.Fatalf("expected cleanup path /tmp, got %q", path)
			}
			return nil
		},
		ListPartitionsFn: func(_ bool) ([]disk.PartitionStat, error) {
			return []disk.PartitionStat{
				{Device: "/dev/sda1", Mountpoint: "/"},
			}, nil
		},
	}
	if err := SetupTmpDir(seams); err != nil {
		t.Fatalf("SetupTmpDir returned error: %v", err)
	}
	if removeCalls != 1 {
		t.Fatalf("expected one multipart cleanup call, got %d", removeCalls)
	}
}

func TestSetupTmpDirReplacesNonSymlinkMmcMount(t *testing.T) {
	t.Parallel()

	var mkdirAllCalls, removeAllCalls, symlinkCalls int
	seams := DiskSeams{
		RemoveMultipartFn: func(string) error { return nil },
		ListPartitionsFn: func(_ bool) ([]disk.PartitionStat, error) {
			return []disk.PartitionStat{
				{Device: "/dev/mmcblk0p1", Mountpoint: "/tmp"},
			}, nil
		},
		LstatFn: func(path string) (os.FileInfo, error) {
			return stubFileInfo{mode: 0}, nil
		},
		MkdirAllFn: func(path string, _ os.FileMode) error {
			if path != "/media/data/tmp" {
				t.Fatalf("expected alternate tmp dir /media/data/tmp, got %q", path)
			}
			mkdirAllCalls++
			return nil
		},
		RemoveAllFn: func(path string) error {
			if path != "/tmp" {
				t.Fatalf("expected remove /tmp, got %q", path)
			}
			removeAllCalls++
			return nil
		},
		SymlinkFn: func(oldname, newname string) error {
			if oldname != "/media/data/tmp" || newname != "/tmp" {
				t.Fatalf("unexpected symlink args: old=%q new=%q", oldname, newname)
			}
			symlinkCalls++
			return nil
		},
	}
	if err := SetupTmpDir(seams); err != nil {
		t.Fatalf("SetupTmpDir returned error: %v", err)
	}
	if mkdirAllCalls != 1 || removeAllCalls != 1 || symlinkCalls != 1 {
		t.Fatalf("expected tmp redirect calls, got mkdir=%d remove=%d symlink=%d", mkdirAllCalls, removeAllCalls, symlinkCalls)
	}
}

func TestSetupTmpDirPreservesExistingSymlink(t *testing.T) {
	t.Parallel()

	var mkdirAllCalls int
	seams := DiskSeams{
		RemoveMultipartFn: func(string) error { return nil },
		ListPartitionsFn: func(_ bool) ([]disk.PartitionStat, error) {
			return []disk.PartitionStat{
				{Device: "/dev/mmcblk0p1", Mountpoint: "/tmp"},
			}, nil
		},
		LstatFn: func(path string) (os.FileInfo, error) {
			return stubFileInfo{mode: os.ModeSymlink}, nil
		},
		MkdirAllFn: func(string, os.FileMode) error {
			mkdirAllCalls++
			return nil
		},
	}
	if err := SetupTmpDir(seams); err != nil {
		t.Fatalf("SetupTmpDir returned error: %v", err)
	}
	if mkdirAllCalls != 0 {
		t.Fatal("did not expect tmp redirect when /tmp is already a symlink")
	}
}

func TestIsMountedMMC(t *testing.T) {
	t.Parallel()

	seams := DiskSeams{
		ListPartitionsFn: func(_ bool) ([]disk.PartitionStat, error) {
			return []disk.PartitionStat{
				{Device: "/dev/mmcblk0p1", Mountpoint: "/media/data"},
				{Device: "/dev/sda1", Mountpoint: "/"},
			}, nil
		},
	}
	if mounted, err := IsMountedMMC("/media/data/tmp", seams); err != nil {
		t.Fatalf("IsMountedMMC returned error: %v", err)
	} else if !mounted {
		t.Fatal("expected /media/data/tmp to be mounted on mmc device via parent mount")
	}

	seams = DiskSeams{
		ListPartitionsFn: func(_ bool) ([]disk.PartitionStat, error) {
			return []disk.PartitionStat{
				{Device: "/dev/sda1", Mountpoint: "/"},
			}, nil
		},
	}
	if mounted, err := IsMountedMMC("/media/data/tmp", seams); err != nil {
		t.Fatalf("IsMountedMMC returned error: %v", err)
	} else if mounted {
		t.Fatal("expected non-mmc mount to return false")
	}
}

func TestIsMountedMMCPropagatesListError(t *testing.T) {
	t.Parallel()
	listErr := errors.New("partition list failed")
	_, err := IsMountedMMC("/tmp", DiskSeams{
		ListPartitionsFn: func(_ bool) ([]disk.PartitionStat, error) {
			return nil, listErr
		},
	})
	if !errors.Is(err, listErr) {
		t.Fatalf("expected list error, got %v", err)
	}
}

func TestRemoveMultipartFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "multipart-1"), []byte("temp"), 0o644); err != nil {
		t.Fatalf("fixture write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatalf("fixture write: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "multipart-dir"), 0o755); err != nil {
		t.Fatalf("fixture mkdir: %v", err)
	}

	var removed []string
	seams := DiskSeams{
		ReadDirFn: func(path string) ([]os.DirEntry, error) {
			return os.ReadDir(path)
		},
		RemoveFn: func(path string) error {
			removed = append(removed, filepath.Base(path))
			return nil
		},
	}
	if err := RemoveMultipartFiles(dir, seams); err != nil {
		t.Fatalf("RemoveMultipartFiles error: %v", err)
	}
	if len(removed) != 1 || removed[0] != "multipart-1" {
		t.Fatalf("expected only multipart-1 removed, got %#v", removed)
	}
}

func TestRemoveMultipartFilesError(t *testing.T) {
	t.Parallel()

	removeErr := errors.New("remove failed")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "multipart-bad"), []byte("x"), 0o644); err != nil {
		t.Fatalf("fixture write: %v", err)
	}
	seams := DiskSeams{
		ReadDirFn: func(path string) ([]os.DirEntry, error) { return os.ReadDir(path) },
		RemoveFn:  func(string) error { return removeErr },
	}
	if err := RemoveMultipartFiles(dir, seams); err == nil || !errors.Is(err, removeErr) {
		t.Fatalf("expected remove error, got %v", err)
	}
}

func TestParseAndReconcileFstab(t *testing.T) {
	t.Parallel()

	if _, ok := ParseFstabLine(" # comment"); ok {
		t.Fatal("expected comment line ignored")
	}
	if _, ok := ParseFstabLine("too-short  "); ok {
		t.Fatal("expected malformed line ignored")
	}

	record := FstabRecord{
		Device:     "UUID=abc",
		MountPoint: "/groundseg-1",
		FSType:     "ext4",
		Options:    "defaults,nofail",
		Dump:       "0",
		Pass:       "2",
	}
	recorded := []string{
		"UUID=abc /groundseg-1 ext4 defaults 0 0",
		"tmpfs /tmp tmpfs defaults 0 0",
	}
	reconciled, changed := ReconcileFstabLines(recorded, record)
	if !changed {
		t.Fatal("expected reconcile to change mount options")
	}
	if len(reconciled) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(reconciled))
	}
	if reconciled[0] != record.Line() {
		t.Fatalf("expected normalized entry, got %q", reconciled[0])
	}

	reconciled, changed = ReconcileFstabLines(reconciled, record)
	if changed {
		t.Fatal("expected idempotent reconcile after normalization")
	}
}

func TestReadWriteFstabLines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "fstab.in")
	if err := os.WriteFile(src, []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	lines, err := ReadFstabLines(src, DefaultSeams())
	if err != nil {
		t.Fatalf("ReadFstabLines error: %v", err)
	}
	if len(lines) != 2 || lines[0] != "line1" || lines[1] != "line2" {
		t.Fatalf("unexpected lines %#v", lines)
	}

	out := filepath.Join(dir, "fstab.out")
	if err := WriteFstabLines(out, []string{"a", "b", "c"}, DiskSeams{
		OpenFileFn: func(path string, _ int, _ os.FileMode) (*os.File, error) {
			return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		},
	}); err != nil {
		t.Fatalf("WriteFstabLines error: %v", err)
	}
	content, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(content) != "a\nb\nc\n" {
		t.Fatalf("unexpected output content: %q", content)
	}
}

func TestNextGroundSegPath(t *testing.T) {
	t.Parallel()

	attempts := 0
	path, err := nextGroundSegPath(DiskSeams{
		StatFn: func(path string) (os.FileInfo, error) {
			attempts++
			if attempts <= 2 {
				return stubFileInfo{mode: os.ModeDir}, nil
			}
			return nil, os.ErrNotExist
		},
	})
	if err != nil {
		t.Fatalf("nextGroundSegPath returned error: %v", err)
	}
	if path != "/groundseg-3" {
		t.Fatalf("expected /groundseg-3, got %q", path)
	}
}

func TestNextGroundSegPathPropagatesStatError(t *testing.T) {
	t.Parallel()

	statErr := errors.New("disk full")
	if _, err := nextGroundSegPath(DiskSeams{
		StatFn: func(_ string) (os.FileInfo, error) { return nil, statErr },
	}); err == nil || !errors.Is(err, statErr) {
		t.Fatalf("expected stat error, got %v", err)
	}
}

func TestFilePathExists(t *testing.T) {
	t.Parallel()

	if exists, err := filePathExists("/missing", func(_ string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}); err != nil {
		t.Fatalf("filePathExists returned error: %v", err)
	} else if exists {
		t.Fatalf("expected missing path to return exists=false")
	}

	exists, err := filePathExists("/tmp", func(_ string) (os.FileInfo, error) {
		return stubFileInfo{mode: 0}, nil
	})
	if err != nil {
		t.Fatalf("filePathExists returned error: %v", err)
	}
	if !exists {
		t.Fatal("expected path to exist")
	}
}

func TestFormatGroundSegFilesystem(t *testing.T) {
	t.Parallel()

	var capturedUUID string
	seams := DiskSeams{
		MkfsExt4CommandFn: func(uuid, devPath string) error {
			if devPath != "/dev/sda" {
				t.Fatalf("unexpected device path %q", devPath)
			}
			if uuid == "" {
				t.Fatal("expected generated UUID")
			}
			capturedUUID = uuid
			return nil
		},
	}
	uuid, err := formatGroundSegFilesystem("sda", seams)
	if err != nil {
		t.Fatalf("formatGroundSegFilesystem error: %v", err)
	}
	if uuid != capturedUUID {
		t.Fatalf("expected returned uuid %q, got %q", capturedUUID, uuid)
	}
}

func TestReconcileGroundSegFstabWritesAndMounts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	in := filepath.Join(dir, "fstab.in")
	out := filepath.Join(dir, "fstab.out")
	if err := os.WriteFile(in, []byte("tmpfs /tmp tmpfs defaults 0 0\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	seams := DiskSeams{
		RunDiskCommandFn: func(string, ...string) (string, error) {
			return `{"blockdevices":[{"name":"sda","mountpoints":["/"]}]}`, nil
		},
		OpenFn: func(path string) (*os.File, error) {
			if path != "/etc/fstab" {
				t.Fatalf("expected /etc/fstab, got %q", path)
			}
			return os.Open(in)
		},
		OpenFileFn: func(path string, _ int, _ os.FileMode) (*os.File, error) {
			if path != "/etc/fstab" {
				t.Fatalf("expected /etc/fstab, got %q", path)
			}
			return os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		},
		MountAllCommandFn: func() error { return nil },
	}

	if err := reconcileGroundSegFstab("sda", "/groundseg-1", "uuid-1234", seams); err != nil {
		t.Fatalf("reconcileGroundSegFstab error: %v", err)
	}

	content, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(content), "/groundseg-1 ext4 defaults,nofail 0 2") {
		t.Fatalf("expected updated fstab line, got %q", content)
	}

	if !strings.Contains(string(content), "tmpfs /tmp tmpfs defaults 0 0") {
		t.Fatalf("expected preserved unrelated line, got %q", content)
	}
}

func TestReconcileGroundSegFstabSkipsMissingDevice(t *testing.T) {
	t.Parallel()

	var mounted bool
	seams := DiskSeams{
		RunDiskCommandFn: func(string, ...string) (string, error) {
			return `{"blockdevices":[{"name":"nvme0n1","mountpoints":["/"]}]}`, nil
		},
		MountAllCommandFn: func() error {
			mounted = true
			return nil
		},
	}
	if err := reconcileGroundSegFstab("sda", "/groundseg-1", "uuid-1234", seams); err != nil {
		t.Fatalf("reconcileGroundSegFstab error: %v", err)
	}
	if mounted {
		t.Fatal("did not expect mount to run when selected device is absent")
	}
}

func TestCreateGroundSegFilesystem(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	in := filepath.Join(dir, "fstab.in")
	out := filepath.Join(dir, "fstab.out")
	if err := os.WriteFile(in, []byte("tmpfs /tmp tmpfs defaults 0 0\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var mkdirCalled, mountCalled bool
	seams := DiskSeams{
		StatFn: func(_ string) (os.FileInfo, error) { return nil, os.ErrNotExist },
		MkdirFn: func(path string, perm os.FileMode) error {
			if path != "/groundseg-1" {
				t.Fatalf("expected /groundseg-1, got %q", path)
			}
			mkdirCalled = true
			return nil
		},
		MkfsExt4CommandFn: func(uuid, devPath string) error {
			if devPath != "/dev/sda" {
				t.Fatalf("unexpected device path %q", devPath)
			}
			return nil
		},
		OpenFn: func(path string) (*os.File, error) {
			if path != "/etc/fstab" {
				t.Fatalf("expected /etc/fstab, got %q", path)
			}
			return os.Open(in)
		},
		OpenFileFn: func(path string, _ int, _ os.FileMode) (*os.File, error) {
			if path != "/etc/fstab" {
				t.Fatalf("expected /etc/fstab, got %q", path)
			}
			return os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		},
		RunDiskCommandFn: func(string, ...string) (string, error) {
			return `{"blockdevices":[{"name":"sda","mountpoints":["/"]}]}`, nil
		},
		MountAllCommandFn: func() error {
			mountCalled = true
			return nil
		},
	}

	path, err := CreateGroundSegFilesystem("sda", seams)
	if err != nil {
		t.Fatalf("CreateGroundSegFilesystem error: %v", err)
	}
	if path != "/groundseg-1" {
		t.Fatalf("expected /groundseg-1, got %q", path)
	}
	if !mkdirCalled || !mountCalled {
		t.Fatalf("expected directory creation and mount, got mkdir=%v mount=%v", mkdirCalled, mountCalled)
	}

	content, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(content), "/groundseg-1 ext4 defaults,nofail 0 2") {
		t.Fatalf("expected reconciled fstab entry, got %q", content)
	}
}

func TestCreateGroundSegFilesystemRejectsPathLookupFailure(t *testing.T) {
	t.Parallel()

	lookupErr := errors.New("cannot stat")
	_, err := CreateGroundSegFilesystem("sda", DiskSeams{
		StatFn: func(_ string) (os.FileInfo, error) {
			return nil, lookupErr
		},
	})
	if !errors.Is(err, lookupErr) {
		t.Fatalf("expected lookup error, got %v", err)
	}
}
