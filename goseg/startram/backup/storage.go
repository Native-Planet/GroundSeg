package backup

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/klauspost/compress/zstd"
	"groundseg/system"
)

const (
	defaultRestoreRoot = "/opt/nativeplanet/groundseg/restore"
	defaultPerm        = 0o755
)

// ResolveLocalBackupPath resolves a backup file path for local retrieval.
func ResolveLocalBackupPath(basePath, ship, bakType string, timestamp int) string {
	mmc, _ := system.IsMountedMMC(basePath)
	if mmc {
		basePath = "/media/data/backup"
	} else {
		basePath = filepath.Join(basePath, "backup")
	}
	return filepath.Join(basePath, ship, bakType, strconv.Itoa(timestamp))
}

// ReadLocalBackup reads a backup from disk for the provided ship.
func ReadLocalBackup(basePath, ship, bakType string, timestamp int) ([]byte, error) {
	backupFile := ResolveLocalBackupPath(basePath, ship, bakType, timestamp)
	file, err := os.Stat(backupFile)
	if err != nil {
		return nil, fmt.Errorf("backup file does not exist: %s", backupFile)
	}
	if file.IsDir() {
		return nil, fmt.Errorf("backup is a directory: %s", backupFile)
	}
	data, err := os.ReadFile(backupFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}
	return data, nil
}

// PersistRemoteBackup writes a decrypted remote backup payload to a restore staging file.
func PersistRemoteBackup(ship string, timestamp int, data []byte) error {
	restoreDir := filepath.Join(defaultRestoreRoot, ship)
	if err := os.MkdirAll(restoreDir, defaultPerm); err != nil {
		return fmt.Errorf("failed to create restore directory: %w", err)
	}
	restoreFile := filepath.Join(restoreDir, strconv.Itoa(timestamp))
	if err := os.WriteFile(restoreFile, data, 0o644); err != nil {
		return fmt.Errorf("failed to write backup to file: %w", err)
	}
	return nil
}

// WriteBackupToVolume extracts a compressed backup archive directly into the ship volume.
func WriteBackupToVolume(volumeMountsCmdOutput, ship string, data []byte) error {
	volumePath := strings.TrimSpace(volumeMountsCmdOutput)
	if volumePath == "" {
		return fmt.Errorf("no Docker volume found for container %s", ship)
	}
	deskDir := filepath.Join(volumePath, ship, "base")
	marDir := filepath.Join(deskDir, "mar")
	bakDir := filepath.Join(deskDir, "bak")
	if _, err := os.Stat(marDir); os.IsNotExist(err) {
		if err := os.MkdirAll(marDir, defaultPerm); err != nil {
			return fmt.Errorf("failed to create mar directory: %w", err)
		}
	}
	if _, err := os.Stat(bakDir); os.IsNotExist(err) {
		if err := os.MkdirAll(bakDir, defaultPerm); err != nil {
			return fmt.Errorf("failed to create backup directory: %w", err)
		}
	}
	decoder, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create zstd decoder: %w", err)
	}
	defer decoder.Close()
	tr := tar.NewReader(decoder)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		target := filepath.Join(bakDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, defaultPerm); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", target, err)
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %v", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to write file %s: %v", target, err)
			}
			f.Close()
		}
	}
	return nil
}
