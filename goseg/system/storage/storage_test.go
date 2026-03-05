package storage

import (
	"errors"
	"os"
	"path/filepath"
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
