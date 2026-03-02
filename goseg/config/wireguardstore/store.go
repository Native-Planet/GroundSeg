package wireguardstore

import (
	"encoding/json"
	"fmt"
	"groundseg/structs"
	"io"
	"os"
	"path/filepath"
)

type WireguardConfigStore interface {
	Load(path string) (structs.WgConfig, error)
	Save(path string, config structs.WgConfig) error
	EnsureDir(path string) error
}

type FileStore struct{}

func (FileStore) Load(path string) (structs.WgConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return structs.WgConfig{}, err
	}
	defer func() {
		_ = file.Close()
	}()

	contents, err := io.ReadAll(file)
	if err != nil {
		return structs.WgConfig{}, err
	}
	var config structs.WgConfig
	if err := json.Unmarshal(contents, &config); err != nil {
		return structs.WgConfig{}, err
	}
	return config, nil
}

func (FileStore) Save(path string, config structs.WgConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create WireGuard settings directory %s: %w", filepath.Dir(path), err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create WireGuard config %s: %w", path, err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(&config); err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to write WireGuard config %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close WireGuard config %s: %w", path, err)
	}
	return nil
}

func (FileStore) EnsureDir(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}
