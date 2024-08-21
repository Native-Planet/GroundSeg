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
		//"(pure:m !>((jam [123 123 123 123])))",
	})
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click %s failed to create hoon: %v", file, err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file, "")
	if err != nil {
		return fmt.Errorf("Click %s failed to get exec: %v", file, err)
	}
	jamFile, jamNoun, err := filterJamResponse(patp, file, response)
	if err != nil {
		return fmt.Errorf("Click %s failed to get exec: %v", file, err)
	}
	zap.L().Debug(fmt.Sprintf("jamNoun for %s: %+v", file, jamNoun))
	if jamNoun == nil {
		zap.L().Error("jamNoun is invalid!")
		//return fmt.Errorf("Click %s failed scry: %s", file, patp)
	}
	zap.L().Debug(fmt.Sprintf("jam response %s placeholder. Jam file: %s", file, jamFile))
	return nil
}

/*
=/  m  (strand ,vase)

	  ;<    a=egg-any:gall
			  bind:m
		(scry egg-any:gall /gv/channels/$)
	(pure:m !>((jam ?>(?=(%live +<.a) a(p.old-state -:!>(*))))))
*/
func backupChannels(patp string) error {
	file := "backup-channels"
	hoon := joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "a=egg-any:gall", "bind:m", "(scry egg-any:gall /gv/channels/$)",
		"(pure:m !>((jam ?>(?=(%live +<.a) a(p.old-state -:!>(*))))))",
	})
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click %s failed to create hoon: %v", file, err)
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
	zap.L().Debug(fmt.Sprintf("jam response %s placeholder. Jam file: %s", file, jamFile))
	return nil
}

// *%/bak/channels-server/jam (jam =+(.^(a=egg-any:gall %gv /=channels-server=/$) ?>(?=(%live +<.a) a(p.old-state -:!>(*)))))
func backupChannelsServer(patp string) error {
	file := "backup-channels-server"
	hoon := joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "a=egg-any:gall", "bind:m", "(scry egg-any:gall /gv/channels-server/$)",
		"(pure:m !>((jam ?>(?=(%live +<.a) a(p.old-state -:!>(*))))))",
	})
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click %s failed to create hoon: %v", file, err)
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
	zap.L().Debug(fmt.Sprintf("jam response %s placeholder. Jam file: %s", file, jamFile))
	return nil
}

func backupGroups(patp string) error {
	file := "backup-groups"
	hoon := joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "a=egg-any:gall", "bind:m", "(scry egg-any:gall /gv/groups/$)",
		"(pure:m !>((jam ?>(?=(%live +<.a) a(p.old-state -:!>(*))))))",
	})
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click %s failed to create hoon: %v", file, err)
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
	zap.L().Debug(fmt.Sprintf("jam response %s placeholder. Jam file: %s", file, jamFile))
	return nil
}

func backupProfile(patp string) error {
	file := "backup-profile"
	hoon := joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "a=egg-any:gall", "bind:m", "(scry egg-any:gall /gv/profile/$)",
		"(pure:m !>((jam ?>(?=(%live +<.a) a(p.old-state -:!>(*))))))",
	})
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click %s failed to create hoon: %v", file, err)
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
	zap.L().Debug(fmt.Sprintf("jam response %s placeholder. Jam file: %s", file, jamFile))
	return nil
}

func backupChat(patp string) error {
	file := "backup-chat"
	hoon := joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "a=egg-any:gall", "bind:m", "(scry egg-any:gall /gv/chat/$)",
		"(pure:m !>((jam ?>(?=(%live +<.a) a(p.old-state -:!>(*))))))",
	})
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click %s failed to create hoon: %v", file, err)
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
	zap.L().Debug(fmt.Sprintf("jam response %s placeholder. Jam file: %s", file, jamFile))
	return nil
}
