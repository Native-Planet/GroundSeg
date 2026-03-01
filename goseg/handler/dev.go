package handler

import (
	"encoding/json"
	"fmt"
	"groundseg/backupsvc"
	"groundseg/click"
	"groundseg/config"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/system"
	"os"
	"time"

	"go.uber.org/zap"
)

var isDev = checkDevMode()

var (
	BackupDir = backupsvc.ResolveBackupRoot(config.BasePath)
)

var (
	listHardDisksForDev            = system.ListHardDisks
	confForDev                     = config.Conf
	createLocalBackupForDev        = backupsvc.CreateLocalBackup
	uploadLatestBackupForDev       = backupsvc.UploadLatestBackup
	restoreBackupWithRequestForDev = startram.RestoreBackupWithRequest
	retrieveStartramForDev         = startram.Retrieve
	nowForDev                      = time.Now
	urbitConfForDev                = config.UrbitConf
	sendNotificationForDev         = click.SendNotification
	updateConfTypedForDev          = config.UpdateConfTyped
	withStartramReminderAllForDev  = config.WithStartramReminderAll
)

func checkDevMode() bool {
	for _, arg := range os.Args[1:] {
		// trigger dev mode with `./groundseg dev`
		if arg == "dev" {
			return true
		}
	}
	return false
}

func DevHandler(msg []byte) error {
	if !isDev {
		return fmt.Errorf("Dev actions not allowed!")
	}
	var devPayload structs.WsDevPayload
	err := json.Unmarshal(msg, &devPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal dev payload: %w", err)
	}
	switch devPayload.Payload.Action {
	case "reset-setup":
		zap.L().Warn("Dev reset-setup not allowed!")
	case "print-mounts":
		if blockDevices, err := listHardDisksForDev(); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to print block mounts: %v", err))
		} else {
			zap.L().Debug(fmt.Sprintf("lsblk: %+v", blockDevices))
		}
	case "backup-tlon":
		conf := confForDev()
		for _, patp := range conf.Piers {
			if err := createLocalBackupForDev(patp, BackupDir); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to backup tlon for %v", err))
			}
		}
	case "remote-backup-tlon":
		conf := confForDev()
		for _, patp := range conf.Piers {
			if err := uploadLatestBackupForDev(patp, conf.RemoteBackupPassword, BackupDir); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to upload backup for %s: %v", patp, err))
			}
		}
	case "restore-tlon":
		patp := devPayload.Payload.Patp
		remote := devPayload.Payload.Remote
		if !remote {
			zap.L().Debug(fmt.Sprintf("Skipping local restore for %s for now....", patp))
		} else {
			req := startram.RestoreBackupRequest{
				Ship:      patp,
				Timestamp: 0,
				Mode:      startram.RestoreBackupModeDevelopment,
				Source:    startram.RestoreBackupSourceRemote,
			}
			if err := restoreBackupWithRequestForDev(req); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to restore backup for %s: %v", patp, err))
			}
		}
	case "startram-reminder":
		conf := confForDev()
		if !conf.WgRegistered {
			return fmt.Errorf("No startram registration")
		}
		retrieve, err := retrieveStartramForDev()
		if err != nil {
			return fmt.Errorf("Failed to retrieve StarTram information: %w", err)
		}
		//retrieve.Ongoing != 0
		// Layout to parse the given date (must match the format of dateStr)
		layout := "2006-01-02"

		// Parse the date string into a time.Time object
		expiryDate, err := time.Parse(layout, retrieve.Lease)
		if err != nil {
			return fmt.Errorf("Failed to parse expiry date %v: %w", retrieve.Lease, err)
		}

		// Get the current time
		currentTime := nowForDev()

		// Calculate the difference in days
		diff := expiryDate.Sub(currentTime).Hours() / 24
		noti := structs.HarkNotification{Type: "startram-reminder", StartramDaysLeft: int(diff)}

		rem := conf.StartramSetReminder
		if !rem.One || !rem.Three || !rem.Seven {
			// Send notification
			for _, patp := range conf.Piers {
				shipConf := urbitConfForDev(patp)
				if shipConf.StartramReminder == true {
					if err := sendNotificationForDev(patp, noti); err != nil {
						zap.L().Error(fmt.Sprintf("Failed to send dev startram reminder to %s: %v", patp, err))
					}
				}
			}
		} else {
			zap.L().Debug("Dev not sending startram reminder. Already reminded!")
		}
	case "startram-reminder-toggle":
		reminded := devPayload.Payload.Reminded
		if err := updateConfTypedForDev(withStartramReminderAllForDev(reminded)); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't reset startram reminder: %v", err))
		}
	default:
		return fmt.Errorf("Unknown Dev action: %v", devPayload.Payload.Action)
	}
	return nil
}
