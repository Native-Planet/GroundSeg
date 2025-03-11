package structs

import (
	"encoding/json"
	"time"
)

// system.json config struct
type SysConfig struct {
	RemoteBackupPassword string   `json:"remoteBackupPassword"`
	GracefulExit         bool     `json:"gracefulExit"`
	LastKnownMDNS        string   `json:"lastKnownMDNS"`
	Setup                string   `json:"setup"`
	EndpointUrl          string   `json:"endpointUrl"`
	ApiVersion           string   `json:"apiVersion"`
	Piers                []string `json:"piers"`
	NetCheck             string   `json:"netCheck"`
	UpdateMode           string   `json:"updateMode"`
	UpdateUrl            string   `json:"updateUrl"`
	UpdateBranch         string   `json:"updateBranch"`
	SwapVal              int      `json:"swapVal"`
	SwapFile             string   `json:"swapFile"`
	KeyFile              string   `json:"keyFile"`
	Sessions             struct {
		Authorized   map[string]SessionInfo `json:"authorized"`
		Unauthorized map[string]SessionInfo `json:"unauthorized"`
	} `json:"sessions"`
	LinuxUpdates struct {
		Value    int    `json:"value"`
		Interval string `json:"interval"`
	} `json:"linuxUpdates"`
	DockerData          string `json:"dockerData"`
	WgOn                bool   `json:"wgOn"`
	WgRegistered        bool   `json:"wgRegistered"`
	StartramSetReminder struct {
		// true if reminder has been sent
		// resets to false when ongoing == 1
		One   bool `json:"one"`
		Three bool `json:"three"`
		Seven bool `json:"seven"`
	} `json:"startramSetReminder"`
	DiskWarning    map[string]DiskWarning `json:"diskWarning"`
	PwHash         string                 `json:"pwHash"`
	C2cInterval    int                    `json:"c2cInterval"`
	GsVersion      string                 `json:"gsVersion"`
	CfgDir         string                 `json:"CFG_DIR"`
	UpdateInterval int                    `json:"updateInterval"`
	BinHash        string                 `json:"binHash"`
	Pubkey         string                 `json:"pubkey"`
	Privkey        string                 `json:"privkey"`
	Salt           string                 `json:"salt"`
	PenpaiAllow    bool                   `json:"penpaiAllow"`
	PenpaiRunning  bool                   `json:"penpaiRunning"`
	PenpaiCores    int                    `json:"penpaiCores"`
	PenpaiModels   []Penpai               `json:"penpaiModels"`
	PenpaiActive   string                 `json:"penpaiActive"`
	DisableSlsa    bool                   `json:"disableSlsa"`
}

type DiskWarning struct {
	// true if warning has been sent
	// resets to false when disk goes below 80% usage
	// 95% will send a recurring warning
	Eighty     bool      `json:"eighty"`
	Ninety     bool      `json:"ninety"`
	NinetyFive time.Time `json:"ninetyFive"`
}

type Penpai struct {
	ModelTitle string `json:"modelTitle"`
	ModelName  string `json:"modelName"`
	ModelUrl   string `json:"modelUrl"`
}

// authenticated browser sessions
type SessionInfo struct {
	Hash    string `json:"hash"`
	Created string `json:"created"`
}

