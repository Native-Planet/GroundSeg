package notify

import (
	"errors"
	"strings"
	"testing"

	"groundseg/click/lifecycle"
	"groundseg/structs"
)

func resetNotifySeams() {
	sendStartramReminderFn = sendStartramReminder
	sendDiskSpaceWarningFn = sendDiskSpaceWarning
	sendSmartWarningFn = sendSmartWarning
	executeClickCommandForHark = executeClickCommandForHarkDefault
}

func TestSendNotificationDispatch(t *testing.T) {
	t.Cleanup(resetNotifySeams)

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

func TestSendStartramReminderBuildsPayload(t *testing.T) {
	t.Cleanup(resetNotifySeams)
	var gotFile, gotHoon string
	executeClickCommandForHark = func(_patp, file, body, _sourcePath, _successToken, _operation string, _clear func(string)) (string, error) {
		gotFile = file
		gotHoon = body
		return "ok", nil
	}

	if err := sendStartramReminder("~zod", 4); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotFile != "startram-hark" {
		t.Fatalf("unexpected file: %s", gotFile)
	}
	if !strings.Contains(gotHoon, "170141184506683847949839018055058849792") {
		t.Fatalf("expected static notification note id in hoon payload: %s", gotHoon)
	}
}

func TestSendDiskSpaceWarningBuildsPayload(t *testing.T) {
	t.Cleanup(resetNotifySeams)
	var gotFile, gotHoon string
	executeClickCommandForHark = func(_patp, file, body, _sourcePath, _successToken, _operation string, _clear func(string)) (string, error) {
		gotFile = file
		gotHoon = body
		return "ok", nil
	}

	if err := sendDiskSpaceWarning("~bus", "nvme0n1", 92.5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotFile != "diskspace-hark" {
		t.Fatalf("unexpected file: %s", gotFile)
	}
	if !strings.Contains(gotHoon, "nvme0n1") || !strings.Contains(gotHoon, "92.5") {
		t.Fatalf("expected disk details in hoon payload: %s", gotHoon)
	}
}

func TestSendSmartWarningExecFailure(t *testing.T) {
	t.Cleanup(resetNotifySeams)
	executeClickCommandForHark = func(_, _, _, _, _, _ string, _clear func(string)) (string, error) {
		return "", errors.New("exec failed")
	}

	err := sendSmartWarning("~nec", "sda")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "failed to execute hoon") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendStartramReminderPokeFailure(t *testing.T) {
	t.Cleanup(resetNotifySeams)
	executeClickCommandForHark = func(_, _, _, _, _, _ string, _clear func(string)) (string, error) {
		return "", errors.New("failed poke")
	}
	lifecycle.BarExit("~zod") // keep package import alive for compile in test package
	err := sendStartramReminder("~pal", 2)
	if err == nil {
		t.Fatalf("expected error")
	}
	if err == nil || !strings.Contains(err.Error(), "failed poke") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func executeClickCommandForHarkDefault(_patp, _file, _body, _sourcePath, _successToken, _operation string, _clear func(string)) (string, error) {
	return "ok", nil
}
