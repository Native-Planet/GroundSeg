package click

import (
	"fmt"
	"goseg/logger"
	"goseg/structs"
	"time"
)

func allowDeskRequest(patp, desk string) bool {
	desksMutex.Lock()
	defer desksMutex.Unlock()
	// if patp doesn't exist
	deskInfo, exists := shipDesks[patp]
	if !exists {
		return true
	}
	data, exists := deskInfo[desk]
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
	// use the desk status stored
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
	logger.Logger.Debug(fmt.Sprintf("Recording penpai desk info failure for %s", patp))
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
	logger.Logger.Info(fmt.Sprintf("Storing %%%v desk status for %s", desk, patp))
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
