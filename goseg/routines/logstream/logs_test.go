package logstream

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"groundseg/logger"
)

func withLoggerLogPath(t *testing.T, logPath string) {
	t.Helper()
	original := logger.LogPath
	logger.LogPath = logPath
	t.Cleanup(func() {
		logger.LogPath = original
	})
}

func TestSplitLogFileCreatesSingleChunkForSmallInput(t *testing.T) {
	outputDir := t.TempDir()
	withLoggerLogPath(t, outputDir+string(os.PathSeparator))

	inputDir := t.TempDir()
	inputFile := filepath.Join(inputDir, "2026-03.log")
	content := "first line\nsecond line\n"
	if err := os.WriteFile(inputFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create input log: %v", err)
	}

	if err := splitLogFile(inputFile); err != nil {
		t.Fatalf("splitLogFile returned error: %v", err)
	}

	part0 := filepath.Join(outputDir, "2026-03-part-0.log")
	data, err := os.ReadFile(part0)
	if err != nil {
		t.Fatalf("failed reading split part-0: %v", err)
	}
	if string(data) != content {
		t.Fatalf("unexpected part-0 contents: %q", string(data))
	}
	if _, err := os.Stat(filepath.Join(outputDir, "2026-03-part-1.log")); !os.IsNotExist(err) {
		t.Fatalf("expected no part-1 file, stat err=%v", err)
	}
}

func TestSplitLogFileCreatesMultipleChunksWhenExceedingMaxChunkSize(t *testing.T) {
	outputDir := t.TempDir()
	withLoggerLogPath(t, outputDir+string(os.PathSeparator))

	inputDir := t.TempDir()
	inputFile := filepath.Join(inputDir, "2026-04.log")
	line := strings.Repeat("a", 6*1024*1024-1) + "\n"
	content := line + line
	if err := os.WriteFile(inputFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create large input log: %v", err)
	}

	if err := splitLogFile(inputFile); err != nil {
		t.Fatalf("splitLogFile returned error: %v", err)
	}

	part0 := filepath.Join(outputDir, "2026-04-part-0.log")
	part1 := filepath.Join(outputDir, "2026-04-part-1.log")
	if _, err := os.Stat(part0); err != nil {
		t.Fatalf("expected part-0 to exist: %v", err)
	}
	if _, err := os.Stat(part1); err != nil {
		t.Fatalf("expected part-1 to exist: %v", err)
	}
}

func TestKeepMostRecentFilesRetainsLatestTenByDateAndPart(t *testing.T) {
	dir := t.TempDir()
	for part := 0; part < 12; part++ {
		name := fmt.Sprintf("2026-01-part-%d.log", part)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("failed to create %s: %v", name, err)
		}
	}

	if err := keepMostRecentFiles(dir); err != nil {
		t.Fatalf("keepMostRecentFiles returned error: %v", err)
	}

	for part := 11; part >= 2; part-- {
		name := filepath.Join(dir, fmt.Sprintf("2026-01-part-%d.log", part))
		if _, err := os.Stat(name); err != nil {
			t.Fatalf("expected %s to remain: %v", name, err)
		}
	}
	for _, removedPart := range []int{0, 1} {
		name := filepath.Join(dir, fmt.Sprintf("2026-01-part-%d.log", removedPart))
		if _, err := os.Stat(name); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed, stat err=%v", name, err)
		}
	}
}

func TestLogContains(t *testing.T) {
	slice := []string{"a", "b", "c"}
	if !logContains(slice, "b") {
		t.Fatal("expected slice to contain b")
	}
	if logContains(slice, "z") {
		t.Fatal("did not expect slice to contain z")
	}
}
