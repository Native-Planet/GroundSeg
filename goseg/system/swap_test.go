package system

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/shirou/gopsutil/disk"
)

func resetSwapSeamsForTest(t *testing.T) {
	t.Helper()
	origExec := execCommandForSwap
	origStat := osStatForSwap
	origRemove := osRemoveForSwap
	origStart := startSwapForConfigure
	origStop := stopSwapForConfigure
	origMake := makeSwapForConfigure
	origActive := activeSwapForConfigure
	origDiskUsage := diskUsageForSwap
	origCap := cap
	t.Cleanup(func() {
		execCommandForSwap = origExec
		osStatForSwap = origStat
		osRemoveForSwap = origRemove
		startSwapForConfigure = origStart
		stopSwapForConfigure = origStop
		makeSwapForConfigure = origMake
		activeSwapForConfigure = origActive
		diskUsageForSwap = origDiskUsage
		cap = origCap
	})
}

func TestConfigureSwapValidatesAndStopsOnZero(t *testing.T) {
	resetSwapSeamsForTest(t)

	if err := ConfigureSwap("/swapfile", -1); err == nil {
		t.Fatal("expected negative swap value validation error")
	}

	stopped := false
	stopSwapForConfigure = func(loc string) error {
		stopped = loc == "/swapfile"
		return nil
	}
	if err := ConfigureSwap("/swapfile", 0); err != nil {
		t.Fatalf("ConfigureSwap(0) returned error: %v", err)
	}
	if !stopped {
		t.Fatal("expected zero swap to call stopSwap")
	}
}

func TestConfigureSwapCreatesStartsAndRebuildsWhenSizeMismatches(t *testing.T) {
	resetSwapSeamsForTest(t)

	osStatForSwap = func(name string) (os.FileInfo, error) {
		if name == "/missing" {
			return nil, os.ErrNotExist
		}
		return nil, nil
	}
	makeCalls := 0
	makeSwapForConfigure = func(loc string, val int) error {
		makeCalls++
		if val != 4 {
			t.Fatalf("unexpected makeSwap value: %d", val)
		}
		return nil
	}
	startCalls := 0
	startSwapForConfigure = func(string) error {
		startCalls++
		return nil
	}
	stopCalls := 0
	stopSwapForConfigure = func(string) error {
		stopCalls++
		return nil
	}
	activeSwapForConfigure = func(loc string) int {
		if loc == "/missing" {
			return 4
		}
		return 2
	}
	removeCalls := 0
	osRemoveForSwap = func(string) error {
		removeCalls++
		return nil
	}

	if err := ConfigureSwap("/missing", 4); err != nil {
		t.Fatalf("ConfigureSwap(missing) returned error: %v", err)
	}
	if makeCalls != 1 || startCalls != 1 || stopCalls != 0 || removeCalls != 0 {
		t.Fatalf("unexpected call counts for missing file: make=%d start=%d stop=%d remove=%d", makeCalls, startCalls, stopCalls, removeCalls)
	}

	makeCalls, startCalls, stopCalls, removeCalls = 0, 0, 0, 0
	if err := ConfigureSwap("/exists", 4); err != nil {
		t.Fatalf("ConfigureSwap(existing mismatch) returned error: %v", err)
	}
	if makeCalls != 1 || startCalls != 2 || stopCalls != 1 || removeCalls != 1 {
		t.Fatalf("unexpected rebuild call counts: make=%d start=%d stop=%d remove=%d", makeCalls, startCalls, stopCalls, removeCalls)
	}
}

func TestActiveSwapParsesSizesFromSwaponOutput(t *testing.T) {
	resetSwapSeamsForTest(t)

	execCommandForSwap = func(name string, args ...string) *exec.Cmd {
		if name != "swapon" || strings.Join(args, " ") != "--show" {
			t.Fatalf("unexpected command invocation: %s %v", name, args)
		}
		return exec.Command("sh", "-c", "printf '/swapfile file 1G 0B -2\\n/other file 512M 0B -2\\n'")
	}
	if size := ActiveSwap("/swapfile"); size != 1 {
		t.Fatalf("expected 1G parsed to 1, got %d", size)
	}

	execCommandForSwap = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf '/swapfile file 2048M 0B -2\\n'")
	}
	if size := ActiveSwap("/swapfile"); size != 2 {
		t.Fatalf("expected 2048M parsed to 2G, got %d", size)
	}
}

func TestMaxSwapUsesDiskFreeSpaceAndCap(t *testing.T) {
	resetSwapSeamsForTest(t)

	diskUsageForSwap = func(path string) (*disk.UsageStat, error) {
		switch path {
		case "/err":
			return nil, errors.New("disk lookup failed")
		case "/large":
			return &disk.UsageStat{Free: uint64(100 * 1024 * 1024 * 1024)}, nil
		default:
			return &disk.UsageStat{Free: uint64(5 * 1024 * 1024 * 1024)}, nil
		}
	}
	cap = 32

	if got := MaxSwap("/err", 0); got != 0 {
		t.Fatalf("expected error path to return 0, got %d", got)
	}
	if got := MaxSwap("/small", 0); got != 5 {
		t.Fatalf("expected free-space-based max swap 5, got %d", got)
	}
	if got := MaxSwap("/large", 0); got != 32 {
		t.Fatalf("expected capped max swap 32, got %d", got)
	}
}

func TestStartSwapNoopsWhenAlreadyActive(t *testing.T) {
	resetSwapSeamsForTest(t)

	calls := []string{}
	execCommandForSwap = func(name string, args ...string) *exec.Cmd {
		calls = append(calls, fmt.Sprintf("%s %s", name, strings.Join(args, " ")))
		if name == "swapon" && strings.Join(args, " ") == "--show" {
			return exec.Command("sh", "-c", "printf '/swapfile file 1G 0B -2\\n'")
		}
		return exec.Command("true")
	}

	if err := startSwap("/swapfile"); err != nil {
		t.Fatalf("startSwap returned error: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected only swapon --show call when already active, got %v", calls)
	}
}
