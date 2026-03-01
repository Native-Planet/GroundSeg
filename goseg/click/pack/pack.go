package pack

import (
	"groundseg/click/internal/runtime"
)

var executeClickCommandForPack = runtime.ExecuteCommand

func SendPack(patp string) error {
	file := "pack"
	hoon := "=/  m  (strand ,vase)  ;<  ~  bind:m  (flog [%pack ~])  (pure:m !>('success'))"
	_, err := executeClickCommandForPack(patp, file, hoon, "", "success", "Click |pack")
	return err
}
