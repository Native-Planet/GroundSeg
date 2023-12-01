package click

import (
	"fmt"
	"goseg/logger"
	"goseg/structs"
	"time"
)

// +code
func GetLusCode(patp string) (string, error) {
	// in var already?
	proceedWithRequest := allowLusCodeRequest(patp)
	if !proceedWithRequest {
		codeMutex.Lock()
		defer codeMutex.Unlock()
		code, exists := lusCodes[patp]
		if !exists {
			return "", fmt.Errorf("Click +code failed to fetch code from memory")
		}
		return code.LusCode, nil
	}
	// logger.Logger.Debug(fmt.Sprintf("Allowing +code request for %s", patp))
	// <file>.hoon
	file := "code"
	// actual hoon
	hoon := "=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  code=@p  bind:m  (scry @p /j/code/(scot %p our))  (pure:m !>((crip (slag 1 (scow %p code)))))"
	// create hoon file
	if err := createHoon(patp, file, hoon); err != nil {
		return "", fmt.Errorf("Click +code failed to create hoon: %v", err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file, "")
	if err != nil {
		storeLusCodeError(patp)
		return "", fmt.Errorf("Click +code failed to get exec: %v", err)
	}
	// retrieve code
	code, _, err := filterResponse("code", response)
	if err != nil {
		storeLusCodeError(patp)
		return "", fmt.Errorf("Click +code failed to get exec: %v", err)
	}
	storeLusCode(patp, code)
	return code, nil
}

func storeLusCodeError(patp string) {
	logger.Logger.Debug(fmt.Sprintf("Recording +code failure for %s", patp))
	codeMutex.Lock()
	defer codeMutex.Unlock()
	lusCodes[patp] = structs.ClickLusCode{
		LastError: time.Now(),
	}
}

func storeLusCode(patp, code string) {
	logger.Logger.Info(fmt.Sprintf("Storing +code for %s", patp))
	codeMutex.Lock()
	defer codeMutex.Unlock()
	lusCodes[patp] = structs.ClickLusCode{
		LastFetch: time.Now(),
		LusCode:   code,
	}
}

func allowLusCodeRequest(patp string) bool {
	codeMutex.Lock()
	defer codeMutex.Unlock()
	// if patp doesn't exist
	data, exists := lusCodes[patp]
	if !exists {
		return true
	}
	// flood control
	if time.Since(data.LastError) < 1*time.Second {
		return false
	}
	// if +code not legit
	if len(data.LusCode) != 27 {
		return true
	}
	// if it has been 15 minutes
	if time.Since(data.LastFetch) > 15*time.Minute {
		return true
	}
	// use the +code stored
	return false
}
