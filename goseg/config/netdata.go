package config

import (
	"encoding/json"
	"groundseg/defaults"
	"groundseg/structs"
	"os"
	"path/filepath"
)

// write a hardcoded default conf to disk
func CreateDefaultNetdataConf() error {
	defaultConfig := defaults.NetdataConfig
	path := filepath.Join(BasePath, "settings", "netdata.json")
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
func UpdateNetdataConf() error {
	conf := Conf()
	releaseChannel := conf.UpdateBranch
	netdataRepo := VersionInfo.Netdata.Repo
	amdHash := VersionInfo.Netdata.Amd64Sha256
	armHash := VersionInfo.Netdata.Arm64Sha256
	newConfig := structs.NetdataConfig{
		NetdataName:    "netdata",
		Repo:           netdataRepo,
		NetdataVersion: releaseChannel,
		Amd64Sha256:    amdHash,
		Arm64Sha256:    armHash,
		CapAdd:         []string{"SYS_PTRACE"},
		Port:           19999,
		Restart:        "unless-stopped",
		SecurityOpt:    "apparmor=unconfined",
		Volumes: []string{
			"netdataconfig:/etc/netdata",
			"netdatalib:/var/lib/netdata",
			"netdatacache:/var/cache/netdata",
			"/etc/passwd:/host/etc/passwd:ro",
			"/etc/group:/host/etc/group:ro",
			"/proc:/host/proc:ro",
			"/sys:/host/sys:ro",
			"/etc/os-release:/host/etc/os-release:ro",
		},
	}
	path := filepath.Join(BasePath, "settings", "netdata.json")
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
