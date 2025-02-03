package config

import (
	"encoding/json"
	"fmt"
	"groundseg/defaults"
	"groundseg/structs"
	"os"
	"path/filepath"
)

// write a hardcoded default conf to disk
func CreateDefaultMcConf() error {
	defaultConfig := defaults.McConfig
	path := filepath.Join(BasePath, "settings", "mc.json")
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(&defaultConfig); err != nil {
		return err
	}
	return nil
}

// write a conf to disk from version server info
func UpdateMcConf() error {
	conf := Conf()
	newConfig := structs.McConfig{
		McName:      "minio_client",
		McVersion:   conf.UpdateBranch,
		Repo:        VersionInfo.Miniomc.Repo,
		Amd64Sha256: VersionInfo.Miniomc.Amd64Sha256,
		Arm64Sha256: VersionInfo.Miniomc.Arm64Sha256,
	}
	path := filepath.Join(BasePath, "settings", "mc.json")
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return fmt.Errorf("error creating directories: %v", err)
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(path), "mc.json.*")
	if err != nil {
		return fmt.Errorf("error creating temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(&newConfig); err != nil {
		tmpFile.Close()
		return fmt.Errorf("error encoding config: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("error closing temp file: %v", err)
	}
	if fi, err := os.Stat(tmpPath); err != nil {
		return fmt.Errorf("error checking temp file: %v", err)
	} else if fi.Size() == 0 {
		return fmt.Errorf("refusing to persist empty configuration")
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("error moving temp file: %v", err)
	}
	return nil
}
