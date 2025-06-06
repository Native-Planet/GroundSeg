package defaults

// default config json structs
// these get written to disk in absence of version server

import (
	"groundseg/structs"
	"os"
	"path/filepath"
	"strings"
)

var (
	UrbitConfig = structs.UrbitDocker{
		PierName:            "",
		HTTPPort:            0,
		AmesPort:            0,
		LoomSize:            31,
		UrbitVersion:        "v2.11",
		MinioVersion:        "latest",
		UrbitRepo:           "registry.hub.docker.com/nativeplanet/urbit",
		MinioRepo:           "registry.hub.docker.com/minio/minio",
		UrbitAmd64Sha256:    "7dc0a1f97214101482d9c329a5108471bcf23fafea421e1ae2662c6c20377037",
		UrbitArm64Sha256:    "1dbded539bd99cd789bfe5cbe11bb89baa598e213881389e8138da8fc0a27fa9",
		MinioAmd64Sha256:    "f6a3001a765dc59a8e365149ade0ea628494230e984891877ead016eb24ba9a9",
		MinioArm64Sha256:    "567779c9f29aca670f84d066051290faeaae6c3ad3a3b7062de4936aaab2a29d",
		MinioPassword:       "",
		Network:             "none",
		WgURL:               "",
		WgHTTPPort:          0,
		WgAmesPort:          0,
		WgS3Port:            0,
		WgConsolePort:       0,
		MeldSchedule:        false,
		MeldScheduleType:    "week",
		MeldFrequency:       1,
		MeldTime:            "0000",
		MeldLast:            "0",
		MeldNext:            "0",
		BootStatus:          "boot",
		CustomPierLocation:  nil,
		CustomUrbitWeb:      "",
		CustomS3Web:         "",
		ShowUrbitWeb:        "",
		DevMode:             false,
		Click:               true,
		StartramReminder:    true,
		RemoteTlonBackup:    true,
		LocalTlonBackup:     true,
		BackupTime:          "0000",
		DisableShipRestarts: false,
		SnapTime:            60,
	}
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

func DockerData(pathType string) string {
	var dockerVolDirs = []string{
		"/media/data/docker/volumes",
		"/var/lib/docker/volumes",
	}
	for _, dir := range dockerVolDirs {
		if _, err := os.Stat(dir); err == nil {
			if pathType == "volumes" {
				return dir
			} else {
				splitPath := strings.Split(dir, "/")
				basePath := "/" + strings.Join(splitPath[:len(splitPath)-1], "/")
				return basePath
			}
		}
	}
	return ""
}

// this one needs params from config so we use a func
func SysConfig(basePath string) structs.SysConfig {
	sysConfig := structs.SysConfig{
		RemoteBackupPassword: "",
		GracefulExit:         false,
		LastKnownMDNS:        "",
		Setup:                "start",
		EndpointUrl:          "api.startram.io",
		ApiVersion:           "v1",
		Piers:                []string{},
		NetCheck:             "1.1.1.1:53",
		UpdateMode:           "auto",
		UpdateUrl:            "https://version.groundseg.app",
		UpdateBranch:         "latest",
		SwapVal:              16,
		SwapFile:             filepath.Join(basePath, "swapfile"),
		KeyFile:              filepath.Join(basePath, "settings", "session.key"),
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
		}{
			Value:    1,
			Interval: "month",
		},
		DockerData:     DockerData("basePath"),
		WgOn:           false,
		WgRegistered:   false,
		PwHash:         "",
		C2cInterval:    0,
		GsVersion:      "v2.0.0",
		CfgDir:         basePath,
		UpdateInterval: 3600,
		BinHash:        "",
		Pubkey:         "",
		Privkey:        "",
		Salt:           "",
		PenpaiRunning:  false,
		PenpaiCores:    1,
		SnapTime:       60,
		PenpaiActive:   "TinyLlama-1.1B",
		PenpaiModels: []structs.Penpai{
			{
				ModelTitle: "TinyLlama 1.1B",
				ModelName:  "TinyLlama-1.1B",
				ModelUrl:   "https://huggingface.co/jartine/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/TinyLlama-1.1B-Chat-v1.0.Q5_K_M.llamafile?download=true",
			},
			{
				ModelTitle: "Mistral 7B Instruct",
				ModelName:  "Mistral 7B Instruct",
				ModelUrl:   "https://huggingface.co/jartine/Mistral-7B-Instruct-v0.2-llamafile/resolve/main/mistral-7b-instruct-v0.2.Q5_K_M.llamafile?download=true",
			},
			{
				ModelTitle: "Mixtral 8x7B Instruct",
				ModelName:  "Mixtral-8x7B-Instruct",
				ModelUrl:   "https://huggingface.co/jartine/Mixtral-8x7B-Instruct-v0.1-llamafile/resolve/main/mixtral-8x7b-instruct-v0.1.Q5_K_M.llamafile?download=true",
			},
			{
				ModelTitle: "WizardCoder Python 13B",
				ModelName:  "WizardCoder-Python-13B",
				ModelUrl:   "https://huggingface.co/jartine/wizardcoder-13b-python/resolve/main/wizardcoder-python-13b.llamafile?download=true",
			},
			{
				ModelTitle: "WizardCoder Python 34B",
				ModelName:  "WizardCoder-Python-34B",
				ModelUrl:   "https://huggingface.co/jartine/WizardCoder-Python-34B-V1.0-llamafile/resolve/main/wizardcoder-python-34b-v1.0.Q5_K_M.llamafile?download=true",
			},
			{
				ModelTitle: "LLaVA 1.5",
				ModelName:  "LLaVA-1.5",
				ModelUrl:   "https://huggingface.co/jartine/llava-v1.5-7B-GGUF/resolve/main/llava-v1.5-7b-q4.llamafile?download=true",
			},
			{
				ModelTitle: "TinyLlama 1.1B",
				ModelName:  "TinyLlama-1.1B",
				ModelUrl:   "https://huggingface.co/jartine/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/TinyLlama-1.1B-Chat-v1.0.Q5_K_M.llamafile?download=true",
			},
			{
				ModelTitle: "Rocket 3B",
				ModelName:  "Rocket-3B",
				ModelUrl:   "https://huggingface.co/jartine/rocket-3B-llamafile/resolve/main/rocket-3b.Q5_K_M.llamafile?download=true",
			},
		},
	}
	return sysConfig
}
