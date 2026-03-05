package disk

import (
	"os"
	"path/filepath"
	"testing"

	"groundseg/structs"
)

func TestParseAndReconcileFstabLines(t *testing.T) {
	t.Parallel()

	record, ok := ParseFstabLine("UUID=abc123 /groundseg-1 ext4 defaults,nofail 0 2")
	if !ok {
		t.Fatal("expected fstab line to parse")
	}
	if record.Device != "UUID=abc123" || record.MountPoint != "/groundseg-1" {
		t.Fatalf("unexpected parsed record: %+v", record)
	}

	lines, changed := ReconcileFstabLines([]string{
		"UUID=abc123 /groundseg-1 ext4 defaults 0 0",
		"tmpfs /run tmpfs defaults 0 0",
	}, record)
	if !changed {
		t.Fatal("expected reconcile to detect changed mount options")
	}
	if len(lines) != 2 {
		t.Fatalf("unexpected reconciled line count: %d", len(lines))
	}
}

func TestReadWriteFstabLinesRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "fstab")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatalf("create fstab file: %v", err)
	}
	original := []string{
		"UUID=abc123 /groundseg-1 ext4 defaults,nofail 0 2",
		"tmpfs /run tmpfs defaults 0 0",
	}
	if err := WriteFstabLines(path, original); err != nil {
		t.Fatalf("WriteFstabLines returned error: %v", err)
	}

	got, err := ReadFstabLines(path)
	if err != nil {
		t.Fatalf("ReadFstabLines returned error: %v", err)
	}
	if len(got) != len(original) {
		t.Fatalf("unexpected number of lines: got %d want %d", len(got), len(original))
	}
	for i := range original {
		if got[i] != original[i] {
			t.Fatalf("unexpected line at index %d: got %q want %q", i, got[i], original[i])
		}
	}
}

func TestSmartResultsSnapshotRoundTrip(t *testing.T) {
	t.Parallel()

	previous := SmartResultsSnapshot()
	t.Cleanup(func() {
		SetSmartResults(previous)
	})

	SetSmartResults(map[string]bool{
		"sda":     true,
		"nvme0n1": false,
	})
	got := SmartResultsSnapshot()
	if got["sda"] != true {
		t.Fatalf("expected smart result for sda to be true, got %v", got["sda"])
	}
	if got["nvme0n1"] != false {
		t.Fatalf("expected smart result for nvme0n1 to be false, got %v", got["nvme0n1"])
	}
}

func TestIsDevMountedAndSmartCheckAllDrivesEmpty(t *testing.T) {
	t.Parallel()

	if IsDevMounted(structs.BlockDev{}) {
		t.Fatal("expected zero-value block device not to be mounted")
	}
	results := SmartCheckAllDrives(structs.LSBLKDevice{})
	if len(results) != 0 {
		t.Fatalf("expected empty smart-check result for empty device list, got %+v", results)
	}
}

func TestRemoveMultipartFilesRemovesMultipartPrefix(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	multipartPath := filepath.Join(dir, "multipart-001")
	keepPath := filepath.Join(dir, "keep.txt")
	if err := os.WriteFile(multipartPath, []byte("temp"), 0o644); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := os.WriteFile(keepPath, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write keep file: %v", err)
	}

	if err := RemoveMultipartFiles(dir); err != nil {
		t.Fatalf("RemoveMultipartFiles returned error: %v", err)
	}
	if _, err := os.Stat(multipartPath); !os.IsNotExist(err) {
		t.Fatalf("expected multipart file to be removed, err=%v", err)
	}
	if _, err := os.Stat(keepPath); err != nil {
		t.Fatalf("expected non-multipart file to remain, err=%v", err)
	}
}
