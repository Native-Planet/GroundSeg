package routines

import (
	"errors"
	"reflect"
	"testing"

	"groundseg/config"
	"groundseg/structs"
)

func resetRenewalSeamsForTest(t *testing.T) {
	t.Helper()
	origOne := withStartramReminderOneForRenewal
	origThree := withStartramReminderThreeForRenewal
	origSeven := withStartramReminderSevenForRenewal
	origUpdate := updateConfTypedForRenewal
	origUrbitConf := urbitConfForRenewal
	origNotify := sendNotificationForRenewal

	t.Cleanup(func() {
		withStartramReminderOneForRenewal = origOne
		withStartramReminderThreeForRenewal = origThree
		withStartramReminderSevenForRenewal = origSeven
		updateConfTypedForRenewal = origUpdate
		urbitConfForRenewal = origUrbitConf
		sendNotificationForRenewal = origNotify
	})
}

func TestSetReminderUnknownTypeDoesNotUpdateConfig(t *testing.T) {
	resetRenewalSeamsForTest(t)

	called := false
	updateConfTypedForRenewal = func(_ ...config.ConfigUpdateOption) error {
		called = true
		return nil
	}

	setReminder("invalid", true)
	if called {
		t.Fatal("expected unknown reminder type to skip config update")
	}
}

func TestSetReminderAppliesCorrectConfigPatch(t *testing.T) {
	resetRenewalSeamsForTest(t)

	type patchResult struct {
		one   *bool
		three *bool
		seven *bool
	}
	captured := []patchResult{}
	updateConfTypedForRenewal = func(opts ...config.ConfigUpdateOption) error {
		if len(opts) != 1 {
			t.Fatalf("expected exactly one config option, got %d", len(opts))
		}
		option := opts[0]
		patch := &config.ConfPatch{}
		option(patch)
		captured = append(captured, patchResult{
			one:   patch.StartramReminderOne,
			three: patch.StartramReminderThree,
			seven: patch.StartramReminderSeven,
		})
		return nil
	}

	setReminder("one", true)
	setReminder("three", false)
	setReminder("seven", true)

	if len(captured) != 3 {
		t.Fatalf("expected 3 config updates, got %d", len(captured))
	}

	if captured[0].one == nil || *captured[0].one != true || captured[0].three != nil || captured[0].seven != nil {
		t.Fatalf("unexpected patch for one-day reminder: %+v", captured[0])
	}
	if captured[1].three == nil || *captured[1].three != false || captured[1].one != nil || captured[1].seven != nil {
		t.Fatalf("unexpected patch for three-day reminder: %+v", captured[1])
	}
	if captured[2].seven == nil || *captured[2].seven != true || captured[2].one != nil || captured[2].three != nil {
		t.Fatalf("unexpected patch for seven-day reminder: %+v", captured[2])
	}
}

func TestSendStartramHarkNotificationOnlyTargetsOptedInShips(t *testing.T) {
	resetRenewalSeamsForTest(t)

	urbitConfForRenewal = func(patp string) structs.UrbitDocker {
		switch patp {
		case "~zod":
			return structs.UrbitDocker{
				UrbitFeatureConfig: structs.UrbitFeatureConfig{StartramReminder: true},
			}
		case "~bus":
			return structs.UrbitDocker{
				UrbitFeatureConfig: structs.UrbitFeatureConfig{StartramReminder: false},
			}
		case "~nec":
			return structs.UrbitDocker{
				UrbitFeatureConfig: structs.UrbitFeatureConfig{StartramReminder: true},
			}
		default:
			return structs.UrbitDocker{}
		}
	}

	type call struct {
		patp string
		noti structs.HarkNotification
	}
	calls := []call{}
	sendNotificationForRenewal = func(patp string, noti structs.HarkNotification) error {
		calls = append(calls, call{patp: patp, noti: noti})
		return nil
	}

	sendStartramHarkNotification(3, []string{"~zod", "~bus", "~nec"})

	if len(calls) != 2 {
		t.Fatalf("expected 2 notifications for opted-in ships, got %d", len(calls))
	}
	if !reflect.DeepEqual([]string{calls[0].patp, calls[1].patp}, []string{"~zod", "~nec"}) {
		t.Fatalf("unexpected notification recipients: %+v", calls)
	}
	for _, c := range calls {
		if c.noti.Type != "startram-reminder" || c.noti.StartramDaysLeft != 3 {
			t.Fatalf("unexpected notification payload: %+v", c.noti)
		}
	}
}

func TestSendStartramHarkNotificationContinuesOnSendError(t *testing.T) {
	resetRenewalSeamsForTest(t)

	urbitConfForRenewal = func(string) structs.UrbitDocker {
		return structs.UrbitDocker{
			UrbitFeatureConfig: structs.UrbitFeatureConfig{StartramReminder: true},
		}
	}
	calls := 0
	sendNotificationForRenewal = func(patp string, noti structs.HarkNotification) error {
		calls++
		if patp == "~zod" {
			return errors.New("send failed")
		}
		return nil
	}

	sendStartramHarkNotification(1, []string{"~zod", "~bus"})
	if calls != 2 {
		t.Fatalf("expected notifications to continue after error, got %d calls", calls)
	}
}