// pier json struct
type UrbitDocker struct {
	PierName            string      `json:"pier_name"`
	HTTPPort            int         `json:"http_port"`
	AmesPort            int         `json:"ames_port"`
	LoomSize            int         `json:"loom_size"`
	UrbitVersion        string      `json:"urbit_version"`
	MinioVersion        string      `json:"minio_version"`
	UrbitRepo           string      `json:"urbit_repo"`
	MinioRepo           string      `json:"minio_repo"`
	UrbitAmd64Sha256    string      `json:"urbit_amd64_sha256"`
	UrbitArm64Sha256    string      `json:"urbit_arm64_sha256"`
	MinioAmd64Sha256    string      `json:"minio_amd64_sha256"`
	MinioArm64Sha256    string      `json:"minio_arm64_sha256"`
	MinioPassword       string      `json:"minio_password"`
	Network             string      `json:"network"`
	WgURL               string      `json:"wg_url"`
	WgHTTPPort          int         `json:"wg_http_port"`
	WgAmesPort          int         `json:"wg_ames_port"`
	WgS3Port            int         `json:"wg_s3_port"`
	WgConsolePort       int         `json:"wg_console_port"`
	MeldSchedule        bool        `json:"meld_schedule"`
	MeldScheduleType    string      `json:"meld_schedule_type"`
	MeldDay             string      `json:"meld_day"`
	MeldDate            int         `json:"meld_date"`
	MeldFrequency       int         `json:"meld_frequency"`
	MeldTime            string      `json:"meld_time"`
	MeldLast            string      `json:"meld_last"`
	MeldNext            string      `json:"meld_next"`
	BootStatus          string      `json:"boot_status"`
	CustomPierLocation  interface{} `json:"custom_pier_location"`
	CustomUrbitWeb      string      `json:"custom_urbit_web"`
	CustomS3Web         string      `json:"custom_s3_web"`
	ShowUrbitWeb        string      `json:"show_urbit_web"`
	DevMode             bool        `json:"dev_mode"`
	Click               bool        `json:"click"`
	MinIOLinked         bool        `json:"minio_linked"`
	StartramReminder    interface{} `json:"startram_reminder"`
	ChopOnUpgrade       interface{} `json:"chop_on_upgrade"`
	SizeLimit           int         `json:"size_limit"`
	RemoteTlonBackup    bool        `json:"remote_tlon_backup"`
	LocalTlonBackup     bool        `json:"local_tlon_backup"`
	BackupTime          string      `json:"backup_time"`
	DisableShipRestarts interface{} `json:"disable_ship_restarts"`
}

// Define the interface
type PortSetter interface {
	SetPort(port interface{})
}

// Add SetPort methods for relevant fields in the struct
func (u *UrbitDocker) SetWgHTTPPort(port interface{}) {
	u.WgHTTPPort = toInt(port)
}

func (u *UrbitDocker) SetWgAmesPort(port interface{}) {
	u.WgAmesPort = toInt(port)
}

func (u *UrbitDocker) SetWgS3Port(port interface{}) {
	u.WgS3Port = toInt(port)
}

func (u *UrbitDocker) SetWgConsolePort(port interface{}) {
	u.WgConsolePort = toInt(port)
}

func (u *UrbitDocker) SetSizeLimit(size interface{}) {
	u.SizeLimit = toInt(size)
}

// Helper function to convert a value to int, returns 0 if not an int
func toInt(value interface{}) int {
	if v, ok := value.(float64); ok { // JSON numbers are float64
		return int(v)
	}
	return 0
}

