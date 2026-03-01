package click

import (
	"fmt"
)

var (
	createHoonForHark     = createHoon
	deleteHoonForHark     = deleteHoon
	clickExecForHark      = clickExec
	filterResponseForHark = filterResponse
)

type harkNotification struct {
	file       string
	id         string
	content    string
	rope       string
	wer        string
	errorLabel string
}

func buildHarkAddYarnHoon(notification harkNotification) string {
	return joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "=bowl:rand", "bind:m", "get-bowl",
		";<", "~", "bind:m",
		fmt.Sprintf("(poke [our.bowl %%hark] %%hark-action !>([%%add-yarn & & [%s %s now.bowl %s %s ~]]))", notification.id, notification.rope, notification.content, notification.wer),
		"(pure:m !>('success'))",
	})
}

func sendHarkNotification(patp string, notification harkNotification) error {
	hoon := buildHarkAddYarnHoon(notification)
	if err := createHoonForHark(patp, notification.file, hoon); err != nil {
		return fmt.Errorf("%s failed to create hoon: %w", notification.errorLabel, err)
	}
	defer deleteHoonForHark(patp, notification.file)

	response, err := clickExecForHark(patp, notification.file, "")
	if err != nil {
		return fmt.Errorf("%s failed to execute hoon: %w", notification.errorLabel, err)
	}
	_, succeeded, err := filterResponseForHark("success", response)
	if err != nil {
		return fmt.Errorf("%s failed to parse response: %w", notification.errorLabel, err)
	}
	if !succeeded {
		return fmt.Errorf("%s failed poke for %s", notification.errorLabel, patp)
	}
	return nil
}

/*
		=/  m  (strand ,vase)
		;<    =bowl:rand
		    bind:m
		  get-our
		;<    ~
		    bind:m
			=+  id=(end 7 (shas %startram-notification eny.bowl))
	    =/  con=(list content:h)  ~[text text]
	    =+  rope=[[~ our.bowl %nativeplanet] [~ %diary our.bowl %changelog] %groups /]
	    =/  wer=path  /groups/'~nattyv'/nativeplanet/channels/diary/'~nattyv'/changelog/note/'170141184506582040503264511680103579648'
			=+  but=~
		  (poke [our.bowl %hark] %hark-action !>([%add-yarn & & id rope now.bowl con wer but])
		(pure:m !>('success'))
*/
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
