package orchestration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"groundseg/structs"
)

func resetStatsSeams() {
	containers = make(map[string]structs.ContainerStats)
	getMemoryUsageForStats = getMemoryUsage
	getDiskUsageForStats = getDiskUsage
	nowForStats = time.Now
}

func TestForceUpdateContainerStatsStoresValues(t *testing.T) {
	t.Cleanup(resetStatsSeams)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	nowForStats = func() time.Time { return now }
	getMemoryUsageForStats = func(string) uint64 { return 123 }
	getDiskUsageForStats = func(string) int64 { return 456 }

	stats := ForceUpdateContainerStats("alpha")
	if stats.MemoryUsage != 123 || stats.DiskUsage != 456 || !stats.LastContact.Equal(now) {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if containers["alpha"].MemoryUsage != 123 {
		t.Fatalf("expected stats cache entry")
	}
}

func TestGetContainerStatsUsesCacheWhenFresh(t *testing.T) {
	t.Cleanup(resetStatsSeams)

	now := time.Now()
	containers["beta"] = structs.ContainerStats{
		LastContact: now.Add(-30 * time.Second),
		MemoryUsage: 11,
		DiskUsage:   22,
	}
	called := false
	getMemoryUsageForStats = func(string) uint64 { called = true; return 99 }
	getDiskUsageForStats = func(string) int64 { called = true; return 99 }

	stats := GetContainerStats("beta")
	if stats.MemoryUsage != 11 || stats.DiskUsage != 22 {
		t.Fatalf("expected cached stats, got %+v", stats)
	}
	if called {
		t.Fatalf("expected no refresh for fresh cache")
	}
}

func TestGetContainerStatsRefreshesAfterMinute(t *testing.T) {
	t.Cleanup(resetStatsSeams)

	now := time.Now()
	nowForStats = func() time.Time { return now }
	containers["gamma"] = structs.ContainerStats{
		LastContact: now.Add(-2 * time.Minute),
		MemoryUsage: 1,
		DiskUsage:   2,
	}
	getMemoryUsageForStats = func(string) uint64 { return 1000 }
	getDiskUsageForStats = func(string) int64 { return 2000 }

	stats := GetContainerStats("gamma")
	if stats.MemoryUsage != 1000 || stats.DiskUsage != 2000 || !stats.LastContact.Equal(now) {
		t.Fatalf("expected refreshed stats, got %+v", stats)
	}
}

func TestGetContainerStatsMissCallsForceUpdatePath(t *testing.T) {
	t.Cleanup(resetStatsSeams)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	nowForStats = func() time.Time { return now }
	getMemoryUsageForStats = func(string) uint64 { return 7 }
	getDiskUsageForStats = func(string) int64 { return 8 }

	stats := GetContainerStats("delta")
	if stats.MemoryUsage != 7 || stats.DiskUsage != 8 {
		t.Fatalf("unexpected stats from miss path: %+v", stats)
	}
}

func TestGetDirSizeSumsFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("abc"), 0o644); err != nil {
		t.Fatalf("write a.txt failed: %v", err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir sub failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("12345"), 0o644); err != nil {
		t.Fatalf("write b.txt failed: %v", err)
	}

	size, err := getDirSize(dir)
	if err != nil {
		t.Fatalf("getDirSize failed: %v", err)
	}
	if size != 8 {
		t.Fatalf("expected file size sum 8, got %d", size)
	}
}