// Custom unmarshaler
func (u *UrbitDocker) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for k, v := range raw {
		switch k {
		case "minio_linked":
			u.MinIOLinked = v.(bool)
		case "pier_name":
			u.PierName, _ = v.(string)
		case "http_port":
			u.HTTPPort = toInt(v)
		case "ames_port":
			u.AmesPort = toInt(v)
		case "loom_size":
			u.LoomSize = toInt(v)
		case "urbit_version":
			u.UrbitVersion, _ = v.(string)
		case "minio_version":
			u.MinioVersion, _ = v.(string)
		case "urbit_repo":
			u.UrbitRepo, _ = v.(string)
		case "minio_repo":
			u.MinioRepo, _ = v.(string)
		case "urbit_amd64_sha256":
			u.UrbitAmd64Sha256, _ = v.(string)
		case "urbit_arm64_sha256":
			u.UrbitArm64Sha256, _ = v.(string)
		case "minio_amd64_sha256":
			u.MinioAmd64Sha256, _ = v.(string)
		case "minio_arm64_sha256":
			u.MinioArm64Sha256, _ = v.(string)
		case "minio_password":
			u.MinioPassword, _ = v.(string)
		case "network":
			u.Network, _ = v.(string)
		case "wg_url":
			u.WgURL, _ = v.(string)
		case "wg_http_port":
			u.SetWgHTTPPort(v)
		case "wg_ames_port":
			u.SetWgAmesPort(v)
		case "wg_s3_port":
			u.SetWgS3Port(v)
		case "wg_console_port":
			u.SetWgConsolePort(v)
		case "meld_schedule":
			u.MeldSchedule, _ = v.(bool)
		case "meld_schedule_type":
			u.MeldScheduleType, _ = v.(string)
		case "meld_day":
			u.MeldDay, _ = v.(string)
		case "meld_date":
			u.MeldDate = toInt(v)
		case "meld_frequency":
			u.MeldFrequency = toInt(v)
		case "meld_time":
			u.MeldTime, _ = v.(string)
		case "meld_last":
			u.MeldLast, _ = v.(string)
		case "meld_next":
			u.MeldNext, _ = v.(string)
		case "boot_status":
			u.BootStatus, _ = v.(string)
		case "disable_ship_restarts":
			if v == nil {
				u.DisableShipRestarts = false
			} else {
				u.DisableShipRestarts = v.(bool)
			}
		case "custom_urbit_web":
			u.CustomUrbitWeb, _ = v.(string)
		case "custom_s3_web":
			u.CustomS3Web, _ = v.(string)
		case "show_urbit_web":
			u.ShowUrbitWeb, _ = v.(string)
		case "dev_mode":
			u.DevMode, _ = v.(bool)
		case "click":
			u.Click, _ = v.(bool)
		case "startram_reminder":
			if v == nil {
				u.StartramReminder = true
			} else {
				u.StartramReminder = v.(bool)
			}
		case "custom_pier_location":
			if v == nil {
				u.CustomPierLocation = nil
			} else {
				u.CustomPierLocation = v.(string)
			}
		case "chop_on_upgrade":
			if v == nil {
				u.ChopOnUpgrade = true
			} else {
				u.ChopOnUpgrade = v.(bool)
			}
		case "size_limit":
			u.SetSizeLimit(v)
		case "remote_tlon_backup":
			if v == nil {
				u.RemoteTlonBackup = true
			} else {
				u.RemoteTlonBackup = v.(bool)
			}
		case "local_tlon_backup":
			if v == nil {
				u.LocalTlonBackup = true
			} else {
				u.LocalTlonBackup = v.(bool)
			}
		case "backup_time":
			u.BackupTime, _ = v.(string)
		}
	}
	return nil
}

// wireguard config json
type WgConfig struct {
	WireguardName    string   `json:"wireguard_name"`
	WireguardVersion string   `json:"wireguard_version"`
	Repo             string   `json:"repo"`
	Amd64Sha256      string   `json:"amd64_sha256"`
	Arm64Sha256      string   `json:"arm64_sha256"`
	CapAdd           []string `json:"cap_add"`
	Volumes          []string `json:"volumes"`
	Sysctls          struct {
		NetIpv4ConfAllSrcValidMark int `json:"net.ipv4.conf.all.src_valid_mark"`
	} `json:"sysctls"`
}

// minio client config json
type McConfig struct {
	McName      string `json:"mc_name"`
	McVersion   string `json:"mc_version"`
	Repo        string `json:"repo"`
	Amd64Sha256 string `json:"amd64_sha256"`
	Arm64Sha256 string `json:"arm64_sha256"`
}

// nedata config json
type NetdataConfig struct {
	NetdataName    string   `json:"netdata_name"`
	Repo           string   `json:"repo"`
	NetdataVersion string   `json:"netdata_version"`
	Amd64Sha256    string   `json:"amd64_sha256"`
	Arm64Sha256    string   `json:"arm64_sha256"`
	CapAdd         []string `json:"cap_add"`
	Port           int      `json:"port"`
	Restart        string   `json:"restart"`
	SecurityOpt    string   `json:"security_opt"`
	Volumes        []string `json:"volumes"`
}
