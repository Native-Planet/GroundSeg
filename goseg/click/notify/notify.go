package notify

import (
	"fmt"

	"groundseg/click/internal/runtime"
	"groundseg/structs"
)

var (
	executeClickCommandForHark = runtime.ExecuteCommandWithSuccess
)

type harkNotification struct {
	file       string
	id         string
	content    string
	rope       string
	wer        string
	errorLabel string
}

var (
	sendStartramReminderFn = sendStartramReminder
	sendDiskSpaceWarningFn = sendDiskSpaceWarning
	sendSmartWarningFn     = sendSmartWarning
)

func SendNotification(patp string, payload structs.HarkNotification) error {
	switch payload.Type {
	case "startram-reminder":
		return sendStartramReminderFn(patp, payload.StartramDaysLeft)
	case "disk-warning":
		return sendDiskSpaceWarningFn(patp, payload.DiskName, payload.DiskUsage)
	case "smart-fail":
		return sendSmartWarningFn(patp, payload.DiskName)
	default:
		return fmt.Errorf("invalid hark notification type: %s", payload.Type)
	}
}

func buildHarkAddYarnHoon(notification harkNotification) string {
	return runtime.JoinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "=bowl:rand", "bind:m", "get-bowl",
		";<", "~", "bind:m",
		fmt.Sprintf("(poke [our.bowl %%hark] %%hark-action !>([%%add-yarn & & [%s %s now.bowl %s %s ~]]))", notification.id, notification.rope, notification.content, notification.wer),
		"(pure:m !>('success'))",
	})
}

func sendHarkNotification(patp string, notification harkNotification) error {
	hoon := buildHarkAddYarnHoon(notification)
	_, err := executeClickCommandForHark(
		patp,
		notification.file,
		hoon,
		"",
		"success",
		notification.errorLabel,
		nil,
	)
	if err != nil {
		return fmt.Errorf("%s failed to execute hoon: %w", notification.errorLabel, err)
	}
	return nil
}

func sendStartramReminder(patp string, daysLeft int) error {
	text := fmt.Sprintf("'Your startram code is expiring in %v days. Click for more information.'", daysLeft)
	con := fmt.Sprintf("~[%s %s]", text, text)
	return sendHarkNotification(patp, harkNotification{
		file:       "startram-hark",
		id:         "(end 7 (shas %startram-notification eny.bowl))",
		content:    con,
		rope:       "[[~ our.bowl %nativeplanet] [~ %diary our.bowl %documentation] %groups /]",
		wer:        "/groups/'~nattyv'/nativeplanet/channels/diary/'~nattyv'/documentation/note/'170141184506683847949839018055058849792'",
		errorLabel: "Click startram hark notification",
	})
}

func sendDiskSpaceWarning(patp, diskName string, diskUsage float64) error {
	text := fmt.Sprintf("'Your drive %s is %v%% full. Manage your disk to prevent issues!'", diskName, diskUsage)
	return sendHarkNotification(patp, harkNotification{
		file:       "diskspace-hark",
		id:         "(end 7 (shas %diskusage eny.bowl))",
		content:    fmt.Sprintf("~[%s]", text),
		rope:       "[[~ our.bowl %nativeplanet] [~ %diary our.bowl %documentation] %groups /]",
		wer:        "/groups/'~nattyv'/nativeplanet/channels/diary/'~nattyv'/documentation/note/'170141184506683847949839018055058849792'",
		errorLabel: "Click disk warning hark notification",
	})
}

func sendSmartWarning(patp, diskName string) error {
	text := fmt.Sprintf("'Your drive %s failed a health check. Replace your hard drive to prevent data loss!'", diskName)
	return sendHarkNotification(patp, harkNotification{
		file:       "smart-fail-hark",
		id:         "(end 7 (shas %smartfail eny.bowl))",
		content:    fmt.Sprintf("~[%s]", text),
		rope:       "[[~ our.bowl %nativeplanet] [~ %diary our.bowl %documentation] %groups /]",
		wer:        "/groups/'~nattyv'/nativeplanet/channels/diary/'~nattyv'/documentation/note/'170141184506683847949839018055058849792'",
		errorLabel: "Click disk failure hark notification",
	})
}
