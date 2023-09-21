package click

import (
	"fmt"
	"goseg/config"
	"goseg/docker"
	"goseg/logger"
	"goseg/structs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	lusCodes  = make(map[string]structs.ClickLusCode)
	codeMutex sync.Mutex
)

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
	logger.Logger.Info(fmt.Sprintf("Allowing +code request for %s", patp))
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
	response, err := clickExec(patp, file)
	if err != nil {
		return "", fmt.Errorf("Click +code failed to get exec: %v", err)
	}
	// retrieve code
	code, _, err := filterResponse("code", response)
	if err != nil {
		return "", fmt.Errorf("Click +code failed to get exec: %v", err)
	}
	storeLusCode(patp, code)
	return code, nil
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

func clickExec(patp, file string) (string, error) {
	execCommand := []string{
		"click",
		"-b",
		"urbit",
		"-kp",
		"-i",
		fmt.Sprintf("%s.hoon", file),
		patp,
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
	case "default":
		return "", false, fmt.Errorf("Unknown poke response")
	}
	return "", false, fmt.Errorf("+code not in poke response")
}
