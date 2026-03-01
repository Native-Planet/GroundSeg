package click

import (
	"errors"
	"strings"
	"testing"
)

func resetHarkSeams() {
	createHoonForHark = createHoon
	deleteHoonForHark = deleteHoon
	clickExecForHark = clickExec
	filterResponseForHark = filterResponse
}

func TestSendStartramReminderCreateFailure(t *testing.T) {
	t.Cleanup(resetHarkSeams)
	createHoonForHark = func(_, _, _ string) error {
		return errors.New("write failed")
	}

	err := sendStartramReminder("~zod", 4)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "failed to create hoon") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendDiskSpaceWarningBuildsPayload(t *testing.T) {
	t.Cleanup(resetHarkSeams)

	var fileName, hoon string
	createHoonForHark = func(_, file, body string) error {
		fileName = file
		hoon = body
		return nil
	}
	deleteHoonForHark = func(_, _ string) {}
	clickExecForHark = func(_, _, _ string) (string, error) { return "ok", nil }
	filterResponseForHark = func(_, _ string) (string, bool, error) {
		return "", true, nil
	}

	if err := sendDiskSpaceWarning("~bus", "nvme0n1", 92.5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fileName != "diskspace-hark" {
		t.Fatalf("unexpected file: %s", fileName)
	}
	if !strings.Contains(hoon, "nvme0n1") || !strings.Contains(hoon, "92.5") {
		t.Fatalf("expected disk details in hoon payload: %s", hoon)
	}
}

func TestSendSmartWarningExecFailure(t *testing.T) {
	t.Cleanup(resetHarkSeams)
	createHoonForHark = func(_, _, _ string) error { return nil }
	deleteHoonForHark = func(_, _ string) {}
	clickExecForHark = func(_, _, _ string) (string, error) {
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
	t.Cleanup(resetHarkSeams)
	createHoonForHark = func(_, _, _ string) error { return nil }
	deleteHoonForHark = func(_, _ string) {}
	clickExecForHark = func(_, _, _ string) (string, error) { return "resp", nil }
	filterResponseForHark = func(_, _ string) (string, bool, error) {
		return "", false, nil
	}

	err := sendStartramReminder("~pal", 2)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "failed poke") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendSmartWarningSuccess(t *testing.T) {
	t.Cleanup(resetHarkSeams)
	createHoonForHark = func(_, _, _ string) error { return nil }
	deleteHoonForHark = func(_, _ string) {}
	clickExecForHark = func(_, _, _ string) (string, error) { return "resp", nil }
	filterResponseForHark = func(_, _ string) (string, bool, error) {
		return "", true, nil
	}

	if err := sendSmartWarning("~mar", "disk2"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
