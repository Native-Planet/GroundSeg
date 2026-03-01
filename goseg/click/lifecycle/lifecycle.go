package lifecycle

import (
	"fmt"

	"groundseg/click/internal/runtime"
	"groundseg/click/luscode"
)

var (
	createHoonForExit     = runtime.CreateHoon
	deleteHoonForExit     = runtime.DeleteHoon
	clickExecForExit      = runtime.ClickExec
	filterResponseForExit = runtime.FilterResponse
	clearLusCode          = luscode.ClearLusCode
)

func createHoonForExitCommand(patp, file, hoon string) error {
	if err := createHoonForExit(patp, file, hoon); err != nil {
		return err
	}
	clearLusCode(patp)
	return nil
}

// BarExit exits the ship and returns an error on failure.
func BarExit(patp string) error {
	defer clearLusCode(patp)
	file := "exit"
	hoon := "=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  ~  bind:m  (poke [our %hood] %drum-exit !>(~))  (pure:m !>('success'))"
	if err := createHoonForExitCommand(patp, file, hoon); err != nil {
		return fmt.Errorf("Click |exit for %v failed to create hoon: %v", patp, err)
	}
	defer deleteHoonForExit(patp, file)

	response, err := clickExecForExit(patp, file, "")
	if err != nil {
		return fmt.Errorf("Click |exit for %v failed to get exec: %v", patp, err)
	}
	_, success, err := filterResponseForExit("success", response)
	if err != nil {
		return fmt.Errorf("Click |exit for %v failed to get exec: %v", patp, err)
	}
	if !success {
		return fmt.Errorf("Click |exit for %v poke failed", patp)
	}
	return nil
}
