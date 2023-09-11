package structs

import "encoding/json"

// system.json config struct
type SysConfig struct {
	Setup        string   `json:"setup"`
	EndpointUrl  string   `json:"endpointUrl"`
	ApiVersion   string   `json:"apiVersion"`
	Piers        []string `json:"piers"`
	NetCheck     string   `json:"netCheck"`
	UpdateMode   string   `json:"updateMode"`
	UpdateUrl    string   `json:"updateUrl"`
	UpdateBranch string   `json:"updateBranch"`
	SwapVal      int      `json:"swapVal"`
	SwapFile     string   `json:"swapFile"`
	KeyFile      string   `json:"keyFile"`
	Sessions     struct {
		Authorized   map[string]SessionInfo `json:"authorized"`
		Unauthorized map[string]SessionInfo `json:"unauthorized"`
	} `json:"sessions"`
	LinuxUpdates struct {
		Value    int    `json:"value"`
		Interval string `json:"interval"`
		Previous bool   `json:"previous"`
	} `json:"linuxUpdates"`
	DockerData     string `json:"dockerData"`
	WgOn           bool   `json:"wgOn"`
	WgRegistered   bool   `json:"wgRegistered"`
	PwHash         string `json:"pwHash"`
	C2cInterval    int    `json:"c2cInterval"`
	FirstBoot      bool   `json:"firstBoot"`
	GsVersion      string `json:"gsVersion"`
	CfgDir         string `json:"CFG_DIR"`
	UpdateInterval int    `json:"updateInterval"`
	BinHash        string `json:"binHash"`
	Pubkey         string `json:"pubkey"`
	Privkey        string `json:"privkey"`
	Salt           string `json:"salt"`
}

// authenticated browser sessions
type SessionInfo struct {
	Hash    string `json:"hash"`
	Created string `json:"created"`
}

// pier json struct
type UrbitDocker struct {
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
	Network          string `json:"network"`
	WgURL            string `json:"wg_url"`
	WgHTTPPort       int    `json:"wg_http_port"`
	WgAmesPort       int    `json:"wg_ames_port"`
	WgS3Port         int    `json:"wg_s3_port"`
	WgConsolePort    int    `json:"wg_console_port"`
	MeldSchedule     bool   `json:"meld_schedule"`
	MeldFrequency    int    `json:"meld_frequency"`
	MeldTime         string `json:"meld_time"`
	MeldLast         string `json:"meld_last"`
	MeldNext         string `json:"meld_next"`
	BootStatus       string `json:"boot_status"`
	CustomUrbitWeb   string `json:"custom_urbit_web"`
	CustomS3Web      string `json:"custom_s3_web"`
	ShowUrbitWeb     string `json:"show_urbit_web"`
	DevMode          bool   `json:"dev_mode"`
	Click            bool   `json:"click"`
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
		case "wg_http_port":
			u.SetWgHTTPPort(v)
		case "wg_ames_port":
			u.SetWgAmesPort(v)
		case "wg_s3_port":
			u.SetWgS3Port(v)
		case "wg_console_port":
			u.SetWgConsolePort(v)
			// Handle other fields similarly
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
