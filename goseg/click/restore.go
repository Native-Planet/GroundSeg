package click

import "fmt"

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
		";<", "~", "bind:m", "(take-poke-ack /pokeas)", "(pure:m !>('success'))",
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
	println(response)
	_, succeeded, err := filterResponse("success", response)
	if err != nil {
		return fmt.Errorf("Click %s failed to get exec: %v", file, err)
	}
	if !succeeded {
		return fmt.Errorf("Click %s failed poke: %s", file, patp)
	}
	return nil
}
