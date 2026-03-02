package desk

import (
	"fmt"
	"sync"
	"time"

	"groundseg/click/internal/runtime"
	"groundseg/click/luscode"
	"groundseg/structs"

	"go.uber.org/zap"
)

var (
	clearLusCode               = luscode.ClearLusCode
	executeClickCommandForDesk = runtime.ExecuteCommandWithSuccess
	parseClickResponseForDesk  = runtime.ExecuteCommandWithResponse

	shipDesks  = make(map[string]map[string]structs.ClickDesks)
	desksMutex sync.Mutex
)

func ReviveDesk(patp, desk string) error {
	return reviveDesk(patp, desk)
}
func UninstallDesk(patp, desk string) error {
	return uninstallDesk(patp, desk)
}
func InstallDesk(patp, ship, desk string) error {
	return installDesk(patp, ship, desk)
}
func GetDesk(patp, desk string, bypass bool) (string, error) {
	return getDesk(patp, desk, bypass)
}
func MountDesk(patp, desk string) error {
	return mountDesk(patp, desk)
}
func CommitDesk(patp, desk string) error {
	return commitDesk(patp, desk)
}

func reviveDesk(patp, desk string) error {
	file := "revive-desk"
	hoon := fmt.Sprintf("=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  ~  bind:m  (poke [our %%hood] %%kiln-revive !>(%%%v))  (pure:m !>('success'))", desk)
	_, err := executeClickCommandForDesk(patp, file, hoon, "", "success", fmt.Sprintf("Click |revive %%%v", desk), nil)
	return err
}

func uninstallDesk(patp, desk string) error {
	file := "uninstall-desk"
	hoon := fmt.Sprintf("=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  ~  bind:m  (poke [our %%hood] %%kiln-uninstall !>(%%%v))  (pure:m !>('success'))", desk)
	_, err := executeClickCommandForDesk(patp, file, hoon, "", "success", fmt.Sprintf("Click |uninstall %%%v", desk), nil)
	return err
}

func installDesk(patp, ship, desk string) error {
	file := "install-desk"
	hoon := fmt.Sprintf("=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  ~  bind:m  (poke [our %%hood] %%kiln-install !>([%%%v %v %%%v]))  (pure:m !>('success'))", desk, ship, desk)
	_, err := executeClickCommandForDesk(patp, file, hoon, "", "success", fmt.Sprintf("Click |install %v %%%v", ship, desk), nil)
	return err
}

func getDesk(patp, desk string, bypass bool) (string, error) {
	if !bypass {
		if !allowDeskRequest(patp, desk) {
			return fetchDeskFromMemory(patp, desk)
		}
	}
	file := "desk-" + desk
	hoon := "=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-our  ;<  now=@da  bind:m  get-time  (pure:m !>((crip ~(ram re [%rose [~ ~ ~] (report-vats our now [%" + desk + " %kids ~] %$ |)]))))"
	_, deskStatus, success, err := parseClickResponseForDesk(
		patp,
		file,
		hoon,
		"/sur/hood/hoon",
		"desk",
		fmt.Sprintf("Click get desk %%%v", desk),
		clearLusCode,
	)
	if err != nil {
		storeDeskError(patp, desk)
		return "", fmt.Errorf("Click penpai desk info failed to get exec: %v", err)
	}
	if !success {
		storeDeskError(patp, desk)
		return "", fmt.Errorf("%s poke failed", fmt.Sprintf("Click get desk %%%v", desk))
	}
	storeDesk(patp, desk, deskStatus)
	return deskStatus, nil
}

func mountDesk(patp, desk string) error {
	file := "mount-" + desk
	hoon := fmt.Sprintf("=/  m  (strand ,vase)  ;<  our=@p  bind:m  get-time  ;<  ~  bind:m  (poke-our %%hood %%kiln-mount !>([(en-beam [our %%%v [%%da now]] /) %%%v]))  (pure:m !>('success'))", desk, desk)
	_, err := executeClickCommandForDesk(
		patp,
		file,
		hoon,
		"/sur/hood/hoon",
		"success",
		fmt.Sprintf("Click |mount %%%v", desk),
		clearLusCode,
	)
	if err != nil {
		storeDeskError(patp, desk)
		return err
	}
	return nil
}

func commitDesk(patp, desk string) error {
	file := "commit-" + desk
	hoon := fmt.Sprintf("=/  m  (strand ,vase)  ;<  ~  bind:m  (poke-our %%hood %%kiln-commit !>([[%%%v] %%.n]))  (pure:m !>('success'))", desk)
	_, err := executeClickCommandForDesk(
		patp,
		file,
		hoon,
		"/sur/hood/hoon",
		"success",
		fmt.Sprintf("Click |commit %%%v", desk),
		clearLusCode,
	)
	if err != nil {
		storeDeskError(patp, desk)
		return err
	}
	return nil
}

func allowDeskRequest(patp, desk string) bool {
	desksMutex.Lock()
	defer desksMutex.Unlock()
	deskInfo, exists := shipDesks[patp]
	if !exists {
		return true
	}
	data, exists := deskInfo[desk]
	if !exists {
		return true
	}
	if time.Since(data.LastError) < 1*time.Second {
		return false
	}
	if time.Since(data.LastFetch) > 2*time.Minute {
		return true
	}
	return false
}

func fetchDeskFromMemory(patp, desk string) (string, error) {
	desksMutex.Lock()
	defer desksMutex.Unlock()
	shipDesk, exists := shipDesks[patp]
	if !exists {
		return "", fmt.Errorf("Click desk request for %%%v failed to fetch patp from memory for %v", desk, patp)
	}
	data, exists := shipDesk[desk]
	if !exists {
		return "", fmt.Errorf("Click desk request for %%%v failed to fetch desk from memory for %v", desk, patp)
	}
	return data.Status, nil
}

func storeDeskError(patp, desk string) {
	zap.L().Debug(fmt.Sprintf("Recording penpai desk info failure for %s", patp))
	desksMutex.Lock()
	defer desksMutex.Unlock()
	deskInfo, exists := shipDesks[patp]
	if !exists {
		deskInfo = make(map[string]structs.ClickDesks)
	}
	deskInfo[desk] = structs.ClickDesks{
		LastError: time.Now(),
	}
	shipDesks[patp] = deskInfo
}

func storeDesk(patp, desk, deskStatus string) {
	zap.L().Info(fmt.Sprintf("Storing %%%v desk status for %s", desk, patp))
	desksMutex.Lock()
	defer desksMutex.Unlock()
	deskInfo, exists := shipDesks[patp]
	if !exists {
		deskInfo = make(map[string]structs.ClickDesks)
	}
	deskInfo[desk] = structs.ClickDesks{
		LastFetch: time.Now(),
		Status:    deskStatus,
	}
	shipDesks[patp] = deskInfo
}
