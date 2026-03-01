package backupsvc

import (
	"fmt"
	"groundseg/backups"
	"groundseg/startram"
	"groundseg/system"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type LocalDirs struct {
	Ship    string
	Daily   string
	Weekly  string
	Monthly string
}

var (
	isMountedMMCFn = system.IsMountedMMC
	mkdirAllFn     = os.MkdirAll
	readDirFn      = os.ReadDir
	createBackupFn = backups.CreateBackup
	uploadBackupFn = startram.UploadBackup
)

func ResolveBackupRoot(basePath string) string {
	mmc, err := isMountedMMCFn(basePath)
	if err == nil && mmc {
		return "/media/data/backup"
	}
	return filepath.Join(basePath, "backup")
}

func LocalDirsForShip(backupRoot, patp string) LocalDirs {
	ship := filepath.Join(backupRoot, patp)
	return LocalDirs{
		Ship:    ship,
		Daily:   filepath.Join(ship, "daily"),
		Weekly:  filepath.Join(ship, "weekly"),
		Monthly: filepath.Join(ship, "monthly"),
	}
}

func EnsureLocalDirs(backupRoot, patp string) (LocalDirs, error) {
	dirs := LocalDirsForShip(backupRoot, patp)
	for _, dir := range []string{dirs.Ship, dirs.Daily, dirs.Weekly, dirs.Monthly} {
		if err := mkdirAllFn(dir, 0755); err != nil {
			return LocalDirs{}, fmt.Errorf("create backup directory for %s: %w", patp, err)
		}
	}
	return dirs, nil
}

func CreateLocalBackup(patp, backupRoot string) error {
	dirs, err := EnsureLocalDirs(backupRoot, patp)
	if err != nil {
		return err
	}
	if err := createBackupFn(patp, dirs.Daily, dirs.Weekly, dirs.Monthly); err != nil {
		return fmt.Errorf("create local backup for %s: %w", patp, err)
	}
	return nil
}

func LatestBackupFile(backupRoot, patp string) (string, error) {
	dirs := LocalDirsForShip(backupRoot, patp)
	candidateDirs := []string{dirs.Daily, dirs.Weekly, dirs.Monthly, dirs.Ship}
	var latestTimestamp int64
	var latestPath string
	for _, dir := range candidateDirs {
		entries, err := readDirFn(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("read backup directory %s: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			timestamp, err := strconv.ParseInt(entry.Name(), 10, 64)
			if err != nil {
				continue
			}
			if timestamp > latestTimestamp {
				latestTimestamp = timestamp
				latestPath = filepath.Join(dir, entry.Name())
			}
		}
	}
	if latestPath == "" {
		return "", fmt.Errorf("no valid backup files found for %s", patp)
	}
	return latestPath, nil
}

func UploadLatestBackup(patp, password, backupRoot string) error {
	latestBackup, err := LatestBackupFile(backupRoot, patp)
	if err != nil {
		return fmt.Errorf("resolve latest backup for %s: %w", patp, err)
	}
	if err := uploadBackupFn(patp, password, latestBackup); err != nil {
		return fmt.Errorf("upload latest backup for %s: %w", patp, err)
	}
	return nil
}

func MostRecentDailyBackupTime(backupRoot, patp string) (time.Time, error) {
	dailyDir := LocalDirsForShip(backupRoot, patp).Daily
	entries, err := readDirFn(dailyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("read daily backup directory %s: %w", dailyDir, err)
	}
	var latestTimestamp int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		timestamp, err := strconv.ParseInt(entry.Name(), 10, 64)
		if err != nil {
			continue
		}
		if timestamp > latestTimestamp {
			latestTimestamp = timestamp
		}
	}
	if latestTimestamp == 0 {
		return time.Time{}, nil
	}
	return time.Unix(latestTimestamp, 0), nil
}
