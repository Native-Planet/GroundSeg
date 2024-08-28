package routines

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"io"
	"math/big"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
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
		}
		now := time.Now()
		if now.Equal(backupTime.Time) || (now.After(backupTime.Time) && now.Sub(backupTime.Time) <= time.Hour) {
			zap.L().Info("run remote backup placeholder")
			// check latest remote backup for each ship
			// if latest remote backup is older than 24 hours, run remote backup
			// else do nothing
		}

		time.Sleep(15 * time.Second) // temp
	}
}
func TlonBackupLocal() {
	for {
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
			if err := os.MkdirAll(shipBackupDir, 0755); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to create backup directory for %v: %v", patp, err))
				continue
			}
			// parse backup time
			parsedTime, err := time.ParseInLocation(timeFormat, shipConf.BackupTime, location)
			if err == nil {
				backupTime = parsedTime
			}
			// List all files in shipBackupDir that are unix timestamps
			files, err := filepath.Glob(filepath.Join(shipBackupDir, "[0-9]*"))
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

				// Create backup (placeholder function)
				err := createBackup(patp, shipBackupDir)
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

func createBackup(patp, shipBackupDir string) error {
	err := click.BackupTlon(patp)
	if err != nil {
		return fmt.Errorf("routine failed to backup tlon: %w", err)
	}
	// Get the Docker volume location for the ship
	cmd := exec.Command("docker", "inspect", "-f", "{{ range .Mounts }}{{ if eq .Type \"volume\" }}{{ .Source }}{{ end }}{{ end }}", patp)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get Docker volume location: %w", err)
	}

	volumePath := strings.TrimSpace(string(output))
	if volumePath == "" {
		return fmt.Errorf("no Docker volume found for container %s", patp)
	}
	putDir := filepath.Join(volumePath, patp, ".urb", "put")

	// Find all .jam files in putDir
	jamFiles, err := filepath.Glob(filepath.Join(putDir, "*.jam"))
	if err != nil {
		return fmt.Errorf("failed to find .jam files: %w", err)
	}

	if len(jamFiles) == 0 {
		return fmt.Errorf("no .jam files found in %s", putDir)
	}

	// Create a buffer to hold the tar content
	var tarBuffer bytes.Buffer

	// Create a new tar writer that writes to the buffer
	tw := tar.NewWriter(&tarBuffer)

	// Add each .jam file to the tar
	for _, jamFile := range jamFiles {
		file, err := os.Open(jamFile)
		if err != nil {
			return fmt.Errorf("failed to open jam file: %w", err)
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			return fmt.Errorf("failed to get file info: %w", err)
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}

		header.Name = filepath.Base(jamFile)

		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		if _, err := io.Copy(tw, file); err != nil {
			return fmt.Errorf("failed to write file to tar: %w", err)
		}
	}

	// Close the tar writer
	if err := tw.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Compress the tar data using zstd
	compressor, err := zstd.NewWriter(nil)
	if err != nil {
		return fmt.Errorf("failed to create zstd compressor: %w", err)
	}
	compressedData := compressor.EncodeAll(tarBuffer.Bytes(), nil)
	compressor.Close()

	// Generate filename with current Unix timestamp
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	filename := filepath.Join(shipBackupDir, timestamp) // tar.zst

	// Write compressed data to file
	if err := os.WriteFile(filename, compressedData, 0644); err != nil {
		return fmt.Errorf("failed to write compressed backup: %w", err)
	}

	// Remove old backups, keeping only the most recent 3
	files, err := os.ReadDir(shipBackupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Sort files by name (timestamp) in descending order
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() > files[j].Name()
	})

	// Keep the first 3 files (most recent) and remove the rest
	for i := 3; i < len(files); i++ {
		oldBackup := filepath.Join(shipBackupDir, files[i].Name())
		if err := os.Remove(oldBackup); err != nil {
			zap.L().Warn("Failed to remove old backup", zap.String("file", oldBackup), zap.Error(err))
		}
	}

	return nil
}
