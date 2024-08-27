package click

import (
	"fmt"
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
	hoon := joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "our=@p", "bind:m", "get-our",
		";<", "a=egg-any:gall", "bind:m", scry,
		";<", "~", "bind:m", fmt.Sprintf("(poke [our %%hood] %%drum-put !>([/%s/jam %s]))", file, stateJam),
		"(pure:m !>('success'))",
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
	_, succeeded, err := filterResponse("success", response)
	if err != nil {
		return fmt.Errorf("Click %s failed to get exec: %v", file, err)
	}
	if !succeeded {
		return fmt.Errorf("Click %s failed poke: %s", patp)
	}
	return nil
}
