package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func dirEntriesToFileInfos(t *testing.T, entries []os.DirEntry) []os.FileInfo {
	t.Helper()
	infos := make([]os.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			t.Fatalf("failed to read file info for %s: %v", entry.Name(), err)
		}
		infos = append(infos, info)
	}
	return infos
}

func TestMostRecentPartedLogPathsSelectsMostRecentByDateAndPart(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	files := []string{
		"2026-02-part-0.log",
		"2026-01-part-9.log",
		"invalid.log",
		"2026-02-part-2.log",
		"2026-02-part-1.log",
		"2026-01-part-10.log",
		"2026-03-part-0.log",
	}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("failed to create fixture %s: %v", name, err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read fixtures: %v", err)
	}
	fileInfos := dirEntriesToFileInfos(t, entries)

	got := MostRecentPartedLogPaths(dir, fileInfos, 3)
	want := []string{
		filepath.Join(dir, "2026-03-part-0.log"),
		filepath.Join(dir, "2026-02-part-2.log"),
		filepath.Join(dir, "2026-02-part-1.log"),
	}
	if len(got) != len(want) {
		t.Fatalf("unexpected results count: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected file at position %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestMostRecentPartedLogPathsHonorsKeepLimit(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		name := filepath.Join(dir, "2026-04-part-"+string(rune('0'+i))+".log")
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatalf("failed to create part %d fixture: %v", i, err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read fixtures: %v", err)
	}
	fileInfos := dirEntriesToFileInfos(t, entries)

	got := MostRecentPartedLogPaths(dir, fileInfos, 0)
	if got != nil {
		t.Fatalf("expected nil when keep is zero, got %v", got)
	}
}
