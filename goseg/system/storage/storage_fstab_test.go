package storage

import (
	"os"
	"path/filepath"
	"testing"

	"groundseg/testutil/disktest"
)

func TestParseFstabLineContractMatrix(t *testing.T) {
	t.Parallel()

	disktest.RunFstabLineParseMatrix(t, []disktest.ParseBoolLineCase{
		{
			Name:      "ignores comment line",
			Input:     " # comment",
			ParseFn:   func(raw string) bool { _, ok := ParseFstabLine(raw); return ok },
			ShouldHit: false,
		},
		{
			Name:      "ignores malformed line",
			Input:     "too-short  ",
			ParseFn:   func(raw string) bool { _, ok := ParseFstabLine(raw); return ok },
			ShouldHit: false,
		},
	})
}

func TestReconcileFstabLinesContractMatrix(t *testing.T) {
	t.Parallel()

	disktest.RunReconcileFstabMatrix(t, []disktest.ReconcileFstabCase{
		{
			Name: "normalizes options and adds no duplicates",
			Run: func() ([]string, bool) {
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
				return ReconcileFstabLines(recorded, record)
			},
			Assert: func(t *testing.T, reconciled []string, changed bool) {
				if !changed {
					t.Fatal("expected reconcile to change mount options")
				}
				if len(reconciled) != 2 {
					t.Fatalf("expected 2 entries, got %d", len(reconciled))
				}
				if reconciled[0] != "UUID=abc /groundseg-1 ext4 defaults,nofail 0 2" {
					t.Fatalf("expected normalized entry, got %q", reconciled[0])
				}
			},
		},
		{
			Name: "normalizes existing record and is idempotent after normalization",
			Run: func() ([]string, bool) {
				record := FstabRecord{
					Device:     "UUID=abc",
					MountPoint: "/groundseg-1",
					FSType:     "ext4",
					Options:    "defaults,nofail",
					Dump:       "0",
					Pass:       "2",
				}
				recorded := []string{
					"UUID=abc /groundseg-1 ext4 defaults,nofail 0 2",
					"tmpfs /tmp tmpfs defaults 0 0",
				}
				return ReconcileFstabLines(recorded, record)
			},
			Assert: func(t *testing.T, reconciled []string, changed bool) {
				if changed {
					t.Fatal("expected idempotent reconcile after normalization")
				}
			},
		},
	})
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
