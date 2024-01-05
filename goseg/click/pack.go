package click

import "fmt"

// |pack
func sendPack(patp string) error {
	// <file>.hoon
	file := "pack"
	// actual hoon
	hoon := "=/  m  (strand ,vase)  ;<  ~  bind:m  (flog [%pack ~])  (pure:m !>('success'))"
	// create hoon file
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click |pack failed to create hoon: %v", err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file, "")
	if err != nil {
		return fmt.Errorf("Click |pack failed to get exec: %v", err)
	}
	// retrieve code
	_, success, err := filterResponse("success", response)
	if err != nil {
		return fmt.Errorf("Click |pack failed to get exec: %v", err)
	}
	if !success {
		return fmt.Errorf("Click |pack poke failed")
	}
	return nil
}
