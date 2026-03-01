package routines

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"groundseg/structs"
)

func resetDiskSeamsForTest(t *testing.T) {
	t.Helper()
	originalUpdate := updateDiskWarningInConf
	originalNotify := sendNotificationForDiskWarning
	t.Cleanup(func() {
		updateDiskWarningInConf = originalUpdate
		sendNotificationForDiskWarning = originalNotify
	})
}

func TestSetWarningInfoInitializesAndPersistsDiskWarning(t *testing.T) {
	resetDiskSeamsForTest(t)

	captured := map[string]structs.DiskWarning{}
	updateDiskWarningInConf = func(warnings map[string]structs.DiskWarning) error {
		captured = warnings
		return nil
	}

	ninetyFive := time.Unix(1_700_000_000, 0)
	if err := setWarningInfo(structs.SysConfig{}, "disk0", true, false, ninetyFive); err != nil {
		t.Fatalf("setWarningInfo returned error: %v", err)
	}

	want := structs.DiskWarning{Eighty: true, Ninety: false, NinetyFive: ninetyFive}
	if got, exists := captured["disk0"]; !exists || !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected captured warning map: %+v", captured)
	}
}

func TestSetWarningInfoReturnsWrappedErrorOnPersistFailure(t *testing.T) {
	resetDiskSeamsForTest(t)

	updateDiskWarningInConf = func(map[string]structs.DiskWarning) error {
		return errors.New("persist failed")
	}

	err := setWarningInfo(structs.SysConfig{}, "disk0", false, true, time.Time{})
	if err == nil {
		t.Fatal("expected setWarningInfo to return error")
	}
	if got := err.Error(); got == "" || !containsAll(got, []string{"80%:false", "90%:true", "persist failed"}) {
		t.Fatalf("unexpected wrapped error: %v", err)
	}
}

func TestSendDriveHarkNotificationSendsPerPier(t *testing.T) {
	resetDiskSeamsForTest(t)

	type call struct {
		patp string
		noti structs.HarkNotification
	}
	calls := []call{}
	sendNotificationForDiskWarning = func(patp string, noti structs.HarkNotification) error {
		calls = append(calls, call{patp: patp, noti: noti})
		return nil
	}

	sendDriveHarkNotification("disk1", 91.25, []string{"~zod", "~bus"})

	if len(calls) != 2 {
		t.Fatalf("expected 2 notification calls, got %d", len(calls))
	}
	if calls[0].noti.Type != "disk-warning" || calls[0].noti.DiskName != "disk1" || calls[0].noti.DiskUsage != 91.25 {
		t.Fatalf("unexpected disk-warning payload: %+v", calls[0].noti)
	}
	if calls[0].patp != "~zod" || calls[1].patp != "~bus" {
		t.Fatalf("unexpected notification recipients: %+v", calls)
	}
}

func TestSendSmartFailHarkNotificationSendsPerPier(t *testing.T) {
	resetDiskSeamsForTest(t)

	type call struct {
		patp string
		noti structs.HarkNotification
	}
	calls := []call{}
	sendNotificationForDiskWarning = func(patp string, noti structs.HarkNotification) error {
		calls = append(calls, call{patp: patp, noti: noti})
		return nil
	}

	sendSmartFailHarkNotification("/dev/sda", []string{"~nec", "~zod"})

	if len(calls) != 2 {
		t.Fatalf("expected 2 notification calls, got %d", len(calls))
	}
	if calls[0].noti.Type != "smart-fail" || calls[0].noti.DiskName != "/dev/sda" {
		t.Fatalf("unexpected smart-fail payload: %+v", calls[0].noti)
	}
	if calls[0].patp != "~nec" || calls[1].patp != "~zod" {
		t.Fatalf("unexpected notification recipients: %+v", calls)
	}
}

func containsAll(s string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
