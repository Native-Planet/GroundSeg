package click

import (
	"fmt"
)

/*
		=/  m  (strand ,vase)
		;<    our=@p
		    bind:m
		  get-our
		;<    ~
		    bind:m
			=+  id=0v0
	    =/  con=(list content:h)  ~[text text]
	    =+  rope=[[~ our %nativeplanet] [~ %diary our %changelog] %groups /]
	    =/  wer=path  /groups/'~nattyv'/nativeplanet/channels/diary/'~nattyv'/changelog/note/'170141184506582040503264511680103579648'
			=+  but=~
		  (poke [our %hark] %hark-action !>([%add-yarn & & id rope con wer but])
		(pure:m !>('success'))
*/
func sendStartramReminder(patp string, daysLeft int) error {
	file := "startram-hark"
	// construct poke
	text := fmt.Sprintf("'Your startram code is expiring in %v days. Click for more information.'", daysLeft)
	con := fmt.Sprintf("~[%s %s]", text, text)
	id := "0v14e.5p95b.d5mk5.vrqe0.oeu0m.3ghcb"
	rope := "[[~ our %nativeplanet] [~ %diary our %changelog] %groups /]"                                                     // temp location
	wer := "/groups/'~nattyv'/nativeplanet/channels/diary/'~nattyv'/changelog/note/'170141184506582040503264511680103579648'" //temp location
	but := "~"
	hoon := joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "our=@p", "bind:m", "get-our",
		";<", "~", "bind:m",
		fmt.Sprintf("(poke [our %%hark] %%hark-action !>([%%add-yarn & & [%s %s %s %s %s]]))", id, rope, con, wer, but),
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
