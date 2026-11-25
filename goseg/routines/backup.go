package routines

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"groundseg/backups"
	"groundseg/config"
	"groundseg/startram"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/disk"
	"go.uber.org/zap"
)

var (
	BackupDir = setBackupDir()
)

func TlonBackupRemote() {
	var lastAttempt time.Time
	for {
		now := time.Now()
		backupTime := generateTimeOfDay(config.StartramConfig.UrlID, now)
		config.BackupTime = backupTime
		conf := config.Conf()
		if conf.RemoteBackupPassword != "" && conf.WgRegistered && remoteBackupDue(backupTime, lastAttempt, now) {
			lastAttempt = now
			zap.L().Info("Time to backup ships remotely")
			uploadScheduledBackups(BackupDir, conf.Piers, conf.RemoteBackupPassword, startram.UploadBackup)
		}
		time.Sleep(time.Minute)
	}
}

func uploadScheduledBackups(root string, piers []string, password string, upload func(ship, password, file string) error) {
	for _, patp := range piers {
		if !config.UrbitConf(patp).RemoteTlonBackup {
			continue
		}
		backupFile, err := latestDailyBackup(root, patp)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to list backup files for %s: %v", patp, err))
			continue
		}
		if backupFile == "" {
			zap.L().Warn(fmt.Sprintf("No valid backup files found for %s", patp))
			continue
		}
		if err := upload(patp, password, backupFile); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to upload backup for %v: %v", patp, err))
		}
	}
}

// Each daily window permits one upload attempt per process, including across UTC midnight.
func remoteBackupDue(scheduled, lastAttempt, now time.Time) bool {
	if now.Before(scheduled) {
		scheduled = scheduled.AddDate(0, 0, -1)
	}
	return now.Sub(scheduled) <= time.Hour && lastAttempt.Before(scheduled)
}

func latestDailyBackup(root, ship string) (string, error) {
	dir := filepath.Join(root, ship, "daily")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var latestTimestamp int64
	var latestFile string
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		timestamp, err := strconv.ParseInt(entry.Name(), 10, 64)
		if err == nil && timestamp > latestTimestamp {
			latestTimestamp = timestamp
			latestFile = filepath.Join(dir, entry.Name())
		}
	}
	return latestFile, nil
}

func TlonBackupLocal() {
	for {
		zap.L().Info("Checking local backups")
		createScheduledBackups(BackupDir, config.Conf().Piers, backups.CreateBackup)
		time.Sleep(1 * time.Hour) // check every hour
	}
}

func createScheduledBackups(root string, piers []string, create func(ship, daily, weekly, monthly string) error) {
	for _, patp := range piers {
		shipConf := config.UrbitConf(patp)
		if !shipConf.LocalTlonBackup {
			continue
		}
		// get local tz
		location := time.Now().Location()
		// default backup time is midnight
		backupTime := time.Date(0, 1, 1, 0, 0, 0, 0, location)
		// time format
		timeFormat := "1504"
		shipBackupDir := filepath.Join(root, patp)
		shipBackupDirDaily := filepath.Join(shipBackupDir, "daily")
		shipBackupDirWeekly := filepath.Join(shipBackupDir, "weekly")
		shipBackupDirMonthly := filepath.Join(shipBackupDir, "monthly")
		if err := os.MkdirAll(shipBackupDirDaily, 0755); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to create backup directory for %v: %v", patp, err))
			continue
		}
		if err := os.MkdirAll(shipBackupDirWeekly, 0755); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to create backup directory for %v: %v", patp, err))
			continue
		}
		if err := os.MkdirAll(shipBackupDirMonthly, 0755); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to create backup directory for %v: %v", patp, err))
			continue
		}
		// parse backup time
		parsedTime, err := time.ParseInLocation(timeFormat, shipConf.BackupTime, location)
		if err == nil {
			backupTime = parsedTime
		}
		// List all files in shipBackupDir that are unix timestamps
		files, err := filepath.Glob(filepath.Join(shipBackupDirDaily, "[0-9]*"))
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to list backup files for %v: %v", patp, err))
			continue
		}

		// check if backups exists, if exists get latest timestamp
		var mostRecentBackup time.Time
		if len(files) > 0 {
			// Find the most recent backup
			var latestUnixTime int64
			for _, file := range files {
				fileName := filepath.Base(file)
				unixTime, err := strconv.ParseInt(fileName, 10, 64)
				if err == nil && unixTime > latestUnixTime {
					latestUnixTime = unixTime
				}
			}

			// Convert the most recent unix timestamp to time.Time
			if latestUnixTime > 0 {
				mostRecentBackup = time.Unix(latestUnixTime, 0)
			}
		}

		// If no valid backups found, set mostRecentBackup to zero time
		if mostRecentBackup.IsZero() {
			mostRecentBackup = time.Time{}
		}

		zap.L().Debug(fmt.Sprintf("Most recent backup for %v: %v", patp, mostRecentBackup))

		// Get the current time
		now := time.Now()

		// Check if it's time for a backup
		if now.After(mostRecentBackup.AddDate(0, 0, 1)) && // Check if at least a day has passed since the last backup
			(now.Hour() > backupTime.Hour() || // Current hour is later than backup hour
				(now.Hour() == backupTime.Hour() && now.Minute() >= backupTime.Minute())) { // Current hour is equal and minute is equal or later

			// Create backup
			err := create(patp, shipBackupDirDaily, shipBackupDirWeekly, shipBackupDirMonthly)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Failed to create backup for %v: %v", patp, err))
			} else {
				zap.L().Info(fmt.Sprintf("Successfully created backup for %v", patp))
			}
		}
	}
}

func generateTimeOfDay(input string, now time.Time) time.Time {
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
	now = now.UTC()
	generatedTime := time.Date(now.Year(), now.Month(), now.Day(), int(hour), int(minute), int(second), 0, time.UTC)
	return generatedTime
}

func setBackupDir() string {
	mmc, _ := isMountedMMC(config.BasePath)
	if mmc {
		return "/media/data/backup"
	} else {
		return filepath.Join(config.BasePath, "backup")
	}

}

func isMountedMMC(dirPath string) (bool, error) {
	partitions, err := disk.Partitions(true)
	if err != nil {
		return false, fmt.Errorf("failed to get list of partitions")
	}
	/*
		the outer loop loops from child up the unix path
		until a mountpoint is found
	*/
OuterLoop:
	for {
		for _, p := range partitions {
			if p.Mountpoint == dirPath {
				devType := "mmc"
				if strings.Contains(p.Device, devType) {
					return true, nil
				} else {
					break OuterLoop
				}
			}
		}
		if dirPath == "/" {
			break
		}
		dirPath = path.Dir(dirPath) // Reduce the path by one level
	}
	return false, nil
}
