package click

import (
	"fmt"
)

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
	file := "startram-hark"
	// construct poke
	text := fmt.Sprintf("'Your startram code is expiring in %v days. Click for more information.'", daysLeft)
	con := fmt.Sprintf("~[%s %s]", text, text)
	//id := "0v14e.5p95b.d5mk5.vrqe0.oeu0m.3ghcb"
	id := "(end 7 (shas %startram-notification eny.bowl))"
	rope := "[[~ our.bowl %nativeplanet] [~ %diary our.bowl %documentation] %groups /]"                                           // temp location
	wer := "/groups/'~nattyv'/nativeplanet/channels/diary/'~nattyv'/documentation/note/'170141184506683847949839018055058849792'" //temp location

	but := "~"
	hoon := joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "=bowl:rand", "bind:m", "get-bowl",
		";<", "~", "bind:m",
		fmt.Sprintf("(poke [our.bowl %%hark] %%hark-action !>([%%add-yarn & & [%s %s now.bowl %s %s %s]]))", id, rope, con, wer, but),
		"(pure:m !>('success'))",
	})

	// create hoon file
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click startram hark notification failed to create hoon: %v", err)
	}
	// defer hoon file deletion
	//defer deleteHoon(patp, file)
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
