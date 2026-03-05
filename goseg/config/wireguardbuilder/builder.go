package wireguardbuilder

import (
	"groundseg/defaults"
	"groundseg/structs"
)

func BuildConfig(conf structs.SysConfig, versionInfo structs.Channel) structs.WgConfig {
	wgConfig := defaults.DefaultWgConfig()
	wgConfig.WireguardVersion = conf.Connectivity.UpdateBranch
	wgConfig.Repo = versionInfo.Wireguard.Repo
	wgConfig.Amd64Sha256 = versionInfo.Wireguard.Amd64Sha256
	wgConfig.Arm64Sha256 = versionInfo.Wireguard.Arm64Sha256
	wgConfig.CapAdd = append([]string(nil), defaults.DefaultWgConfig().CapAdd...)
	wgConfig.Volumes = append([]string(nil), defaults.DefaultWgConfig().Volumes...)
	return wgConfig
}
