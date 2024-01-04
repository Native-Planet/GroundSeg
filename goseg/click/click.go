package click

import (
	"fmt"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/structs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

var (
	lusCodes   = make(map[string]structs.ClickLusCode)
	shipDesks  = make(map[string]map[string]structs.ClickDesks)
	codeMutex  sync.Mutex
	desksMutex sync.Mutex
)

func createHoon(patp, file, hoon string) error {
	dockerDir := config.DockerDir
	hoonFile := filepath.Join(dockerDir, patp, "_data", fmt.Sprintf("%s.hoon", file))
	if err := ioutil.WriteFile(hoonFile, []byte(hoon), 0644); err != nil {
		return err
	}
	ClearLusCode(patp)
	return nil
}

func ClearLusCode(patp string) {
	codeMutex.Lock()
	defer codeMutex.Unlock()
	delete(lusCodes, patp)
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

// Exports

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
func GetDesk(patp, desk string, bypass bool) (string, error) {
	if !bypass {
		if !allowDeskRequest(patp, desk) {
			status, err := fetchDeskFromMemory(patp, desk)
			return status, err
		}
	}
	// <file>.hoon
	file := "desk-" + desk
	// actual hoon
	hoon := "=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  now=@da  bind:m  get-time  (pure:m !>((crip ~(ram re [%rose [~ ~ ~] (report-vats our now [%" + desk + " %kids ~] %$ |)]))))"
	// create hoon file
	if err := createHoon(patp, file, hoon); err != nil {
		return "", fmt.Errorf("Click get desk %%%v failed to create hoon: %v", desk, err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file, "/sur/hood/hoon")
	if err != nil {
		storeDeskError(patp, desk)
		return "", fmt.Errorf("Click get desk %%%v failed to get exec: %v", desk, err)
	}
	// retrieve +vats
	vats, _, err := filterResponse("desk", response)
	if err != nil {
		storeDeskError(patp, desk)
		return "", fmt.Errorf("Click penpai desk info failed to get exec: %v", err)
	}
	storeDesk(patp, desk, vats)
	return vats, nil
}
