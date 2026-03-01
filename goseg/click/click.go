package click

import (
	"fmt"
	"groundseg/click/acme"
	backupdomain "groundseg/click/backup"
	"groundseg/structs"
	"strings"
	"sync"
)

var (
	lusCodes   = make(map[string]structs.ClickLusCode)
	shipDesks  = make(map[string]map[string]structs.ClickDesks)
	codeMutex  sync.Mutex
	desksMutex sync.Mutex

	sendStartramReminderFn = sendStartramReminder
	sendDiskSpaceWarningFn = sendDiskSpaceWarning
	sendSmartWarningFn     = sendSmartWarning

	fixAcmeFn      = acme.Fix
	backupTlonFn   = backupdomain.BackupTlon
	restoreAgentFn = restoreAgent
)

// Exports

// acme.go
func FixAcme(patp string) error { return fixAcmeFn(patp) }

// code.go
func ClearLusCode(patp string)               { clearLusCode(patp) }
func GetLusCode(patp string) (string, error) { return getLusCode(patp) }

// desk.go
func ReviveDesk(patp, desk string) error                     { return reviveDesk(patp, desk) }
func UninstallDesk(patp, desk string) error                  { return uninstallDesk(patp, desk) }
func InstallDesk(patp, ship, desk string) error              { return installDesk(patp, ship, desk) }
func GetDesk(patp, desk string, bypass bool) (string, error) { return getDesk(patp, desk, bypass) }
func MountDesk(patp, desk string) error                      { return mountDesk(patp, desk) }
func CommitDesk(patp, desk string) error                     { return commitDesk(patp, desk) }

// exit.go
func BarExit(patp string) error { return barExit(patp) }

// hark.go
func SendNotification(patp string, payload structs.HarkNotification) error {
	switch payload.Type {
	case "startram-reminder":
		return sendStartramReminderFn(patp, payload.StartramDaysLeft)
	case "disk-warning":
		return sendDiskSpaceWarningFn(patp, payload.DiskName, payload.DiskUsage)
	case "smart-fail":
		return sendSmartWarningFn(patp, payload.DiskName)
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

// restore.go
func RestoreTlon(patp string) error {
	var errors []string

	components := []struct {
		name string
		err  error
	}{
		{"activity", restoreAgentFn(patp, "activity")},
		{"channels", restoreAgentFn(patp, "channels")},
		{"channels-server", restoreAgentFn(patp, "channels-server")},
		{"groups", restoreAgentFn(patp, "groups")},
		{"profile", restoreAgentFn(patp, "profile")},
		{"chat", restoreAgentFn(patp, "chat")},
	}

	for _, component := range components {
		if component.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", component.name, component.err))
		}
	}

	if len(errors) == 0 {
		return nil // No errors, return nil
	}

	return fmt.Errorf("restore errors: %s", strings.Join(errors, ", "))
}

// backup.go
func BackupTlon(patp string) error { return backupTlonFn(patp) }
