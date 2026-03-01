package handler

import (
	"strings"
	"testing"
	"time"

	"groundseg/broadcast"
	"groundseg/config"
	"groundseg/structs"
)

func setupUrbitOperationsConfig(t *testing.T, patp string, conf structs.UrbitDocker) {
	t.Helper()
	oldBasePath := config.BasePath
	oldUrbits := config.UrbitConfAll()

	config.BasePath = t.TempDir()
	config.UrbitsConfig = map[string]structs.UrbitDocker{
		patp: conf,
	}

	t.Cleanup(func() {
		config.BasePath = oldBasePath
		config.UrbitsConfig = oldUrbits
	})
}

func TestToggleAliasPersistsUpdatedDisplayMode(t *testing.T) {
	patp := "~zod"
	setupUrbitOperationsConfig(t, patp, structs.UrbitDocker{
		PierName:     patp,
		ShowUrbitWeb: "custom",
	})

	if err := toggleAlias(patp, config.UrbitConf(patp)); err != nil {
		t.Fatalf("toggleAlias returned error: %v", err)
	}
	if got := config.UrbitConf(patp).ShowUrbitWeb; got != "default" {
		t.Fatalf("expected alias mode default after toggle, got %q", got)
	}

	if err := toggleAlias(patp, config.UrbitConf(patp)); err != nil {
		t.Fatalf("toggleAlias second toggle returned error: %v", err)
	}
	if got := config.UrbitConf(patp).ShowUrbitWeb; got != "custom" {
		t.Fatalf("expected alias mode custom after second toggle, got %q", got)
	}
}

func TestStartramReminderPersistsConfigFlag(t *testing.T) {
	patp := "~bus"
	setupUrbitOperationsConfig(t, patp, structs.UrbitDocker{
		PierName:         patp,
		StartramReminder: false,
	})

	if err := startramReminder(patp, true); err != nil {
		t.Fatalf("startramReminder returned error: %v", err)
	}
	updated := config.UrbitConf(patp).StartramReminder
	remind, ok := updated.(bool)
	if !ok || !remind {
		t.Fatalf("expected startram reminder flag true, got %#v", updated)
	}
}

func TestSchedulePackValidatesInputsAndPersistsSchedule(t *testing.T) {
	patp := "~nec"
	setupUrbitOperationsConfig(t, patp, structs.UrbitDocker{
		PierName:      patp,
		MeldSchedule:  false,
		MeldFrequency: 0,
	})

	if err := schedulePack(patp, structs.WsUrbitPayload{
		Payload: structs.WsUrbitAction{
			Frequency:    0,
			IntervalType: "week",
		},
	}, config.UrbitConf(patp)); err == nil || !strings.Contains(err.Error(), "pack frequency cannot be 0") {
		t.Fatalf("expected frequency validation error, got %v", err)
	}

	if err := schedulePack(patp, structs.WsUrbitPayload{
		Payload: structs.WsUrbitAction{
			Frequency:    1,
			IntervalType: "year",
		},
	}, config.UrbitConf(patp)); err == nil || !strings.Contains(err.Error(), "unknown interval type") {
		t.Fatalf("expected interval type validation error, got %v", err)
	}

	reasonCh := make(chan string, 1)
	go func() {
		reasonCh <- <-broadcast.SchedulePackEvents()
	}()

	payload := structs.WsUrbitPayload{
		Payload: structs.WsUrbitAction{
			Frequency:    2,
			IntervalType: "week",
			Time:         "04:20",
			Day:          "monday",
			Date:         12,
		},
	}
	if err := schedulePack(patp, payload, config.UrbitConf(patp)); err != nil {
		t.Fatalf("schedulePack returned error: %v", err)
	}

	select {
	case reason := <-reasonCh:
		if reason != "schedule" {
			t.Fatalf("expected schedule publish reason, got %q", reason)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for schedule publish")
	}

	updated := config.UrbitConf(patp)
	if !updated.MeldSchedule || updated.MeldScheduleType != "week" || updated.MeldFrequency != 2 || updated.MeldTime != "04:20" || updated.MeldDay != "monday" || updated.MeldDate != 12 {
		t.Fatalf("unexpected persisted schedule: %+v", updated)
	}
}

func TestPausePackScheduleAndSetNewMaxPierSizePersistConfig(t *testing.T) {
	patp := "~rov"
	setupUrbitOperationsConfig(t, patp, structs.UrbitDocker{
		PierName:     patp,
		MeldSchedule: true,
		SizeLimit:    64,
	})

	if err := pausePackSchedule(patp, structs.WsUrbitPayload{}, config.UrbitConf(patp)); err != nil {
		t.Fatalf("pausePackSchedule returned error: %v", err)
	}
	if config.UrbitConf(patp).MeldSchedule {
		t.Fatal("expected meld schedule to be paused")
	}

	if err := setNewMaxPierSize(patp, structs.WsUrbitPayload{
		Payload: structs.WsUrbitAction{Value: 512},
	}, config.UrbitConf(patp)); err != nil {
		t.Fatalf("setNewMaxPierSize returned error: %v", err)
	}
	if got := config.UrbitConf(patp).SizeLimit; got != 512 {
		t.Fatalf("expected size limit 512, got %d", got)
	}
}

func TestToggleChopOnVereUpdateAndBootStatusPersist(t *testing.T) {
	patp := "~dop"
	setupUrbitOperationsConfig(t, patp, structs.UrbitDocker{
		PierName:      patp,
		ChopOnUpgrade: false,
		BootStatus:    "boot",
		ShowUrbitWeb:  "default",
	})

	if err := toggleChopOnVereUpdate(patp, config.UrbitConf(patp)); err != nil {
		t.Fatalf("toggleChopOnVereUpdate returned error: %v", err)
	}
	chopValue := config.UrbitConf(patp).ChopOnUpgrade
	enabled, ok := chopValue.(bool)
	if !ok || !enabled {
		t.Fatalf("expected chop-on-upgrade true, got %#v", chopValue)
	}

	if err := toggleBootStatus(patp, config.UrbitConf(patp)); err != nil {
		t.Fatalf("toggleBootStatus returned error: %v", err)
	}
	if got := config.UrbitConf(patp).BootStatus; got != "ignore" {
		t.Fatalf("expected boot status ignore, got %q", got)
	}
}
