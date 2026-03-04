package devsvc

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"groundseg/backupsvc"
	"groundseg/click"
	"groundseg/config"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/system"

	"go.uber.org/zap"
)

type Dependencies struct {
	IsDevMode                bool
	ListHardDisks            func() (structs.LSBLKDevice, error)
	Conf                     func() structs.SysConfig
	CreateLocalBackup        func(patp, backupRoot string) error
	UploadLatestBackup       func(patp, pw, backupRoot string) error
	RestoreBackupWithRequest func(startram.RestoreBackupRequest) error
	Retrieve                 func() (structs.StartramRetrieve, error)
	Now                      func() time.Time
	UrbitConf                func(string) structs.UrbitDocker
	SendNotification         func(string, structs.HarkNotification) error
	UpdateConfTyped          func(...config.ConfUpdateOption) error
	WithStartramReminderAll  func(bool) config.ConfUpdateOption
	BackupDir                string
}

var (
	isDevFn                        = checkDevMode
	backupDirFn                    = func() string { return backupsvc.ResolveBackupRoot(config.BasePath()) }
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
		if arg == "dev" {
			return true
		}
	}
	return false
}

func CheckDevMode() bool {
	return isDevFn()
}

func DevHandler(msg []byte) error {
	return Handler(msg, Dependencies{
		IsDevMode:                isDevFn(),
		ListHardDisks:            listHardDisksForDev,
		Conf:                     confForDev,
		CreateLocalBackup:        createLocalBackupForDev,
		UploadLatestBackup:       uploadLatestBackupForDev,
		RestoreBackupWithRequest: restoreBackupWithRequestForDev,
		Retrieve:                 retrieveStartramForDev,
		Now:                      nowForDev,
		UrbitConf:                urbitConfForDev,
		SendNotification:         sendNotificationForDev,
		UpdateConfTyped:          updateConfTypedForDev,
		WithStartramReminderAll:  withStartramReminderAllForDev,
		BackupDir:                backupDirFn(),
	})
}

func Handler(msg []byte, deps Dependencies) error {
	if !deps.IsDevMode {
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
		if deps.ListHardDisks == nil {
			return nil
		}
		if blockDevices, err := deps.ListHardDisks(); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to print block mounts: %v", err))
		} else {
			zap.L().Debug(fmt.Sprintf("lsblk: %+v", blockDevices))
		}
	case "backup-tlon":
		conf := deps.Conf()
		for _, patp := range conf.Connectivity.Piers {
			if err := deps.CreateLocalBackup(patp, deps.BackupDir); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to backup tlon for %v", err))
			}
		}
	case "remote-backup-tlon":
		conf := deps.Conf()
		for _, patp := range conf.Connectivity.Piers {
			if err := deps.UploadLatestBackup(patp, conf.Connectivity.RemoteBackupPassword, deps.BackupDir); err != nil {
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
			if err := deps.RestoreBackupWithRequest(req); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to restore backup for %s: %v", patp, err))
			}
		}
	case "startram-reminder":
		conf := deps.Conf()
		if !conf.Connectivity.WgRegistered {
			return fmt.Errorf("No startram registration")
		}
		retrieve, err := deps.Retrieve()
		if err != nil {
			return fmt.Errorf("Failed to retrieve StarTram information: %w", err)
		}

		layout := "2006-01-02"
		expiryDate, err := time.Parse(layout, retrieve.Lease)
		if err != nil {
			return fmt.Errorf("Failed to parse expiry date %v: %w", retrieve.Lease, err)
		}

		currentTime := deps.Now()
		diff := expiryDate.Sub(currentTime).Hours() / 24
		noti := structs.HarkNotification{Type: "startram-reminder", StartramDaysLeft: int(diff)}

		rem := conf.Startram.StartramSetReminder
		if !rem.One || !rem.Three || !rem.Seven {
			for _, patp := range conf.Connectivity.Piers {
				shipConf := deps.UrbitConf(patp)
				if shipConf.StartramReminder == true {
					if err := deps.SendNotification(patp, noti); err != nil {
						zap.L().Error(fmt.Sprintf("Failed to send dev startram reminder to %s: %v", patp, err))
					}
				}
			}
		} else {
			zap.L().Debug("Dev not sending startram reminder. Already reminded!")
		}
	case "startram-reminder-toggle":
		reminded := devPayload.Payload.Reminded
		if err := deps.UpdateConfTyped(deps.WithStartramReminderAll(reminded)); err != nil {
			return fmt.Errorf("Couldn't reset startram reminder: %w", err)
		}
	default:
		return fmt.Errorf("Unknown Dev action: %v", devPayload.Payload.Action)
	}
	return nil
}

func ResetDevSeams() {
	isDevFn = checkDevMode
	backupDirFn = func() string { return backupsvc.ResolveBackupRoot(config.BasePath()) }
	listHardDisksForDev = system.ListHardDisks
	confForDev = config.Conf
	createLocalBackupForDev = backupsvc.CreateLocalBackup
	uploadLatestBackupForDev = backupsvc.UploadLatestBackup
	restoreBackupWithRequestForDev = startram.RestoreBackupWithRequest
	retrieveStartramForDev = startram.Retrieve
	nowForDev = time.Now
	urbitConfForDev = config.UrbitConf
	sendNotificationForDev = click.SendNotification
	updateConfTypedForDev = config.UpdateConfTyped
	withStartramReminderAllForDev = config.WithStartramReminderAll
}
