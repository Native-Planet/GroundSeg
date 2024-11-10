package handler

import (
	"encoding/json"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/system"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go.uber.org/zap"
)

var isDev = checkDevMode()

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
		return fmt.Errorf("Couldn't unmarshal dev payload: %v", err)
	}
	switch devPayload.Payload.Action {
	case "reset-setup":
		zap.L().Warn("Dev reset-setup not allowed!")
	case "print-mounts":
		if blockDevices, err := system.ListHardDisks(); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to print block mounts: %v", err))
		} else {
			zap.L().Debug(fmt.Sprintf("lsblk: %+v", blockDevices))
		}
	case "backup-tlon":
		conf := config.Conf()
		for _, patp := range conf.Piers {
			if err := click.BackupTlon(patp); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to backup tlon for %v", err))
			}
		}
	case "remote-backup-tlon":
		conf := config.Conf()
		for _, patp := range conf.Piers {
			shipBak := filepath.Join(config.BasePath, "backup", patp)
			files, err := os.ReadDir(shipBak)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Failed to read backup directory for %s: %v", patp, err))
				continue
			}

			var latestTimestamp int64
			var latestFile string

			for _, file := range files {
				if !file.IsDir() {
					timestamp, err := strconv.ParseInt(file.Name(), 10, 64)
					if err == nil && timestamp > latestTimestamp {
						latestTimestamp = timestamp
						latestFile = file.Name()
					}
				}
			}

			if latestFile != "" {
				latestBak := filepath.Join(shipBak, latestFile)
				if err := startram.UploadBackup(patp, conf.RemoteBackupPassword, latestBak); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to upload backup for %s: %v", patp, err))
				}
			}
		}
	case "restore-tlon":
		patp := devPayload.Payload.Patp
		remote := devPayload.Payload.Remote
		if !remote {
			zap.L().Debug(fmt.Sprintf("Skipping local restore for %s for now....", patp))
		} else {
			if err := startram.RestoreBackup(patp, remote, 0, "", true); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to restore backup for %s: %v", patp, err))
			}
		}
	case "startram-reminder":
		conf := config.Conf()
		if !conf.WgRegistered {
			return fmt.Errorf("No startram registration")
		}
		retrieve, err := startram.Retrieve()
		if err != nil {
			return fmt.Errorf("Failed to retrieve StarTram information: %v", err)
		}
		//retrieve.Ongoing != 0
		// Layout to parse the given date (must match the format of dateStr)
		layout := "2006-01-02"

		// Parse the date string into a time.Time object
		expiryDate, err := time.Parse(layout, retrieve.Lease)
		if err != nil {
			return fmt.Errorf("Failed to parse expiry date %v: %v", retrieve.Lease, err)
		}

		// Get the current time
		currentTime := time.Now()

		// Calculate the difference in days
		diff := expiryDate.Sub(currentTime).Hours() / 24
		noti := structs.HarkNotification{Type: "startram-reminder", StartramDaysLeft: int(diff)}

		rem := conf.StartramSetReminder
		if !rem.One || !rem.Three || !rem.Seven {
			// Send notification
			for _, patp := range conf.Piers {
				shipConf := config.UrbitConf(patp)
				if shipConf.StartramReminder == true {
					if err := click.SendNotification(patp, noti); err != nil {
						zap.L().Error(fmt.Sprintf("Failed to send dev startram reminder to %s: %v", patp, err))
					}
				}
			}
		} else {
			zap.L().Debug("Dev not sending startram reminder. Already reminded!")
		}
	case "startram-reminder-toggle":
		reminded := devPayload.Payload.Reminded
		if err := config.UpdateConf(map[string]interface{}{
			"startramSetReminder": map[string]bool{
				"one":   reminded,
				"three": reminded,
				"seven": reminded,
			},
		}); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't reset startram reminder: %v", err))
		}
	default:
		return fmt.Errorf("Unknown Dev action: %v", devPayload.Payload.Action)
	}
	return nil
}
