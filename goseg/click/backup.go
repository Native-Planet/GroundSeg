package click

import (
	"fmt"
	"strings"
	"unicode"
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
func backupActivity(patp string) error {
	file := "backup-activity"
	//hoon_thread = "=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  code=@p  bind:m  (scry @p /j/code/(scot %p our))  (pure:m !>((crip (slag 1 (scow %p code)))))"

	hoon := joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "a=egg-any:gall", "bind:m", "(scry egg-any:gall /g/v/=activity=/$)",
		"(pure:m !>(?>(?=(%live +<.a) a(p.old-state -:!>(*)))))",
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
	cleanString := func(input string) string {
		var builder strings.Builder
		for _, r := range input {
			// Only append printable characters
			if unicode.IsPrint(r) {
				builder.WriteRune(r)
			}
		}
		return builder.String()
	}
	res := strings.Split(response, "\r\n")
	for _, ln := range res {
		fmt.Println(cleanString(ln))
	}

	/*
		_, succeeded, err := filterResponse("success", response)
		if err != nil {
			return fmt.Errorf("Click startram hark notification failed to get exec: %v", err)
		}
		if !succeeded {
			return fmt.Errorf("Click startram hark notification failed poke: %s", patp)
		}
	*/
	return nil
}
