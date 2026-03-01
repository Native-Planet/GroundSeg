package backups

import (
	"archive/tar"
	"bytes"
	"fmt"
	backupdomain "groundseg/click/backup"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"go.uber.org/zap"
)

var (
	backupTlonFn = backupdomain.BackupTlon
	nowFn        = time.Now
	getVolumeFn  = func(patp string) (string, error) {
		cmd := exec.Command("docker", "inspect", "-f", "{{ range .Mounts }}{{ if eq .Type \"volume\" }}{{ .Source }}{{ end }}{{ end }}", patp)
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(output)), nil
	}
)

func CreateBackup(patp, shipBackupDirDaily, shipBackupDirWeekly, shipBackupDirMonthly string) error {
	err := backupTlonFn(patp)
	if err != nil {
		return fmt.Errorf("routine failed to backup tlon: %w", err)
	}
	// Get the Docker volume location for the ship
	volumePath, err := getVolumeFn(patp)
	if err != nil {
		return fmt.Errorf("failed to get Docker volume location: %w", err)
	}

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

		info, err := file.Stat()
		if err != nil {
			file.Close()
			return fmt.Errorf("failed to get file info: %w", err)
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}

		header.Name = filepath.Base(jamFile)

		if err := tw.WriteHeader(header); err != nil {
			file.Close()
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		if _, err := io.Copy(tw, file); err != nil {
			file.Close()
			return fmt.Errorf("failed to write file to tar: %w", err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("failed to close jam file: %w", err)
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
	now := nowFn()
	timestamp := strconv.FormatInt(now.Unix(), 10)
	filenameDaily := filepath.Join(shipBackupDirDaily, timestamp)     // tar.zst
	filenameWeekly := filepath.Join(shipBackupDirWeekly, timestamp)   // tar.zst
	filenameMonthly := filepath.Join(shipBackupDirMonthly, timestamp) // tar.zst

	// Write compressed data to file
	if err := os.WriteFile(filenameDaily, compressedData, 0644); err != nil {
		return fmt.Errorf("failed to write compressed backup: %w", err)
	}
	// if timestamp falls on sunday, also write to weekly
	if now.Weekday() == time.Sunday {
		if err := os.WriteFile(filenameWeekly, compressedData, 0644); err != nil {
			return fmt.Errorf("failed to write compressed backup: %w", err)
		}
	}
	// if timestamp falls on first day of month, also write to monthly
	if now.Day() == 1 {
		if err := os.WriteFile(filenameMonthly, compressedData, 0644); err != nil {
			return fmt.Errorf("failed to write compressed backup: %w", err)
		}
	}

	if err := pruneBackups(shipBackupDirDaily, 3); err != nil {
		return fmt.Errorf("failed to prune daily backups: %w", err)
	}
	if err := pruneBackups(shipBackupDirWeekly, 3); err != nil {
		return fmt.Errorf("failed to prune weekly backups: %w", err)
	}
	if err := pruneBackups(shipBackupDirMonthly, 3); err != nil {
		return fmt.Errorf("failed to prune monthly backups: %w", err)
	}

	return nil
}

func backupTimestampFromName(name string) int64 {
	value, err := strconv.ParseInt(name, 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func pruneBackups(backupDir string, keep int) error {
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool {
		iTime := backupTimestampFromName(files[i].Name())
		jTime := backupTimestampFromName(files[j].Name())
		if iTime == jTime {
			return files[i].Name() > files[j].Name()
		}
		return iTime > jTime
	})
	for i := keep; i < len(files); i++ {
		oldBackup := filepath.Join(backupDir, files[i].Name())
		if err := os.Remove(oldBackup); err != nil {
			zap.L().Warn("Failed to remove old backup", zap.String("file", oldBackup), zap.Error(err))
		}
	}
	return nil
}
