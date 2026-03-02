package lifecycle

import (
	"fmt"

	"groundseg/click/internal/runtime"
	"groundseg/click/luscode"
)

var (
	executeClickCommandForExit = runtime.ExecuteCommandWithSuccess
	clearLusCode               = luscode.ClearLusCode
)

// BarExit exits the ship and returns an error on failure.
func BarExit(patp string) error {
	defer clearLusCode(patp)
	file := "exit"
	hoon := "=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  ~  bind:m  (poke [our %hood] %drum-exit !>(~))  (pure:m !>('success'))"
	if _, err := executeClickCommandForExit(
		patp,
		file,
		hoon,
		"",
		"success",
		fmt.Sprintf("Click |exit for %v", patp),
		clearLusCode,
	); err != nil {
		return fmt.Errorf("Click |exit for %v failed to get exec: %v", patp, err)
	}
	return nil
}
