package system

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"groundseg/structs"

	"github.com/shirou/gopsutil/disk"
)

func resetDiskSeams() func() {
	originalRunDiskCommand := runDiskCommandFn
	originalListPartitions := listPartitionsFn
	originalCheckSata := checkSataDriveFn
	originalCheckNvme := checkNvmeDriveFn
	originalRemoveMultipart := removeMultipartFn
	originalIsMountedMMC := isMountedMMCFn
	originalMkdirAll := mkdirAllFn
	originalRemoveAll := removeAllFn
	originalSymlink := symlinkFn
	originalLstat := lstatFn
	originalMkdir := mkdirFn
	originalOpen := openFn
	originalOpenFile := openFileFn
	originalMountAll := mountAllCommandFn
	originalMkfsExt4 := mkfsExt4CommandFn
	return func() {
		runDiskCommandFn = originalRunDiskCommand
		listPartitionsFn = originalListPartitions
		checkSataDriveFn = originalCheckSata
		checkNvmeDriveFn = originalCheckNvme
		removeMultipartFn = originalRemoveMultipart
		isMountedMMCFn = originalIsMountedMMC
		mkdirAllFn = originalMkdirAll
		removeAllFn = originalRemoveAll
		symlinkFn = originalSymlink
		lstatFn = originalLstat
		mkdirFn = originalMkdir
		openFn = originalOpen
		openFileFn = originalOpenFile
		mountAllCommandFn = originalMountAll
		mkfsExt4CommandFn = originalMkfsExt4
	}
}

func TestListHardDisksParsesJSON(t *testing.T) {
	restore := resetDiskSeams()
	t.Cleanup(restore)

	runDiskCommandFn = func(string, ...string) (string, error) {
		return `{"blockdevices":[{"name":"sda","mountpoints":["/"]}]}`, nil
	}
	devices, err := ListHardDisks()
	if err != nil {
		t.Fatalf("ListHardDisks returned error: %v", err)
	}
	if len(devices.BlockDevices) != 1 || devices.BlockDevices[0].Name != "sda" {
		t.Fatalf("unexpected parsed devices: %+v", devices.BlockDevices)
	}
}

func TestListHardDisksPropagatesErrors(t *testing.T) {
	restore := resetDiskSeams()
	t.Cleanup(restore)

	runDiskCommandFn = func(string, ...string) (string, error) {
		return "", errors.New("lsblk failed")
	}
	if _, err := ListHardDisks(); err == nil {
		t.Fatal("expected command failure from ListHardDisks")
	}

	runDiskCommandFn = func(string, ...string) (string, error) {
		return `{invalid-json`, nil
	}
	if _, err := ListHardDisks(); err == nil {
		t.Fatal("expected json unmarshal failure from ListHardDisks")
	}
}

func TestRemoveMultipartFilesRemovesOnlyMultipartPrefix(t *testing.T) {
	dir := t.TempDir()
	multipartFile := filepath.Join(dir, "multipart-001")
	regularFile := filepath.Join(dir, "keep-me")
	if err := os.WriteFile(multipartFile, []byte("temp"), 0o644); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := os.WriteFile(regularFile, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write regular file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "multipart-dir"), 0o755); err != nil {
		t.Fatalf("write multipart directory: %v", err)
	}

	if err := RemoveMultipartFiles(dir); err != nil {
		t.Fatalf("RemoveMultipartFiles returned error: %v", err)
	}
	if _, err := os.Stat(multipartFile); !os.IsNotExist(err) {
		t.Fatalf("expected multipart file to be removed, err=%v", err)
	}
	if _, err := os.Stat(regularFile); err != nil {
		t.Fatalf("expected regular file to remain, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "multipart-dir")); err != nil {
		t.Fatalf("expected multipart directory to remain, err=%v", err)
	}
}

func TestIsMountedMMCUsesPartitionHierarchy(t *testing.T) {
	restore := resetDiskSeams()
	t.Cleanup(restore)

	listPartitionsFn = func(bool) ([]disk.PartitionStat, error) {
		return []disk.PartitionStat{
			{Device: "/dev/mmcblk0p1", Mountpoint: "/media/data"},
			{Device: "/dev/sda1", Mountpoint: "/"},
		}, nil
	}
	mounted, err := IsMountedMMC("/media/data/tmp/child")
	if err != nil {
		t.Fatalf("IsMountedMMC returned error: %v", err)
	}
	if !mounted {
		t.Fatal("expected true when parent mountpoint is mmc")
	}

	listPartitionsFn = func(bool) ([]disk.PartitionStat, error) {
		return []disk.PartitionStat{
			{Device: "/dev/sda1", Mountpoint: "/media/data"},
		}, nil
	}
	mounted, err = IsMountedMMC("/media/data/tmp/child")
	if err != nil {
		t.Fatalf("IsMountedMMC returned error: %v", err)
	}
	if mounted {
		t.Fatal("expected false for non-mmc partition")
	}
}

func TestSmartCheckAllDrivesDelegatesByDeviceType(t *testing.T) {
	restore := resetDiskSeams()
	t.Cleanup(restore)

	sataCalls := 0
	nvmeCalls := 0
	checkSataDriveFn = func(name string) (bool, error) {
		sataCalls++
		if name != "sda" {
			t.Fatalf("unexpected sata drive name %s", name)
		}
		return true, nil
	}
	checkNvmeDriveFn = func(name string) (bool, error) {
		nvmeCalls++
		if name != "nvme0n1" {
			t.Fatalf("unexpected nvme drive name %s", name)
		}
		return false, nil
	}

	results := SmartCheckAllDrives(structs.LSBLKDevice{
		BlockDevices: []structs.BlockDev{
			{Name: "sda"},
			{Name: "nvme0n1"},
			{Name: "loop0"},
		},
	})

	if sataCalls != 1 || nvmeCalls != 1 {
		t.Fatalf("unexpected smart-check call counts: sata=%d nvme=%d", sataCalls, nvmeCalls)
	}
	if results["sda"] != true {
		t.Fatalf("expected sda result true, got %+v", results["sda"])
	}
	if results["nvme0n1"] != false {
		t.Fatalf("expected nvme result false, got %+v", results["nvme0n1"])
	}
	if _, exists := results["loop0"]; exists {
		t.Fatalf("expected unsupported drive to be omitted, got results=%+v", results)
	}
}

func TestSmartCheckAllDrivesSkipsFailedChecks(t *testing.T) {
	restore := resetDiskSeams()
	t.Cleanup(restore)

	checkSataDriveFn = func(string) (bool, error) {
		return false, errors.New("smart read failed")
	}
	checkNvmeDriveFn = func(string) (bool, error) {
		return true, nil
	}
	results := SmartCheckAllDrives(structs.LSBLKDevice{
		BlockDevices: []structs.BlockDev{
			{Name: "sda"},
			{Name: "nvme0n1"},
		},
	})
	if _, exists := results["sda"]; exists {
		t.Fatalf("expected failed sata check to be omitted, got %+v", results)
	}
	if results["nvme0n1"] != true {
		t.Fatalf("expected nvme result true, got %+v", results["nvme0n1"])
	}
}
