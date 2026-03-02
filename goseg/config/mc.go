package config

import (
	"encoding/json"
	"groundseg/defaults"
	"groundseg/structs"
	"path/filepath"
)

var (
	confForMc              = Conf
	getVersionChannelForMc = GetVersionChannel
	pathForMc              = func() string { return filepath.Join(BasePath(), "settings", "mc.json") }
) 

// write a hardcoded default conf to disk
func CreateDefaultMcConf() error {
	defaultConfig := defaults.McConfig
	path := pathForMc()
	rawConfig, err := json.MarshalIndent(defaultConfig, "", "    ")
	if err != nil {
		return err
	}
	return persistConfigJSON(path, rawConfig)
}

// write a conf to disk from version server info
func UpdateMcConf() error {
	conf := confForMc()
	versionInfo := getVersionChannelForMc()
	newConfig := structs.McConfig{
		McName:      "minio_client",
		McVersion:   conf.UpdateBranch,
		Repo:        versionInfo.Miniomc.Repo,
		Amd64Sha256: versionInfo.Miniomc.Amd64Sha256,
		Arm64Sha256: versionInfo.Miniomc.Arm64Sha256,
	}
	path := pathForMc()
	rawConfig, err := json.MarshalIndent(newConfig, "", "    ")
	if err != nil {
		return err
	}
	return persistConfigJSON(path, rawConfig)
}
