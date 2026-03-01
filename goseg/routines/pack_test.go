package routines

import (
	"strings"
	"testing"
	"time"
)

func TestConvertMeldTimeParsesValidHHMM(t *testing.T) {
	hour, minute, err := convertMeldTime("0935")
	if err != nil {
		t.Fatalf("convertMeldTime returned error: %v", err)
	}
	if hour != 9 || minute != 35 {
		t.Fatalf("unexpected parsed values: hour=%d minute=%d", hour, minute)
	}
}

func TestConvertMeldTimeRejectsInvalidHour(t *testing.T) {
	_, _, err := convertMeldTime("ab35")
	if err == nil {
		t.Fatal("expected error for invalid hour")
	}
	if !strings.Contains(err.Error(), "Invalid hour") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConvertMeldTimeRejectsInvalidMinute(t *testing.T) {
	_, _, err := convertMeldTime("09ab")
	if err == nil {
		t.Fatal("expected error for invalid minute")
	}
	if !strings.Contains(err.Error(), "Invalid minute") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetDayScheduleAdvancesByFrequencyAndAppliesTime(t *testing.T) {
	last := time.Date(2026, time.March, 10, 1, 0, 0, 0, time.UTC)
	next, err := setDaySchedule(last, 2, "0930")
	if err != nil {
		t.Fatalf("setDaySchedule returned error: %v", err)
	}
	want := time.Date(2026, time.March, 12, 9, 30, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("unexpected next day schedule: got %v want %v", next, want)
	}
}

func TestSetWeekScheduleTargetsRequestedWeekday(t *testing.T) {
	last := time.Date(2026, time.March, 3, 12, 0, 0, 0, time.UTC) // Tuesday
	next, err := setWeekSchedule(last, 1, "friday", "1015")
	if err != nil {
		t.Fatalf("setWeekSchedule returned error: %v", err)
	}
	want := time.Date(2026, time.March, 6, 10, 15, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("unexpected next week schedule: got %v want %v", next, want)
	}
}

func TestSetWeekScheduleSameDayWithFrequencyTwoMovesOneWeek(t *testing.T) {
	last := time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC) // Monday
	next, err := setWeekSchedule(last, 2, "monday", "0700")
	if err != nil {
		t.Fatalf("setWeekSchedule returned error: %v", err)
	}
	want := time.Date(2026, time.March, 9, 7, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("unexpected same-day weekly schedule: got %v want %v", next, want)
	}
}

func TestSetWeekScheduleRejectsInvalidWeekday(t *testing.T) {
	last := time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC)
	next, err := setWeekSchedule(last, 1, "funday", "0700")
	if err == nil {
		t.Fatal("expected invalid weekday error")
	}
	if !next.Equal(last) {
		t.Fatalf("expected invalid weekday to return original time, got %v want %v", next, last)
	}
}

func TestSetMonthScheduleUsesCurrentMonthWhenDateNotElapsed(t *testing.T) {
	last := time.Date(2026, time.March, 1, 8, 0, 0, 0, time.UTC)
	next, err := setMonthSchedule(last, 1, 15, "1200")
	if err != nil {
		t.Fatalf("setMonthSchedule returned error: %v", err)
	}
	want := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("unexpected monthly schedule: got %v want %v", next, want)
	}
}

func TestSetMonthScheduleAdvancesFrequencyWhenDateElapsed(t *testing.T) {
	last := time.Date(2026, time.March, 20, 8, 0, 0, 0, time.UTC)
	next, err := setMonthSchedule(last, 2, 15, "1200")
	if err != nil {
		t.Fatalf("setMonthSchedule returned error: %v", err)
	}
	want := time.Date(2026, time.May, 15, 12, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("unexpected monthly rollover schedule: got %v want %v", next, want)
	}
}
