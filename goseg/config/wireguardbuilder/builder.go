package wireguardbuilder

import (
	"groundseg/defaults"
	"groundseg/structs"
)

func BuildConfig(conf structs.SysConfig, versionInfo structs.Channel) structs.WgConfig {
	wgConfig := defaults.WgConfig
	wgConfig.WireguardVersion = conf.UpdateBranch
	wgConfig.Repo = versionInfo.Wireguard.Repo
	wgConfig.Amd64Sha256 = versionInfo.Wireguard.Amd64Sha256
	wgConfig.Arm64Sha256 = versionInfo.Wireguard.Arm64Sha256
	wgConfig.CapAdd = append([]string(nil), defaults.WgConfig.CapAdd...)
	wgConfig.Volumes = append([]string(nil), defaults.WgConfig.Volumes...)
	return wgConfig
}
