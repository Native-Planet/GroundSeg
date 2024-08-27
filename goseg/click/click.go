package click

import (
	"fmt"
	"groundseg/structs"
	"strings"
	"sync"
)

var (
	lusCodes   = make(map[string]structs.ClickLusCode)
	shipDesks  = make(map[string]map[string]structs.ClickDesks)
	codeMutex  sync.Mutex
	desksMutex sync.Mutex
)

// Exports

// acme.go
func FixAcme(patp string) error { return fixAcme(patp) }

// code.go
func ClearLusCode(patp string)               { clearLusCode(patp) }
func GetLusCode(patp string) (string, error) { return getLusCode(patp) }

// desk.go
func ReviveDesk(patp, desk string) error                     { return reviveDesk(patp, desk) }
func UninstallDesk(patp, desk string) error                  { return uninstallDesk(patp, desk) }
func InstallDesk(patp, ship, desk string) error              { return installDesk(patp, ship, desk) }
func GetDesk(patp, desk string, bypass bool) (string, error) { return getDesk(patp, desk, bypass) }

// exit.go
func BarExit(patp string) error { return barExit(patp) }

// hark.go
func SendNotification(patp string, payload structs.HarkNotification) error {
	switch payload.Type {
	case "startram-reminder":
		return sendStartramReminder(patp, payload.StartramDaysLeft)
	case "disk-warning":
		return sendDiskSpaceWarning(patp, payload.DiskName, payload.DiskUsage)
	case "smart-fail":
		return sendSmartWarning(patp, payload.DiskName)
		/*
			case "cpu-temperature":
				return sendCPUTempWarning(patp)
		*/
	default:
		return fmt.Errorf("invalid hark notification type: %s", payload.Type)
	}
}

// pack.go
func SendPack(patp string) error { return sendPack(patp) }

// storage.go
func UnlinkStorage(patp string) error { return unlinkStorage(patp) }
func LinkStorage(patp, endpoint string, svcAccount structs.MinIOServiceAccount) error {
	return linkStorage(patp, endpoint, svcAccount)
}

// backup.go
func BackupTlon(patp string) error {
	var errors []string

	components := []struct {
		name string
		err  error
	}{
		{"activity", backupAgent(patp, "activity")},
		{"channels", backupAgent(patp, "channels")},
		{"channels-server", backupAgent(patp, "channels-server")},
		{"groups", backupAgent(patp, "groups")},
		{"profile", backupAgent(patp, "profile")},
		{"chat", backupAgent(patp, "chat")},
	}

	for _, component := range components {
		if component.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", component.name, component.err))
		}
	}

	if len(errors) == 0 {
		return nil // No errors, return nil
	}

	return fmt.Errorf("backup errors: %s", strings.Join(errors, ", "))
}
