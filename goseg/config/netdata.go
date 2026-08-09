package config

import (
	"groundseg/defaults"
	"path/filepath"
	"strings"
)

// write a hardcoded default conf to disk
func CreateDefaultNetdataConf() error {
	defaultConfig := defaults.NetdataConfig
	path := filepath.Join(BasePath, "settings", "netdata.json")
	return writeJSONDurably(path, &defaultConfig, 0o644)
}

// write a conf to disk from version server info
func UpdateNetdataConf() error {
	newConfig := defaults.NetdataConfig
	versionInfo := VersionInfo.Netdata
	if strings.Contains(versionInfo.Repo, "henrygd/beszel") {
		newConfig.Repo = versionInfo.Repo
		newConfig.NetdataVersion = versionInfo.Tag
		newConfig.Amd64Sha256 = versionInfo.Amd64Sha256
		newConfig.Arm64Sha256 = versionInfo.Arm64Sha256
	}
	path := filepath.Join(BasePath, "settings", "netdata.json")
	return writeJSONDurably(path, &newConfig, 0o644)
}
