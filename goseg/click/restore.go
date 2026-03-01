package click

import (
	"fmt"

	"go.uber.org/zap"
)

var (
	executeClickCommandForRestore = executeClickCommand
	filterResponseForRestore      = filterResponse
)

func restoreAgent(patp, agent string) error {
	file := fmt.Sprintf("restore-%s", agent)
	hoon := joinGap([]string{
		"=/", "m", "(strand ,vase)",
		"^-", "form:m",
		";<", "egg=@", "bind:m", fmt.Sprintf("(scry @ /cx/base/bak/backup-%s/jam)", agent),
		";<", "our=@p", "bind:m", "get-our",
		"=/", "dk=dock", fmt.Sprintf("[our %%%s]", agent),
		"=/", "cg=cage", "[%noun !>((cue egg))]",
		"=/", "=card:agent:gall", "[%pass /pokeas %agent dk %poke %egg-any -:!>(*egg-any:gall) (cue egg)]",
		";<", "~", "bind:m", "(send-raw-card card)",
		";<", "~", "bind:m", "(take-poke-ack /pokeas)",
		"(pure:m !>('success'))",
	})
	response, err := executeClickCommandForRestore(patp, file, hoon, "", "", fmt.Sprintf("Click %s", file))
	if err != nil {
		return err
	}
	_, succeeded, err := filterResponseForRestore("success", response)
	if err != nil {
		return fmt.Errorf("Click %s failed to get exec: %v", file, err)
	}
	if !succeeded {
		return fmt.Errorf("Click %s failed poke: %s", file, patp)
	}
	zap.L().Info(fmt.Sprintf("Click %s restored agent %s on %s", file, agent, patp))
	return nil
}
