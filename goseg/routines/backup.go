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

type BackupTime struct {
	IsSet bool
	Time  time.Time
}

func TlonBackupRemote() {
	backupTime := BackupTime{IsSet: false}
	for {
		if !backupTime.IsSet {
			backupTime = BackupTime{IsSet: true, Time: generateTimeOfDay(config.StartramConfig.UrlID)}
			config.BackupTime = backupTime.Time
		}
		now := time.Now()
		if now.Equal(backupTime.Time) || (now.After(backupTime.Time) && now.Sub(backupTime.Time) <= time.Hour) {
			conf := config.Conf()
			if conf.RemoteBackupPassword != "" && conf.WgRegistered {
				zap.L().Info("Time to backup ships remotely")
				for _, patp := range conf.Piers {
					zap.L().Info(fmt.Sprintf("Backing up %s", patp))
					shipBackupDir := filepath.Join(BackupDir, patp)
					// List all files in shipBackupDir
					files, err := filepath.Glob(filepath.Join(shipBackupDir, "*"))
					if err != nil {
						zap.L().Error(fmt.Sprintf("Failed to list backup files for %s: %v", patp, err))
						continue
					}

					var latestTimestamp int64
					var latestFile string

					// Find the file with the most recent timestamp
					for _, file := range files {
						baseName := filepath.Base(file)
						timestamp, err := strconv.ParseInt(baseName, 10, 64)
						if err != nil {
							// Skip files that are not valid timestamps
							continue
						}
						if timestamp > latestTimestamp {
							latestTimestamp = timestamp
							latestFile = file
						}
					}

					var backupFile string
					if latestFile != "" {
						backupFile = latestFile
					} else {
						zap.L().Warn(fmt.Sprintf("No valid backup files found for %s", patp))
						continue
					}
					err = startram.UploadBackup(patp, conf.RemoteBackupPassword, backupFile)
					if err != nil {
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
			shipBackupDir := filepath.Join(BackupDir, patp)
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
				err := backups.CreateBackup(patp, shipBackupDirDaily, shipBackupDirWeekly, shipBackupDirMonthly)
				if err != nil {
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
