package acme

import (
	"fmt"
	"groundseg/click/internal/runtime"
)

var (
	executeCommandForAcme = runtime.ExecuteCommand
)

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
func Fix(patp string) error {
	file := "acmeresetcert"
	hoon := "=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  ~  bind:m  (poke [our %hood] %helm-pass !>([%e %rule %cert ~]))  ;<  ~  bind:m  (poke [our %acme] %noun !>([%init]))  (pure:m !>('success'))"
	_, err := executeCommandForAcme(patp, file, hoon, "", "success", "Click acme reset cert")
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}
