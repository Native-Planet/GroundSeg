package click

import (
	"fmt"

	"go.uber.org/zap"
)

/*
=/  m  (strand ,vase)

	  ;<    a=egg-any:gall
			  bind:m
		(scry egg-any:gall /gv/activity/$)

(pure:m !>((jam ?>(?=(%live +<.a) a(p.old-state -:!>(*))))))
*/
func backupActivity(patp string) error {
	file := "backup-activity"
	hoon := joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "a=egg-any:gall", "bind:m", "(scry egg-any:gall /gv/activity/$)",
		"(pure:m !>((jam ?>(?=(%live +<.a) a(p.old-state -:!>(*))))))",
	})
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click startram hark notification failed to create hoon: %v", err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file, "")
	if err != nil {
		return fmt.Errorf("Click %s failed to get exec: %v", file, err)
	}
	jamFile, succeeded, err := filterResponse("jam", response)
	if err != nil {
		return fmt.Errorf("Click %s failed to get exec: %v", file, err)
	}
	if !succeeded {
		return fmt.Errorf("Click %s failed scry: %s", file, patp)
	}
	zap.L().Debug(fmt.Sprintf("jam response placeholder. Jam file: %s", jamFile))
	return nil
}
