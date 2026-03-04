package structs

import (
	"encoding/json"
	"fmt"
	"time"
)

type DiskWarning struct {
	// true if warning has been sent
	// resets to false when disk goes below 80% usage
	// 95% will send a recurring warning
	Eighty     bool      `json:"eighty"`
	Ninety     bool      `json:"ninety"`
	NinetyFive time.Time `json:"ninetyFive"`
}

// system.json config struct grouped by domain
type SysConfig struct {
	Connectivity ConnectivityConfig `json:"-"`
	Startram     StartramConfig     `json:"-"`
	Runtime      RuntimeConfig      `json:"-"`
	AuthSession  AuthSessionConfig  `json:"-"`
	Penpai       PenpaiConfig       `json:"-"`
}

type SysConfigSection string

const (
	SysConfigSectionConnectivity SysConfigSection = "connectivity"
	SysConfigSectionStartram     SysConfigSection = "startram"
	SysConfigSectionRuntime      SysConfigSection = "runtime"
	SysConfigSectionAuthSession  SysConfigSection = "authSession"
	SysConfigSectionPenpai       SysConfigSection = "penpai"
)

func (section SysConfigSection) String() string {
	return string(section)
}

type sysConfigPersistenceShape struct {
	ConnectivityConfig
	StartramConfig
	RuntimeConfig
	AuthSessionConfig
	PenpaiConfig
}

func (config SysConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(sysConfigPersistenceShape{
		ConnectivityConfig: config.Connectivity,
		StartramConfig:     config.Startram,
		RuntimeConfig:      config.Runtime,
		AuthSessionConfig:  config.AuthSession,
		PenpaiConfig:       config.Penpai,
	})
}

func (config *SysConfig) UnmarshalJSON(data []byte) error {
	var raw sysConfigPersistenceShape
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*config = SysConfig{
		Connectivity: raw.ConnectivityConfig,
		Startram:     raw.StartramConfig,
		Runtime:      raw.RuntimeConfig,
		AuthSession:  raw.AuthSessionConfig,
		Penpai:       raw.PenpaiConfig,
	}
	return nil
}

func (config *SysConfig) ConnectivitySection() *ConnectivityConfig {
	if config == nil {
		return nil
	}
	return &config.Connectivity
}

func (config *SysConfig) StartramSection() *StartramConfig {
	if config == nil {
		return nil
	}
	return &config.Startram
}

func (config *SysConfig) RuntimeSection() *RuntimeConfig {
	if config == nil {
		return nil
	}
	return &config.Runtime
}

func (config *SysConfig) AuthSessionSection() *AuthSessionConfig {
	if config == nil {
		return nil
	}
	return &config.AuthSession
}

func (config *SysConfig) PenpaiSection() *PenpaiConfig {
	if config == nil {
		return nil
	}
	return &config.Penpai
}

func (config *SysConfig) UpdateConnectivityConfig(update func(*ConnectivityConfig)) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}
	if update == nil {
		return fmt.Errorf("mutate function is required")
	}
	update(&config.Connectivity)
	return nil
}

func (config *SysConfig) UpdateStartramConfig(update func(*StartramConfig)) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}
	if update == nil {
		return fmt.Errorf("mutate function is required")
	}
	update(&config.Startram)
	return nil
}

func (config *SysConfig) UpdateRuntimeConfig(update func(*RuntimeConfig)) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}
	if update == nil {
		return fmt.Errorf("mutate function is required")
	}
	update(&config.Runtime)
	return nil
}

func (config *SysConfig) UpdateAuthSessionConfig(update func(*AuthSessionConfig)) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}
	if update == nil {
		return fmt.Errorf("mutate function is required")
	}
	update(&config.AuthSession)
	return nil
}

func (config *SysConfig) UpdatePenpaiConfig(update func(*PenpaiConfig)) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}
	if update == nil {
		return fmt.Errorf("mutate function is required")
	}
	update(&config.Penpai)
	return nil
}


type AuthSessionConfig struct {
	Sessions AuthSessionBag `json:"sessions"`
	PwHash   string         `json:"pwHash"`
	Salt     string         `json:"salt"`
	KeyFile  string         `json:"keyFile"`
}

type AuthSessionBag struct {
	Authorized   map[string]SessionInfo `json:"authorized"`
	Unauthorized map[string]SessionInfo `json:"unauthorized"`
}

