package click

import (
	"errors"
	"strings"
	"testing"

	backupdomain "groundseg/click/backup"
	"groundseg/structs"
)

func resetClickSeams() {
	sendStartramReminderFn = sendStartramReminder
	sendDiskSpaceWarningFn = sendDiskSpaceWarning
	sendSmartWarningFn = sendSmartWarning
	restoreAgentFn = restoreAgent
	backupTlonFn = backupdomain.BackupTlon
}

func TestSendNotificationDispatch(t *testing.T) {
	t.Cleanup(resetClickSeams)

	var called string
	sendStartramReminderFn = func(patp string, days int) error {
		called = "startram:" + patp
		if days != 5 {
			t.Fatalf("unexpected days: %d", days)
		}
		return nil
	}
	sendDiskSpaceWarningFn = func(patp, disk string, usage float64) error {
		called = "disk:" + patp + ":" + disk
		if usage != 91 {
			t.Fatalf("unexpected usage: %v", usage)
		}
		return nil
	}
	sendSmartWarningFn = func(patp, disk string) error {
		called = "smart:" + patp + ":" + disk
		return nil
	}

	err := SendNotification("~zod", structs.HarkNotification{Type: "startram-reminder", StartramDaysLeft: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != "startram:~zod" {
		t.Fatalf("unexpected dispatch: %s", called)
	}

	err = SendNotification("~bus", structs.HarkNotification{Type: "disk-warning", DiskName: "disk0", DiskUsage: 91})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != "disk:~bus:disk0" {
		t.Fatalf("unexpected dispatch: %s", called)
	}

	err = SendNotification("~nec", structs.HarkNotification{Type: "smart-fail", DiskName: "nvme0n1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != "smart:~nec:nvme0n1" {
		t.Fatalf("unexpected dispatch: %s", called)
	}
}

func TestSendNotificationInvalidType(t *testing.T) {
	err := SendNotification("~zod", structs.HarkNotification{Type: "unknown"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "invalid hark notification type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackupTlonRunsAllAgents(t *testing.T) {
	t.Cleanup(resetClickSeams)
	called := false
	backupTlonFn = func(patp string) error {
		called = patp == "~zod"
		return nil
	}

	if err := BackupTlon("~zod"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected backup wrapper to call delegated service")
	}
}

func TestBackupTlonAggregatesErrors(t *testing.T) {
	t.Cleanup(resetClickSeams)
	backupTlonFn = func(string) error {
		return errors.New("boom")
	}

	err := BackupTlon("~zod")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRestoreTlonAggregatesErrors(t *testing.T) {
	t.Cleanup(resetClickSeams)
	restoreAgentFn = func(_ string, agent string) error {
		if agent == "chat" {
			return errors.New("restore failed")
		}
		return nil
	}

	err := RestoreTlon("~zod")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "chat: restore failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
