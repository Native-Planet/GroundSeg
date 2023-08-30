package defaults

// default config json structs
// these get written to disk in absence of version server

import (
	"goseg/structs"
	"path/filepath"
)

var (
	NetdataConfig = structs.NetdataConfig{
		NetdataName:    "netdata",
		Repo:           "registry.hub.docker.com/netdata/netdata",
		NetdataVersion: "latest",
		Amd64Sha256:    "95e74c36f15091bcd7983ee162248f1f91c21207c235fce6b0d6f8ed9a11732a",
		Arm64Sha256:    "cd3dc9d182a4561b162f03c6986f4647bbb704f8e7e4872ee0611b1b9e86e1b0",
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

	McConfig = structs.McConfig{
		McName:      "minio_client",
		McVersion:   "latest",
		Repo:        "registry.hub.docker.com/minio/mc",
		Amd64Sha256: "6ffd76764e8ca484de12c6ecaa352db3d8efd5c9d44f393718b29b6600e0a559",
		Arm64Sha256: "6825aecd2f123c9d4408e660aba8a72f9e547a3774350b8f4d2d9b674e99e424",
	}

	WgConfig = structs.WgConfig{
		WireguardName:    "wireguard",
		WireguardVersion: "latest",
		Repo:             "registry.hub.docker.com/linuxserver/wireguard",
		Amd64Sha256:      "ae6f8e8cc1303bc9c0b5fa1b1ef4176c25a2c082e29bf8b554ce1196731e7db2",
		Arm64Sha256:      "403d741b1b5bcf5df1e48eab0af8038355fae3e29419ad5980428f9aebd1576c",
		CapAdd:           []string{"NET_ADMIN", "SYS_MODULE"},
		Volumes:          []string{"/lib/modules:/lib/modules"},
		Sysctls: struct {
			NetIpv4ConfAllSrcValidMark int `json:"net.ipv4.conf.all.src_valid_mark"`
		}{
			NetIpv4ConfAllSrcValidMark: 1,
		},
	}
)

// this one needs params from config so we use a func
func SysConfig(basePath string) structs.SysConfig {
	sysConfig := structs.SysConfig{
		Setup:        "start",
		EndpointUrl:  "api.startram.io",
		ApiVersion:   "v1",
		Piers:        []string{},
		NetCheck:     "1.1.1.1:53",
		UpdateMode:   "auto",
		UpdateUrl:    "https://version.groundseg.app",
		UpdateBranch: "latest",
		SwapVal:      16,
		SwapFile:     filepath.Join(basePath, "settings", "swapfile"),
		KeyFile:      filepath.Join(basePath, "settings", "session.key"),
		Sessions: struct {
			Authorized   map[string]structs.SessionInfo `json:"authorized"`
			Unauthorized map[string]structs.SessionInfo `json:"unauthorized"`
		}{
			Authorized:   make(map[string]structs.SessionInfo),
			Unauthorized: make(map[string]structs.SessionInfo),
		},
		LinuxUpdates: struct {
			Value    int    `json:"value"`
			Interval string `json:"interval"`
			Previous bool   `json:"previous"`
		}{
			Value:    1,
			Interval: "week",
			Previous: false,
		},
		DockerData:     "/var/lib/docker",
		WgOn:           false,
		WgRegistered:   false,
		PwHash:         "",
		C2cInterval:    0,
		FirstBoot:      true,
		GsVersion:      "v2.0.0",
		CfgDir:         basePath,
		UpdateInterval: 0,
		BinHash:        "",
		Pubkey:         "",
		Privkey:        "",
		Salt:           "",
	}
	return sysConfig
}
