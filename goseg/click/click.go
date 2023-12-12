package click

import (
	"fmt"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/logger"
	"groundseg/structs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	lusCodes    = make(map[string]structs.ClickLusCode)
	shipDesks   = make(map[string]structs.ClickPenpaiDesk)
	codeMutex   sync.Mutex
	penpaiMutex sync.Mutex
)

func BarExit(patp string) error {
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
	ClearLusCode(patp)
	return nil
}

func ClearLusCode(patp string) {
	codeMutex.Lock()
	defer codeMutex.Unlock()
	delete(lusCodes, patp)
}

func SendPack(patp string) error {
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

func ReviveDesk(patp, desk string) error {
	// <file>.hoon
	file := "revive-desk"
	// actual hoon
	hoon := fmt.Sprintf("=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  ~  bind:m  (poke [our %%hood] %%kiln-revive !>(%%%v))  (pure:m !>('success'))", desk)
	// create hoon file
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click |revive %%%v failed to create hoon: %v", desk, err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file, "")
	if err != nil {
		return fmt.Errorf("Click |revive %%%v failed to get exec: %v", desk, err)
	}
	// retrieve code
	_, success, err := filterResponse("success", response)
	if err != nil {
		return fmt.Errorf("Click |revive %%%v failed to get exec: %v", desk, err)
	}
	if !success {
		return fmt.Errorf("Click |revive %%%v poke failed", desk)
	}
	return nil
}

func UninstallDesk(patp, desk string) error {
	// <file>.hoon
	file := "uninstall-desk"
	// actual hoon
	hoon := fmt.Sprintf("=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  ~  bind:m  (poke [our %%hood] %%kiln-uninstall !>(%%%v))  (pure:m !>('success'))", desk)
	// create hoon file
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click |uninstall %%%v failed to create hoon: %v", desk, err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file, "")
	if err != nil {
		return fmt.Errorf("Click |uninstall %%%v failed to get exec: %v", desk, err)
	}
	// retrieve code
	_, success, err := filterResponse("success", response)
	if err != nil {
		return fmt.Errorf("Click |uninstall %%%v failed to get exec: %v", desk, err)
	}
	if !success {
		return fmt.Errorf("Click |uninstall %%%v poke failed", desk)
	}
	return nil
}

func InstallDesk(patp, ship, desk string) error {
	// <file>.hoon
	file := "install-desk"
	// actual hoon
	hoon := fmt.Sprintf("=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  ~  bind:m  (poke [our %%hood] %%kiln-install !>([%%%v %v %%%v]))  (pure:m !>('success'))", desk, ship, desk)
	// create hoon file
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click |install %v %%%v failed to create hoon: %v", ship, desk, err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file, "")
	if err != nil {
		return fmt.Errorf("Click |install %v %%%v failed to get exec: %v", ship, desk, err)
	}
	// retrieve code
	_, success, err := filterResponse("success", response)
	if err != nil {
		return fmt.Errorf("Click |install %v %%%v failed to get exec: %v", ship, desk, err)
	}
	if !success {
		return fmt.Errorf("Click |install %v %%%v poke failed", ship, desk)
	}
	return nil
}

func SetPenpaiDeskLoading(patp string, loading bool) error {
	penpaiMutex.Lock()
	defer penpaiMutex.Unlock()
	penpaiInfo, exists := shipDesks[patp]
	if !exists {
		return fmt.Errorf("Click penpai desk request failed to fetch desks from memory for %v", patp)
	}
	penpaiInfo.Loading = loading
	shipDesks[patp] = penpaiInfo
	return nil
}

func GetPenpaiInstalling(patp string) bool {
	penpaiMutex.Lock()
	defer penpaiMutex.Unlock()
	penpaiInfo, exists := shipDesks[patp]
	if !exists {
		return false
	}
	return penpaiInfo.Loading
}

func GetDesk(patp, desk string, bypass bool) (string, error) {
	if !bypass {
		proceedWithRequest := true
		switch desk {
		case "penpai":
			proceedWithRequest = allowPenpaiDeskRequest(patp)
		case "groundseg":
			//proceedWithRequest = true
			return "running", nil
		default:
			logger.Logger.Warn(fmt.Sprintf("Desk %%%v information is not stored in groundseg. Proceeding with request by default"))
		}
		if !proceedWithRequest {
			penpaiMutex.Lock()
			defer penpaiMutex.Unlock()
			penpaiInfo, exists := shipDesks[patp]
			if !exists {
				return "", fmt.Errorf("Click penpai desk request failed to fetch desks from memory for %v", patp)
			}
			return penpaiInfo.Status, nil
		}
	}
	// <file>.hoon
	file := "desk-" + desk
	// actual hoon
	hoon := "=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  now=@da  bind:m  get-time  (pure:m !>((crip ~(ram re [%rose [~ ~ ~] (report-vats our now [%" + desk + " %kids ~] %$ |)]))))"
	// create hoon file
	if err := createHoon(patp, file, hoon); err != nil {
		return "", fmt.Errorf("Click +vats failed to create hoon: %v", err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file, "/sur/hood/hoon")
	if err != nil {
		storePenpaiDeskError(patp)
		return "", fmt.Errorf("Click +code failed to get exec: %v", err)
		logger.Logger.Warn(fmt.Sprintf("error %v", err)) // temp
		return "", err
	}
	// retrieve +vats
	vats, _, err := filterResponse("desk", response)
	if err != nil {
		storePenpaiDeskError(patp)
		return "", fmt.Errorf("Click penpai desk info failed to get exec: %v", err)
	}
	storePenpaiDesk(patp, vats)
	return vats, nil
}

// Get +code from Urbit
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

func storePenpaiDeskError(patp string) {
	logger.Logger.Debug(fmt.Sprintf("Recording penpai desk info failure for %s", patp))
	penpaiMutex.Lock()
	defer penpaiMutex.Unlock()
	shipDesks[patp] = structs.ClickPenpaiDesk{
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

func storePenpaiDesk(patp, deskStatus string) {
	logger.Logger.Info(fmt.Sprintf("Storing penpai desk status for %s", patp))
	penpaiMutex.Lock()
	defer penpaiMutex.Unlock()
	deskInfo, exists := shipDesks[patp]
	if !exists {
		shipDesks[patp] = structs.ClickPenpaiDesk{
			LastFetch: time.Now(),
			Status:    deskStatus,
		}
	} else {
		deskInfo.Status = deskStatus
		deskInfo.LastFetch = time.Now()
		shipDesks[patp] = deskInfo
	}
}

func allowPenpaiDeskRequest(patp string) bool {
	penpaiMutex.Lock()
	defer penpaiMutex.Unlock()
	// if patp doesn't exist
	data, exists := shipDesks[patp]
	if !exists {
		return true
	}
	// flood control
	if time.Since(data.LastError) < 1*time.Second {
		return false
	}
	// if it has been 2 minutes
	if time.Since(data.LastFetch) > 2*time.Minute {
		return true
	}
	// use the penpai desk status stored
	return false
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

func createHoon(patp, file, hoon string) error {
	dockerDir := config.DockerDir
	hoonFile := filepath.Join(dockerDir, patp, "_data", fmt.Sprintf("%s.hoon", file))
	if err := ioutil.WriteFile(hoonFile, []byte(hoon), 0644); err != nil {
		return err
	}
	return nil
}

func deleteHoon(patp, file string) {
	dockerDir := config.DockerDir
	hoonFile := filepath.Join(dockerDir, patp, "_data", fmt.Sprintf("%s.hoon", file))
	if _, err := os.Stat(hoonFile); !os.IsNotExist(err) {
		os.Remove(hoonFile)
	}
}

func clickExec(patp, file, dependency string) (string, error) {
	execCommand := []string{
		"click",
		"-b",
		"urbit",
		"-kp",
		"-i",
		fmt.Sprintf("%s.hoon", file),
		patp,
		dependency,
	}
	res, err := docker.ExecDockerCommand(patp, execCommand)
	if err != nil {
		return "", err
	}
	return res, nil
}

func filterResponse(resType string, response string) (string, bool, error) {
	responseSlice := strings.Split(response, "\n")
	/*
		example usage:
		code, _, err := filterResponse("code",[]string{"some","response"})
		_, ack, err := filterResponse("pack",[]string{"pack","response"})
	*/
	switch resType {
	case "success": // use this if no value need to be returned
		for _, line := range responseSlice {
			if strings.Contains(line, "[0 %avow 0 %noun %success]") {
				return "", true, nil
			}
		}
		return "", false, nil
	case "code":
		for _, line := range responseSlice {
			if strings.Contains(line, "%avow") {
				// Find the last % before the closing ]
				endIndex := strings.Index(line, "]")
				lastPercentIndex := strings.LastIndex(line[:endIndex], "%")

				if lastPercentIndex != -1 && endIndex != -1 && lastPercentIndex < endIndex {
					// Extract the substring
					code := line[lastPercentIndex+1 : endIndex]
					return code, false, nil
				}
			}
		}
	case "desk":
		for _, line := range responseSlice {
			if strings.Contains(line, "%avow") {
				if strings.Contains(line, "does not yet exist") {
					return "not-found", false, nil
				}
				// Define a regular expression to match "app status" and capture it
				regex := regexp.MustCompile(`app status:\s+([^\s]+)`)
				// Find the first match in the input string
				match := regex.FindStringSubmatch(line)
				// Check if a match was found
				if len(match) >= 2 {
					appStatus := match[1]
					return appStatus, false, nil
				} else {
					return "not-found", false, nil
				}
				return "not found", false, nil
				//}
			}
		}
	case "default":
		return "", false, fmt.Errorf("Unknown poke response")
	}
	return "", false, fmt.Errorf("+code not in poke response")
}
