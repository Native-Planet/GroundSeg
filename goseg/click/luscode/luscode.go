package luscode

import (
	"fmt"
	"sync"
	"time"

	"groundseg/click/internal/runtime"
	"groundseg/structs"

	"go.uber.org/zap"
)

var (
	lusCodes  = make(map[string]structs.ClickLusCode)
	codeMutex sync.Mutex

	executeClickCommandForCode = runtime.ExecuteCommandWithResponse
)

// ClearLusCode clears cached +code values for a ship.
func ClearLusCode(patp string) {
	codeMutex.Lock()
	defer codeMutex.Unlock()
	delete(lusCodes, patp)
}

// GetLusCode returns the cached or fresh +code result for a ship.
func GetLusCode(patp string) (string, error) {
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
	file := "code"
	hoon := "=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  code=@p  bind:m  (scry @p /j/code/(scot %p our))  (pure:m !>((crip (slag 1 (scow %p code)))))"
	_, code, success, err := executeClickCommandForCode(
		patp,
		file,
		hoon,
		"",
		"code",
		"Click +code",
		ClearLusCode,
	)
	if err != nil {
		storeLusCodeError(patp)
		return "", fmt.Errorf("Click +code failed to get exec: %v", err)
	}
	if !success {
		storeLusCodeError(patp)
		return "", fmt.Errorf("Click +code failed poke")
	}
	storeLusCode(patp, code)
	return code, nil
}

func allowLusCodeRequest(patp string) bool {
	codeMutex.Lock()
	defer codeMutex.Unlock()
	data, exists := lusCodes[patp]
	if !exists {
		return true
	}
	if time.Since(data.LastError) < 1*time.Second {
		return false
	}
	if len(data.LusCode) != 27 {
		return true
	}
	if time.Since(data.LastFetch) > 15*time.Minute {
		return true
	}
	return false
}

func storeLusCodeError(patp string) {
	zap.L().Debug(fmt.Sprintf("Recording +code failure for %s", patp))
	codeMutex.Lock()
	defer codeMutex.Unlock()
	lusCodes[patp] = structs.ClickLusCode{
		LastError: time.Now(),
	}
}

func storeLusCode(patp, code string) {
	zap.L().Info(fmt.Sprintf("Storing +code for %s", patp))
	codeMutex.Lock()
	defer codeMutex.Unlock()
	lusCodes[patp] = structs.ClickLusCode{
		LastFetch: time.Now(),
		LusCode:   code,
	}
}
