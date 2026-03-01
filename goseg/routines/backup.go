package routines

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"groundseg/backupsvc"
	"groundseg/config"
	"math/big"
	"time"

	"go.uber.org/zap"
)

var (
	BackupDir                    = backupsvc.ResolveBackupRoot(config.BasePath)
	createLocalBackupForRoutine  = backupsvc.CreateLocalBackup
	uploadLatestBackupForRoutine = backupsvc.UploadLatestBackup
	latestDailyBackupForRoutine  = backupsvc.MostRecentDailyBackupTime
)

type BackupTime struct {
	IsSet bool
	Time  time.Time
}

func TlonBackupRemote() {
	backupTime := BackupTime{IsSet: false}
	for {
		if !backupTime.IsSet {
			snapshot := config.GetStartramConfigSnapshot()
			if !snapshot.Fresh || snapshot.Value.UrlID == "" {
				zap.L().Debug("Remote backup schedule waiting for fresh StarTram config snapshot")
				time.Sleep(30 * time.Second)
				continue
			}
			backupTime = BackupTime{IsSet: true, Time: generateTimeOfDay(snapshot.Value.UrlID)}
			config.BackupTime = backupTime.Time
		}
		now := time.Now()
		if now.Equal(backupTime.Time) || (now.After(backupTime.Time) && now.Sub(backupTime.Time) <= time.Hour) {
			conf := config.Conf()
			if conf.RemoteBackupPassword != "" && conf.WgRegistered {
				zap.L().Info("Time to backup ships remotely")
				for _, patp := range conf.Piers {
					zap.L().Info(fmt.Sprintf("Backing up %s", patp))
					if err := uploadLatestBackupForRoutine(patp, conf.RemoteBackupPassword, BackupDir); err != nil {
						zap.L().Error(fmt.Sprintf("Failed to upload backup for %v: %v", patp, err))
					}
				}
			}
		}
		time.Sleep(1 * time.Minute)
	}
}
func TlonBackupLocal() {
	for {
		zap.L().Info("Checking local backups")
		conf := config.Conf()
		for _, patp := range conf.Piers {
			// get local tz
			location := time.Now().Location()
			// default backup time is midnight
			backupTime := time.Date(0, 1, 1, 0, 0, 0, 0, location)
			// time format
			timeFormat := "1504"
			// retrieve config
			shipConf := config.UrbitConf(patp)
			// parse backup time
			parsedTime, err := time.ParseInLocation(timeFormat, shipConf.BackupTime, location)
			if err == nil {
				backupTime = parsedTime
			}
			mostRecentBackup, err := latestDailyBackupForRoutine(BackupDir, patp)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Failed to inspect backups for %v: %v", patp, err))
				continue
			}

			zap.L().Debug(fmt.Sprintf("Most recent backup for %v: %v", patp, mostRecentBackup))

			// Get the current time
			now := time.Now()

			// Check if it's time for a backup
			if now.After(mostRecentBackup.AddDate(0, 0, 1)) && // Check if at least a day has passed since the last backup
				(now.Hour() > backupTime.Hour() || // Current hour is later than backup hour
					(now.Hour() == backupTime.Hour() && now.Minute() >= backupTime.Minute())) { // Current hour is equal and minute is equal or later

				// Create backup
				if err := createLocalBackupForRoutine(patp, BackupDir); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to create backup for %v: %v", patp, err))
				} else {
					zap.L().Info(fmt.Sprintf("Successfully created backup for %v", patp))
				}
			}
		}
		time.Sleep(1 * time.Hour) // check every hour
	}
}

func generateTimeOfDay(input string) time.Time {
	// modulos
	mod24 := big.NewInt(24)
	mod60 := big.NewInt(60)
	// time maker
	makeTime := func(text string, mod *big.Int) int64 {
		// get hash
		hashed := sha256.Sum256([]byte(text))
		hex := hex.EncodeToString(hashed[:])
		// to big int
		bigInt := new(big.Int)
		bigInt.SetString(hex, 16)
		// mod and convert to int64
		return new(big.Int).Mod(bigInt, mod).Int64()
	}
	hour := makeTime(input+"hour", mod24)
	minute := makeTime(input+"minute", mod60)
	second := makeTime(input+"second", mod60)
	// Construct a time.Time object with the generated hour, minute, and second
	generatedTime := time.Date(0, time.January, 1, int(hour), int(minute), int(second), 0, time.UTC)
	return generatedTime
}
