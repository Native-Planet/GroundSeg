package config

import (
	"fmt"
	"groundseg/defaults"
	"groundseg/structs"
	"path/filepath"
)

// write a hardcoded default conf to disk
func CreateDefaultMcConf() error {
	defaultConfig := defaults.McConfig
	path := filepath.Join(BasePath, "settings", "mc.json")
	return writeJSONDurably(path, &defaultConfig, 0o644)
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
	if err := writeJSONDurably(path, &newConfig, 0o644); err != nil {
		return fmt.Errorf("error persisting mc config: %v", err)
	}
	return nil
}
