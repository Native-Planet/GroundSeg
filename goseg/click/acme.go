package click

import "fmt"

/*
=/  m  (strand ,vase)
;<    our=@p
		bind:m
	get-our
;<    ~
		bind:m
	(poke [our %hood] %helm-pass !>([%e %rule %cert ~]))
;<    ~
		bind:m
	(poke [our %acme] %noun !>([%init]))
(pure:m !>('success'))
*/

func FixAcme(patp string) error {
	// <file>.hoon
	file := "acmeresetcert"
	// actual hoon
	hoon := "=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  ~  bind:m  (poke [our %hood] %helm-pass !>([%e %rule %cert ~]))  ;<  ~  bind:m  (poke [our %acme] %noun !>([%init]))  (pure:m !>('success'))"
	// create hoon file
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click acme reset cert failed to create hoon: %v", err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file)
	if err != nil {
		return fmt.Errorf("Click acme reset cert failed to get exec: %v", err)
	}
	_, _, err = filterResponse("success", response)
	if err != nil {
		return fmt.Errorf("Click acme reset cert failed to get exec: %v", err)
	}
	return nil
}
