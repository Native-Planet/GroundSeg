package click

import (
	"fmt"
	"groundseg/structs"
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
func BackupActivity(patp string) error       { return backupActivity(patp) }
func BackupChannels(patp string) error       { return backupChannels(patp) }
func BackupChannelsServer(patp string) error { return backupChannelsServer(patp) }
func BackupGroups(patp string) error         { return backupGroups(patp) }
func BackupProfile(patp string) error        { return backupProfile(patp) }
func BackupChat(patp string) error           { return backupChat(patp) }
