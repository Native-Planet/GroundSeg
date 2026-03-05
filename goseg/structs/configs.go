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

type UrbitConfigSection string

const (
	UrbitConfigSectionRuntime  UrbitConfigSection = "runtime"
	UrbitConfigSectionNetwork  UrbitConfigSection = "network"
	UrbitConfigSectionSchedule UrbitConfigSection = "schedule"
	UrbitConfigSectionFeature  UrbitConfigSection = "feature"
	UrbitConfigSectionWeb      UrbitConfigSection = "web"
	UrbitConfigSectionBackup   UrbitConfigSection = "backup"
)

func (section UrbitConfigSection) String() string {
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

func updateSysConfigSection[T any](config *SysConfig, section *T, update func(*T)) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}
	if update == nil {
		return fmt.Errorf("mutate function is required")
	}
	update(section)
	return nil
}

func (config *SysConfig) UpdateConnectivityConfig(update func(*ConnectivityConfig)) error {
	return updateSysConfigSection(config, &config.Connectivity, update)
}

func (config *SysConfig) UpdateStartramConfig(update func(*StartramConfig)) error {
	return updateSysConfigSection(config, &config.Startram, update)
}

func (config *SysConfig) UpdateRuntimeConfig(update func(*RuntimeConfig)) error {
	return updateSysConfigSection(config, &config.Runtime, update)
}

func (config *SysConfig) UpdateAuthSessionConfig(update func(*AuthSessionConfig)) error {
	return updateSysConfigSection(config, &config.AuthSession, update)
}

func (config *SysConfig) UpdatePenpaiConfig(update func(*PenpaiConfig)) error {
	return updateSysConfigSection(config, &config.Penpai, update)
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
	UpdateURL            string                 `json:"updateUrl"`
	UpdateBranch         string                 `json:"updateBranch"`
	RemoteBackupPassword string                 `json:"remoteBackupPassword"`
	C2CInterval          int                    `json:"c2cInterval"`
	DiskWarning          map[string]DiskWarning `json:"diskWarning"`
	WgRegistered         bool                   `json:"wgRegistered"`
	EndpointURL          string                 `json:"endpointUrl"`
	ApiVersion           string                 `json:"apiVersion"`
}

type LinuxUpdatesConfig struct {
	Value    int    `json:"value"`
	Interval string `json:"interval"`
}

type RuntimeConfig struct {
	GracefulExit   bool               `json:"gracefulExit"`
	LastKnownMDNS  string             `json:"lastKnownMDNS"`
	Setup          string             `json:"setup"`
	SwapVal        int                `json:"swapVal"`
	SwapFile       string             `json:"swapFile"`
	LinuxUpdates   LinuxUpdatesConfig `json:"linuxUpdates"`
	DockerData     string             `json:"dockerData"`
	GsVersion      string             `json:"gsVersion"`
	CfgDir         string             `json:"CFG_DIR"`
	UpdateInterval int                `json:"updateInterval"`
	BinHash        string             `json:"binHash"`
	Disable502     bool               `json:"disable502"`
	SnapTime       int                `json:"snapTime"`
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

type urbitDockerPersistenceShape UrbitDocker

// Custom unmarshaler
func (u *UrbitDocker) UnmarshalJSON(data []byte) error {
	if u == nil {
		return fmt.Errorf("urbit docker config target is nil")
	}
	decoded := urbitDockerPersistenceShape{
		UrbitFeatureConfig: UrbitFeatureConfig{
			StartramReminder:    true,
			ChopOnUpgrade:       true,
			DisableShipRestarts: false,
		},
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return fmt.Errorf("decode urbit docker config: %w", err)
	}
	*u = UrbitDocker(decoded)
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
