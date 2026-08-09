package config

import (
	"groundseg/defaults"
	"path/filepath"
	"strings"
)

// write a hardcoded default conf to disk
func CreateDefaultBeszelConf() error {
	defaultConfig := defaults.BeszelConfig
	path := filepath.Join(BasePath, "settings", "beszel.json")
	return writeJSONDurably(path, &defaultConfig, 0o644)
}

// UpdateBeszelConf writes the monitoring config from version server info.
func UpdateBeszelConf() error {
	newConfig := defaults.BeszelConfig
	versionInfo := VersionInfo.Beszel
	if strings.TrimSpace(versionInfo.Repo) != "" {
		newConfig.Repo = versionInfo.Repo
		newConfig.BeszelVersion = versionInfo.Tag
		newConfig.Amd64Sha256 = versionInfo.Amd64Sha256
		newConfig.Arm64Sha256 = versionInfo.Arm64Sha256
	}
	path := filepath.Join(BasePath, "settings", "beszel.json")
	return writeJSONDurably(path, &newConfig, 0o644)
}
