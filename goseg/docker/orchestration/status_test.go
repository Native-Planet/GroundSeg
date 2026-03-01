package orchestration

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func resetStatusSeams() {
	getShipStatusForWait = GetShipStatus
	nowForShipExit = time.Now
	sleepForShipExit = time.Sleep
	shipExitPollInterval = 1 * time.Second
}

func TestWaitForShipExitReturnsWhenShipStops(t *testing.T) {
	t.Cleanup(resetStatusSeams)

	calls := 0
	getShipStatusForWait = func([]string) (map[string]string, error) {
		calls++
		if calls == 1 {
			return map[string]string{"~zod": "Up 2 seconds"}, nil
		}
		return map[string]string{"~zod": "Exited (0)"}, nil
	}
	sleepForShipExit = func(time.Duration) {}
	nowForShipExit = func() time.Time { return time.Now() }

	if err := WaitForShipExit("~zod", 5*time.Second); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestWaitForShipExitStatusErrors(t *testing.T) {
	t.Cleanup(resetStatusSeams)

	getShipStatusForWait = func([]string) (map[string]string, error) {
		return nil, errors.New("boom")
	}
	if err := WaitForShipExit("~zod", time.Second); err == nil || !strings.Contains(err.Error(), "failed to get statuses") {
		t.Fatalf("expected status fetch error, got %v", err)
	}

	getShipStatusForWait = func([]string) (map[string]string, error) {
		return map[string]string{"~bus": "Up"}, nil
	}
	if err := WaitForShipExit("~zod", time.Second); err == nil || !strings.Contains(err.Error(), "status doesn't exist") {
		t.Fatalf("expected missing status error, got %v", err)
	}
}

func TestWaitForShipExitTimeout(t *testing.T) {
	t.Cleanup(resetStatusSeams)
	getShipStatusForWait = func([]string) (map[string]string, error) {
		return map[string]string{"~zod": "Up 99 seconds"}, nil
	}
	sleepForShipExit = func(time.Duration) {}

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	calls := 0
	nowForShipExit = func() time.Time {
		calls++
		if calls == 1 {
			return start
		}
		return start.Add(10 * time.Second)
	}

	err := WaitForShipExit("~zod", time.Second)
	if err == nil || !strings.Contains(err.Error(), "timed out waiting for ~zod") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}
