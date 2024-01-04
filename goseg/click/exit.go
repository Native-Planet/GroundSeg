package click

import "fmt"

// |exit
func BarExit(patp string) error {
	defer ClearLusCode(patp)
	// <file>.hoon
	file := "exit"
	// actual hoon
	hoon := "=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  ~  bind:m  (poke [our %hood] %drum-exit !>(~))  (pure:m !>('success'))"
	// create hoon file
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click |exit for %v failed to create hoon: %v", patp, err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file, "")
	if err != nil {
		return fmt.Errorf("Click |exit for %v failed to get exec: %v", patp, err)
	}
	// retrieve code
	_, success, err := filterResponse("success", response)
	if err != nil {
		return fmt.Errorf("Click |exit for %v failed to get exec: %v", patp, err)
	}
	if !success {
		return fmt.Errorf("Click |exit for %v poke failed", patp)
	}
	return nil
}