type ConnectivityConfig struct {
	Piers                []string               `json:"piers"`
	WgOn                 bool                   `json:"wgOn"`
	NetCheck             string                 `json:"netCheck"`
	UpdateMode           string                 `json:"updateMode"`
	UpdateUrl            string                 `json:"updateUrl"`
	UpdateBranch         string                 `json:"updateBranch"`
	RemoteBackupPassword string                 `json:"remoteBackupPassword"`
	C2cInterval          int                    `json:"c2cInterval"`
	DiskWarning          map[string]DiskWarning `json:"diskWarning"`
	WgRegistered         bool                   `json:"wgRegistered"`
	EndpointUrl          string                 `json:"endpointUrl"`
	ApiVersion           string                 `json:"apiVersion"`
}

type RuntimeConfig struct {
	GracefulExit  bool   `json:"gracefulExit"`
	LastKnownMDNS string `json:"lastKnownMDNS"`
	Setup         string `json:"setup"`
	SwapVal       int    `json:"swapVal"`
	SwapFile      string `json:"swapFile"`
	LinuxUpdates  struct {
		Value    int    `json:"value"`
		Interval string `json:"interval"`
	} `json:"linuxUpdates"`
	DockerData     string `json:"dockerData"`
	GsVersion      string `json:"gsVersion"`
	CfgDir         string `json:"CFG_DIR"`
	UpdateInterval int    `json:"updateInterval"`
	BinHash        string `json:"binHash"`
	Disable502     bool   `json:"disable502"`
	SnapTime       int    `json:"snapTime"`
}

type StartramConfig struct {
	StartramSetReminder struct {
		One   bool `json:"one"`
		Three bool `json:"three"`
		Seven bool `json:"seven"`
	} `json:"startramSetReminder"`
	Pubkey      string `json:"pubkey"`
	Privkey     string `json:"privkey"`
	DisableSlsa bool   `json:"disableSlsa"`
}

type PenpaiConfig struct {
	PenpaiAllow   bool     `json:"penpaiAllow"`
	PenpaiRunning bool     `json:"penpaiRunning"`
	PenpaiCores   int      `json:"penpaiCores"`
	PenpaiModels  []Penpai `json:"penpaiModels"`
	PenpaiActive  string   `json:"penpaiActive"`
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

type UrbitRuntimeConfig struct {
	PierName         string `json:"pier_name"`
	HTTPPort         int    `json:"http_port"`
	AmesPort         int    `json:"ames_port"`
	LoomSize         int    `json:"loom_size"`
	UrbitVersion     string `json:"urbit_version"`
	MinioVersion     string `json:"minio_version"`
	UrbitRepo        string `json:"urbit_repo"`
	MinioRepo        string `json:"minio_repo"`
	UrbitAmd64Sha256 string `json:"urbit_amd64_sha256"`
	UrbitArm64Sha256 string `json:"urbit_arm64_sha256"`
	MinioAmd64Sha256 string `json:"minio_amd64_sha256"`
	MinioArm64Sha256 string `json:"minio_arm64_sha256"`
	MinioPassword    string `json:"minio_password"`
	BootStatus       string `json:"boot_status"`
	SizeLimit        int    `json:"size_limit"`
	SnapTime         int    `json:"snap_time"`
}

type UrbitNetworkConfig struct {
	Network       string `json:"network"`
	WgURL         string `json:"wg_url"`
	WgHTTPPort    int    `json:"wg_http_port"`
	WgAmesPort    int    `json:"wg_ames_port"`
	WgS3Port      int    `json:"wg_s3_port"`
	WgConsolePort int    `json:"wg_console_port"`
}

type UrbitScheduleConfig struct {
	MeldSchedule     bool   `json:"meld_schedule"`
	MeldScheduleType string `json:"meld_schedule_type"`
	MeldDay          string `json:"meld_day"`
	MeldDate         int    `json:"meld_date"`
	MeldFrequency    int    `json:"meld_frequency"`
	MeldTime         string `json:"meld_time"`
	MeldLast         string `json:"meld_last"`
	MeldNext         string `json:"meld_next"`
}

type UrbitWebConfig struct {
	CustomPierLocation string `json:"custom_pier_location"`
	CustomUrbitWeb     string `json:"custom_urbit_web"`
	CustomS3Web        string `json:"custom_s3_web"`
	ShowUrbitWeb       string `json:"show_urbit_web"`
}

type UrbitFeatureConfig struct {
	DevMode             bool `json:"dev_mode"`
	Click               bool `json:"click"`
	MinIOLinked         bool `json:"minio_linked"`
	StartramReminder    bool `json:"startram_reminder"`
	ChopOnUpgrade       bool `json:"chop_on_upgrade"`
	DisableShipRestarts bool `json:"disable_ship_restarts"`
}

type UrbitBackupConfig struct {
	RemoteTlonBackup bool   `json:"remote_tlon_backup"`
	LocalTlonBackup  bool   `json:"local_tlon_backup"`
	BackupTime       string `json:"backup_time"`
}

// UrbitDocker is a compatibility-aware model split by concern.
// Concerns are preserved through embedding so callsites can continue
// using existing field names while enabling focused updates.
type UrbitDocker struct {
	UrbitRuntimeConfig
	UrbitNetworkConfig
	UrbitScheduleConfig
	UrbitWebConfig
	UrbitFeatureConfig
	UrbitBackupConfig
}

func parseRequiredString(name string, value any) (string, error) {
	parsed, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("invalid %s value: %T", name, value)
	}
	return parsed, nil
}

