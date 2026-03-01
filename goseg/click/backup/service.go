package backup

import (
	"fmt"
	"groundseg/click/internal/runtime"
	"strings"
)

var (
	executeClickCommandForBackup = runtime.ExecuteCommand
	filterResponseForBackup      = runtime.FilterResponse
	joinGapForBackup             = runtime.JoinGap
	backupAgentFn                = backupAgent
)

/*
	  =/  m  (strand ,vase)
		  ;<  our=@p  bind:m  get-our
	    ;<  a=egg-any:gall
			  bind:m
			(scry egg-any:gall /gv/<agent>/$)
	  (pure:m !>((jam ?>(?=(%live +<.a) a(p.old-state -:!>(*))))))
*/
func backupAgent(patp, agent string) error {
	file := fmt.Sprintf("backup-%s", agent)
	stateJam := "(jam ?>(?=(%live +<.a) a(p.old-state -:!>(*))))"
	scry := fmt.Sprintf("(scry egg-any:gall /gv/%s/$)", agent)
	hoon := joinGapForBackup([]string{
		"=/", "m", "(strand ,vase)",
		";<", "our=@p", "bind:m", "get-our",
		";<", "a=egg-any:gall", "bind:m", scry,
		";<", "~", "bind:m", fmt.Sprintf("(poke [our %%hood] %%drum-put !>([/%s/jam %s]))", file, stateJam),
		"(pure:m !>('success'))",
	})
	response, err := executeClickCommandForBackup(patp, file, hoon, "", "", fmt.Sprintf("Click %s", file))
	if err != nil {
		return err
	}
	_, succeeded, err := filterResponseForBackup("success", response)
	if err != nil {
		return fmt.Errorf("Click %s failed to get exec: %v", file, err)
	}
	if !succeeded {
		return fmt.Errorf("Click %s failed poke for %s", file, patp)
	}
	return nil
}

func BackupTlon(patp string) error {
	var errors []string

	components := []struct {
		name string
		err  error
	}{
		{"activity", backupAgentFn(patp, "activity")},
		{"channels", backupAgentFn(patp, "channels")},
		{"channels-server", backupAgentFn(patp, "channels-server")},
		{"groups", backupAgentFn(patp, "groups")},
		{"profile", backupAgentFn(patp, "profile")},
		{"chat", backupAgentFn(patp, "chat")},
	}

	for _, component := range components {
		if component.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", component.name, component.err))
		}
	}

	if len(errors) == 0 {
		return nil
	}
	return fmt.Errorf("backup errors: %s", strings.Join(errors, ", "))
}
