package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// writeFileDurably atomically replaces path and does not return until both the
// new contents and the directory entry have been flushed to stable storage.
// If power is lost before the rename, the previous file remains intact.
func writeFileDurably(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	// Make a newly-created config directory durable before placing a file in it.
	if err := syncDirectory(filepath.Dir(dir)); err != nil {
		return fmt.Errorf("sync config parent directory: %w", err)
	}

	fileMode := mode.Perm()
	if info, err := os.Stat(path); err == nil {
		fileMode = info.Mode().Perm()
		existing, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read existing config: %w", err)
		}
		if bytes.Equal(existing, data) {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat existing config: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	closeWithError := func(writeErr error) error {
		if closeErr := tmpFile.Close(); writeErr == nil {
			writeErr = closeErr
		}
		return writeErr
	}
	if err := tmpFile.Chmod(fileMode); err != nil {
		return fmt.Errorf("set temporary config permissions: %w", closeWithError(err))
	}
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("write temporary config: %w", closeWithError(err))
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("sync temporary config: %w", closeWithError(err))
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	if err := syncDirectory(dir); err != nil {
		return fmt.Errorf("sync config directory: %w", err)
	}
	return nil
}

func writeJSONDurably(path string, value any, mode os.FileMode) error {
	data, err := json.MarshalIndent(value, "", "    ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')
	return writeFileDurably(path, data, mode)
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