func parseRequiredBool(name string, value any) (bool, error) {
	parsed, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("invalid %s value: %T", name, value)
	}
	return parsed, nil
}

func parseRequiredInt(name string, value any) (int, error) {
	switch parsed := value.(type) {
	case float64:
		return int(parsed), nil
	case int:
		return parsed, nil
	default:
		return 0, fmt.Errorf("invalid %s value: %T", name, value)
	}
}

// Custom unmarshaler
func (u *UrbitDocker) UnmarshalJSON(data []byte) error {
	u.StartramReminder = true
	u.ChopOnUpgrade = true
	u.DisableShipRestarts = false
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for k, v := range raw {
		switch k {
		case "minio_linked":
			parsed, err := parseRequiredBool("minio_linked", v)
			if err != nil {
				return err
			}
			u.MinIOLinked = parsed
		case "pier_name":
			parsed, err := parseRequiredString("pier_name", v)
			if err != nil {
				return err
			}
			u.PierName = parsed
		case "http_port":
			parsed, err := parseRequiredInt("http_port", v)
			if err != nil {
				return err
			}
			u.HTTPPort = parsed
		case "ames_port":
			parsed, err := parseRequiredInt("ames_port", v)
			if err != nil {
				return err
			}
			u.AmesPort = parsed
		case "loom_size":
			parsed, err := parseRequiredInt("loom_size", v)
			if err != nil {
				return err
			}
			u.LoomSize = parsed
		case "urbit_version":
			parsed, err := parseRequiredString("urbit_version", v)
			if err != nil {
				return err
			}
			u.UrbitVersion = parsed
		case "minio_version":
			parsed, err := parseRequiredString("minio_version", v)
			if err != nil {
				return err
			}
			u.MinioVersion = parsed
		case "urbit_repo":
			parsed, err := parseRequiredString("urbit_repo", v)
			if err != nil {
				return err
			}
			u.UrbitRepo = parsed
		case "minio_repo":
			parsed, err := parseRequiredString("minio_repo", v)
			if err != nil {
				return err
			}
			u.MinioRepo = parsed
		case "urbit_amd64_sha256":
			parsed, err := parseRequiredString("urbit_amd64_sha256", v)
			if err != nil {
				return err
			}
			u.UrbitAmd64Sha256 = parsed
		case "urbit_arm64_sha256":
			parsed, err := parseRequiredString("urbit_arm64_sha256", v)
			if err != nil {
				return err
			}
			u.UrbitArm64Sha256 = parsed
		case "minio_amd64_sha256":
			parsed, err := parseRequiredString("minio_amd64_sha256", v)
			if err != nil {
				return err
			}
			u.MinioAmd64Sha256 = parsed
		case "minio_arm64_sha256":
			parsed, err := parseRequiredString("minio_arm64_sha256", v)
			if err != nil {
				return err
			}
			u.MinioArm64Sha256 = parsed
		case "minio_password":
			parsed, err := parseRequiredString("minio_password", v)
			if err != nil {
				return err
			}
			u.MinioPassword = parsed
		case "network":
			parsed, err := parseRequiredString("network", v)
			if err != nil {
				return err
			}
			u.Network = parsed
		case "wg_url":
			parsed, err := parseRequiredString("wg_url", v)
			if err != nil {
				return err
			}
			u.WgURL = parsed
		case "wg_http_port":
			parsed, err := parseRequiredInt("wg_http_port", v)
			if err != nil {
				return err
			}
			u.WgHTTPPort = parsed
		case "wg_ames_port":
			parsed, err := parseRequiredInt("wg_ames_port", v)
			if err != nil {
				return err
			}
			u.WgAmesPort = parsed
		case "wg_s3_port":
			parsed, err := parseRequiredInt("wg_s3_port", v)
			if err != nil {
				return err
			}
			u.WgS3Port = parsed
		case "wg_console_port":
			parsed, err := parseRequiredInt("wg_console_port", v)
			if err != nil {
				return err
			}
			u.WgConsolePort = parsed
		case "meld_schedule":
			parsed, err := parseRequiredBool("meld_schedule", v)
			if err != nil {
				return err
			}
			u.MeldSchedule = parsed
		case "meld_schedule_type":
			parsed, err := parseRequiredString("meld_schedule_type", v)
			if err != nil {
				return err
			}
			u.MeldScheduleType = parsed
		case "meld_day":
			parsed, err := parseRequiredString("meld_day", v)
			if err != nil {
				return err
			}
			u.MeldDay = parsed
		case "meld_date":
			parsed, err := parseRequiredInt("meld_date", v)
			if err != nil {
				return err
			}
			u.MeldDate = parsed
		case "meld_frequency":
			parsed, err := parseRequiredInt("meld_frequency", v)
			if err != nil {
				return err
			}
			u.MeldFrequency = parsed
		case "meld_time":
			parsed, err := parseRequiredString("meld_time", v)
			if err != nil {
				return err
			}
			u.MeldTime = parsed
		case "meld_last":
			parsed, err := parseRequiredString("meld_last", v)
			if err != nil {
				return err
			}
			u.MeldLast = parsed
		case "meld_next":
			parsed, err := parseRequiredString("meld_next", v)
			if err != nil {
				return err
			}
			u.MeldNext = parsed
		case "boot_status":
			parsed, err := parseRequiredString("boot_status", v)
			if err != nil {
				return err
			}
			u.BootStatus = parsed
		case "disable_ship_restarts":
			parsed, err := parseRequiredBool("disable_ship_restarts", v)
			if err != nil {
				return err
			}
			u.DisableShipRestarts = parsed
		case "custom_urbit_web":
			parsed, err := parseRequiredString("custom_urbit_web", v)
			if err != nil {
				return err
			}
			u.CustomUrbitWeb = parsed
		case "custom_s3_web":
			parsed, err := parseRequiredString("custom_s3_web", v)
			if err != nil {
				return err
			}
			u.CustomS3Web = parsed
		case "show_urbit_web":
			parsed, err := parseRequiredString("show_urbit_web", v)
			if err != nil {
				return err
			}
			u.ShowUrbitWeb = parsed
		case "dev_mode":
			parsed, err := parseRequiredBool("dev_mode", v)
			if err != nil {
				return err
			}
			u.DevMode = parsed
		case "click":
			parsed, err := parseRequiredBool("click", v)
			if err != nil {
				return err
			}
			u.Click = parsed
		case "startram_reminder":
			parsed, err := parseRequiredBool("startram_reminder", v)
			if err != nil {
				return err
			}
			u.StartramReminder = parsed
		case "custom_pier_location":
			parsed, err := parseRequiredString("custom_pier_location", v)
			if err != nil {
				return err
			}
			u.CustomPierLocation = parsed
		case "chop_on_upgrade":
			parsed, err := parseRequiredBool("chop_on_upgrade", v)
			if err != nil {
				return err
			}
			u.ChopOnUpgrade = parsed
		case "size_limit":
			parsed, err := parseRequiredInt("size_limit", v)
			if err != nil {
				return err
			}
			u.SizeLimit = parsed
		case "remote_tlon_backup":
			parsed, err := parseRequiredBool("remote_tlon_backup", v)
			if err != nil {
				return err
			}
			u.RemoteTlonBackup = parsed
		case "local_tlon_backup":
			parsed, err := parseRequiredBool("local_tlon_backup", v)
			if err != nil {
				return err
			}
			u.LocalTlonBackup = parsed
		case "backup_time":
			parsed, err := parseRequiredString("backup_time", v)
			if err != nil {
				return err
			}
			u.BackupTime = parsed
		case "snap_time":
			parsed, err := parseRequiredInt("snap_time", v)
			if err != nil {
				return err
			}
			u.SnapTime = parsed
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
