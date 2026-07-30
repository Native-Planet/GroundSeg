package routines

import (
	"testing"
	"time"
)

func TestReserveScheduledPackPreventsDuplicateUntilRelease(t *testing.T) {
	scheduledPackState.Lock()
	scheduledPackState.active = make(map[string]time.Time)
	scheduledPackState.Unlock()

	when := time.Unix(100, 0)
	if !reserveScheduledPack("zod", when) {
		t.Fatal("expected first reservation to succeed")
	}
	if reserveScheduledPack("zod", when.Add(time.Minute)) {
		t.Fatal("expected duplicate reservation to be rejected")
	}

	releaseScheduledPack("zod")
	if !reserveScheduledPack("zod", when.Add(time.Minute)) {
		t.Fatal("expected reservation after release to succeed")
	}
	releaseScheduledPack("zod")
}

func TestSetWeekScheduleSameDayAfterScheduledTimeUsesNextCycle(t *testing.T) {
	location := time.FixedZone("test", -5*60*60)
	last := time.Date(2026, time.July, 29, 18, 30, 3, 0, location)

	next, err := setWeekSchedule(last, 1, "wednesday", "1830")
	if err != nil {
		t.Fatal(err)
	}

	want := time.Date(2026, time.August, 5, 18, 30, 0, 0, location)
	if !next.Equal(want) {
		t.Fatalf("next weekly pack = %v, want %v", next, want)
	}
}

func TestSetWeekScheduleSameDayBeforeScheduledTimeUsesCurrentCycle(t *testing.T) {
	location := time.FixedZone("test", -5*60*60)
	last := time.Date(2026, time.July, 29, 12, 0, 0, 0, location)

	next, err := setWeekSchedule(last, 1, "wednesday", "1830")
	if err != nil {
		t.Fatal(err)
	}

	want := time.Date(2026, time.July, 29, 18, 30, 0, 0, location)
	if !next.Equal(want) {
		t.Fatalf("next weekly pack = %v, want %v", next, want)
	}
}

func TestSetWeekScheduleHonorsMultiWeekFrequency(t *testing.T) {
	location := time.FixedZone("test", -5*60*60)
	last := time.Date(2026, time.July, 29, 18, 30, 3, 0, location)

	next, err := setWeekSchedule(last, 2, "wednesday", "1830")
	if err != nil {
		t.Fatal(err)
	}

	want := time.Date(2026, time.August, 12, 18, 30, 0, 0, location)
	if !next.Equal(want) {
		t.Fatalf("next two-week pack = %v, want %v", next, want)
	}
}

func TestSkipMissedPackCyclesKeepsScheduleWithinGrace(t *testing.T) {
	now := time.Date(2026, time.July, 29, 18, 30, 59, 0, time.UTC)
	scheduled := time.Date(2026, time.July, 29, 18, 30, 0, 0, time.UTC)

	next, skipped, err := skipMissedPackCycles(scheduled, now, "week", 1)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped cycles = %d, want 0", skipped)
	}
	if !next.Equal(scheduled) {
		t.Fatalf("next pack = %v, want unchanged schedule %v", next, scheduled)
	}
}

func TestSkipMissedPackCyclesAdvancesOverdueSchedule(t *testing.T) {
	now := time.Date(2026, time.July, 29, 18, 31, 1, 0, time.UTC)
	scheduled := time.Date(2026, time.July, 29, 18, 30, 0, 0, time.UTC)

	next, skipped, err := skipMissedPackCycles(scheduled, now, "week", 1)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 1 {
		t.Fatalf("skipped cycles = %d, want 1", skipped)
	}
	want := time.Date(2026, time.August, 5, 18, 30, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next pack = %v, want %v", next, want)
	}
}

func TestSkipMissedPackCyclesSkipsEveryStaleCycle(t *testing.T) {
	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	scheduled := time.Date(2026, time.July, 27, 18, 30, 0, 0, time.UTC)

	next, skipped, err := skipMissedPackCycles(scheduled, now, "day", 1)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 3 {
		t.Fatalf("skipped cycles = %d, want 3", skipped)
	}
	want := time.Date(2026, time.July, 30, 18, 30, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next pack = %v, want %v", next, want)
	}
}
