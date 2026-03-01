package backup

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

const (
	remoteBackupInterval    = 1 * time.Minute
	localBackupInterval     = 1 * time.Hour
	waitForStartramSnapshot = 30 * time.Second
)

var (
	BackupDir                           = backupsvc.ResolveBackupRoot(config.BasePath)
	CreateLocalBackupForRoutine         = backupsvc.CreateLocalBackup
	UploadLatestBackupForRoutine        = backupsvc.UploadLatestBackup
	LatestDailyBackupForRoutine         = backupsvc.MostRecentDailyBackupTime
	GetStartramConfigSnapshotForRoutine = config.GetStartramConfigSnapshot
	ConfForRoutine                      = config.Conf
	UrbitConfForRoutine                 = config.UrbitConf
	SleepForRoutine                     = time.Sleep
	NowForRoutine                       = time.Now
)

type BackupTime struct {
	IsSet bool
	Time  time.Time
}

var remoteBackupState BackupTime

func RunRemoteBackupPass() {
	if !remoteBackupState.IsSet {
		snapshot := GetStartramConfigSnapshotForRoutine()
		if !snapshot.Fresh || snapshot.Value.UrlID == "" {
			zap.L().Debug("Remote backup schedule waiting for fresh StarTram config snapshot")
			SleepForRoutine(waitForStartramSnapshot)
			return
		}
		remoteBackupState = BackupTime{IsSet: true, Time: GenerateTimeOfDay(snapshot.Value.UrlID)}
		config.BackupTime = remoteBackupState.Time
	}

	now := NowForRoutine()
	if now.Equal(remoteBackupState.Time) || (now.After(remoteBackupState.Time) && now.Sub(remoteBackupState.Time) <= time.Hour) {
		conf := ConfForRoutine()
		if conf.RemoteBackupPassword != "" && conf.WgRegistered {
			zap.L().Info("Time to backup ships remotely")
			for _, patp := range conf.Piers {
				zap.L().Info(fmt.Sprintf("Backing up %s", patp))
				if err := UploadLatestBackupForRoutine(patp, conf.RemoteBackupPassword, BackupDir); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to upload backup for %v: %v", patp, err))
				}
			}
		}
	}
}

func RunLocalBackupPass() {
	zap.L().Info("Checking local backups")
	conf := ConfForRoutine()
	for _, patp := range conf.Piers {
		location := NowForRoutine().Location()
		backupTime := time.Date(0, 1, 1, 0, 0, 0, 0, location)
		shipConf := UrbitConfForRoutine(patp)
		if parsedTime, err := time.ParseInLocation("1504", shipConf.BackupTime, location); err == nil {
			backupTime = parsedTime
		}

		mostRecentBackup, err := LatestDailyBackupForRoutine(BackupDir, patp)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to inspect backups for %v: %v", patp, err))
			continue
		}

		zap.L().Debug(fmt.Sprintf("Most recent backup for %v: %v", patp, mostRecentBackup))

		now := NowForRoutine()
		if shouldCreateDailyBackup(backupTime, mostRecentBackup, now) {
			if err := CreateLocalBackupForRoutine(patp, BackupDir); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to create backup for %v: %v", patp, err))
			} else {
				zap.L().Info(fmt.Sprintf("Successfully created backup for %v", patp))
			}
		}
	}
}

func StartBackupRoutines() {
	go TlonBackupRemote()
	go TlonBackupLocal()
}

func TlonBackupRemote() {
	remoteInterval, _ := WaitIntervals()
	runRoutine(remoteInterval, RunRemoteBackupPass, SleepForRoutine)
}

func TlonBackupLocal() {
	_, localInterval := WaitIntervals()
	runRoutine(localInterval, RunLocalBackupPass, SleepForRoutine)
}

func runRoutine(interval time.Duration, fn func(), sleep func(time.Duration)) {
	for {
		fn()
		sleep(interval)
	}
}

func ResetRemoteBackupStateForTest() {
	remoteBackupState = BackupTime{}
}

func GenerateTimeOfDay(input string) time.Time {
	mod24 := big.NewInt(24)
	mod60 := big.NewInt(60)
	makeTime := func(text string, mod *big.Int) int64 {
		hashed := sha256.Sum256([]byte(text))
		hex := hex.EncodeToString(hashed[:])
		bigInt := new(big.Int)
		bigInt.SetString(hex, 16)
		return new(big.Int).Mod(bigInt, mod).Int64()
	}
	hour := makeTime(input+"hour", mod24)
	minute := makeTime(input+"minute", mod60)
	second := makeTime(input+"second", mod60)
	return time.Date(0, time.January, 1, int(hour), int(minute), int(second), 0, time.UTC)
}

func shouldCreateDailyBackup(backupTime, mostRecentBackup, now time.Time) bool {
	return now.After(mostRecentBackup.AddDate(0, 0, 1)) &&
		(now.Hour() > backupTime.Hour() || (now.Hour() == backupTime.Hour() && now.Minute() >= backupTime.Minute()))
}

func WaitIntervals() (remote time.Duration, local time.Duration) {
	return remoteBackupInterval, localBackupInterval
}
