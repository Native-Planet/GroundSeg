package config

import (
	"encoding/json"
	"goseg/defaults"
	"goseg/structs"
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
	releaseChannel := conf.UpdateBranch
	mcRepo := VersionInfo.Miniomc.Repo
	amdHash := VersionInfo.Miniomc.Amd64Sha256
	armHash := VersionInfo.Miniomc.Arm64Sha256
	newConfig := structs.McConfig{
		McName:      "minio_client",
		McVersion:   releaseChannel,
		Repo:        mcRepo,
		Amd64Sha256: amdHash,
		Arm64Sha256: armHash,
	}
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
	if err := encoder.Encode(&newConfig); err != nil {
		return err
	}
	return nil
}
