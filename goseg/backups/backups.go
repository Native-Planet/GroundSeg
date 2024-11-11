package backups

import (
	"archive/tar"
	"bytes"
	"fmt"
	"groundseg/click"
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

func CreateBackup(patp, shipBackupDir string) error {
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
