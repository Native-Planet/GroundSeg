package click

import (
	"fmt"
)

func harkNotificationHoon(category, text string) string {
	place := fmt.Sprintf("[q.byk.bowl /nativeplanet/%s]", category)
	bin := fmt.Sprintf("[/%s place]", category)
	content := fmt.Sprintf("~[[%%text %s]]", text)

	return joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "=bowl:rand", "bind:m", "get-bowl",
		"=/", "place", place,
		"=/", "bin", bin,
		"=/", "body", fmt.Sprintf("[%s %s now.bowl / /]", content, content),
		";<", "~", "bind:m",
		"(poke [our.bowl %hark-store] %hark-action !>([%add-note bin body]))",
		"(pure:m !>('success'))",
	})
}

func sendStartramReminder(patp string, daysLeft int) error {
	file := "startram-hark"
	text := fmt.Sprintf("'Your startram code is expiring in %v days. Click for more information.'", daysLeft)
	hoon := harkNotificationHoon("startram", text)

	// create hoon file
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click startram hark notification failed to create hoon: %v", err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file, "")
	if err != nil {
		return fmt.Errorf("Click startram hark notification failed to get exec: %v", err)
	}
	_, succeeded, err := filterResponse("success", response)
	if err != nil {
		return fmt.Errorf("Click startram hark notification failed to get exec: %v", err)
	}
	if !succeeded {
		return fmt.Errorf("Click startram hark notification failed poke: %s", patp)
	}
	return nil
}

func sendDiskSpaceWarning(patp, diskName string, diskUsage float64) error {
	file := "diskspace-hark"
	text := fmt.Sprintf("'Your drive %s is %v%% full. Manage your disk to prevent issues!'", diskName, diskUsage)
	hoon := harkNotificationHoon("disk-space", text)

	// create hoon file
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click disk warning hark notification failed to create hoon: %v", err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file, "")
	if err != nil {
		return fmt.Errorf("Click disk warning hark notification failed to get exec: %v", err)
	}
	_, succeeded, err := filterResponse("success", response)
	if err != nil {
		return fmt.Errorf("Click disk warning hark notification failed to get exec: %v", err)
	}
	if !succeeded {
		return fmt.Errorf("Click disk warning hark notification failed poke: %s", patp)
	}
	return nil
}

func sendSmartWarning(patp, diskName string) error {
	file := "smart-fail-hark"
	text := fmt.Sprintf("'Your drive %s failed a health check. Replace your hard drive to prevent data loss!'", diskName)
	hoon := harkNotificationHoon("disk-health", text)

	// create hoon file
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click disk failure hark notification failed to create hoon: %v", err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file, "")
	if err != nil {
		return fmt.Errorf("Click disk failure hark notification failed to get exec: %v", err)
	}
	_, succeeded, err := filterResponse("success", response)
	if err != nil {
		return fmt.Errorf("Click disk failure hark notification failed to get exec: %v", err)
	}
	if !succeeded {
		return fmt.Errorf("Click disk failure hark notification failed poke: %s", patp)
	}
	return nil
}
