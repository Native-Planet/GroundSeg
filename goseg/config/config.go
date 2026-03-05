package config

// code for managing groundseg and container configurations

import (
	"groundseg/config/contextapi"
	"groundseg/config/startramstate"

	"groundseg/structs"
)

type StartramConfigSnapshot = startramstate.Snapshot

type StartramSettings struct {
	EndpointURL          string
	Pubkey               string
	RemoteBackupPassword string
	WgRegistered         bool
	WgOn                 bool
	Piers                []string
}

type AuthSettings struct {
	KeyFile            string
	Salt               string
	PasswordHash       string
	AuthorizedSessions map[string]structs.SessionInfo
}

type PenpaiSettings struct {
	Models      []structs.Penpai
	Allowed     bool
	ActiveModel string
	Running     bool
	ActiveCores int
}

type Check502Settings struct {
	Piers      []string
	WgOn       bool
	Disable502 bool
}

type HealthCheckSettings struct {
	Piers        []string
	DiskWarnings map[string]structs.DiskWarning
}

type ShipSettings struct {
	Piers []string
}

type ConnectivitySettings struct {
	C2CInterval int
}

type UpdateSettings struct {
	UpdateMode   string
	UpdateBranch string
}

type SwapSettings struct {
	SwapFile string
	SwapVal  int
}

type ShipRuntimeSettings struct {
	SnapTime int
}

type ImportSettings struct {
	Swap     SwapSettings
	Startram StartramSettings
}

type StartupSettings struct {
	Update   UpdateSettings
	Startram StartramSettings
}

type C2CSettings struct {
	Runtime      RuntimeContext
	Connectivity ConnectivitySettings
}

type RuntimeSettings struct {
	BasePath     string
	Architecture string
	DebugMode    bool
}

type RuntimeContext = contextapi.RuntimeContext

// RuntimeContextSnapshot returns an immutable runtime context for the current
// process state. Callers should pass this into orchestration boundaries instead
// of reaching into package-level mutable config state.
func RuntimeContextSnapshot() RuntimeContext {
	return contextapi.Snapshot()
}

// BasePath returns the active process base path used by config-owned filesystem paths.
func BasePath() string {
	return contextapi.BasePath()
}

// Architecture returns the active process architecture used by config-owned configuration.
func Architecture() string {
	return contextapi.Architecture()
}

// DebugMode returns whether runtime debug mode is enabled.
func DebugMode() bool {
	return contextapi.DebugMode()
}

// DockerDir returns the active docker volume root used by config-owned paths.
func DockerDir() string {
	return contextapi.DockerDir()
}

func ImportSettingsSnapshot() ImportSettings {
	return ImportSettings{
		Swap:     SwapSettingsSnapshot(),
		Startram: StartramSettingsSnapshot(),
	}
}

func StartupSettingsSnapshot() StartupSettings {
	return StartupSettings{
		Update:   UpdateSettingsSnapshot(),
		Startram: StartramSettingsSnapshot(),
	}
}

func C2CSettingsSnapshot() C2CSettings {
	return C2CSettings{
		Runtime:      RuntimeContextSnapshot(),
		Connectivity: ConnectivitySettingsSnapshot(),
	}
}

// SetBasePath updates process context state in one location.
func SetBasePath(basePath string) {
	contextapi.SetBasePath(basePath)
}

// SetArchitecture updates process context architecture in one location.
func SetArchitecture(architecture string) {
	contextapi.SetArchitecture(architecture)
}

// SetDebugMode updates process context debug mode in one location.
func SetDebugMode(debugMode bool) {
	contextapi.SetDebugMode(debugMode)
}

// SetDockerDir updates process context docker directory in one location.
func SetDockerDir(dockerDir string) {
	contextapi.SetDockerDir(dockerDir)
}

func SetStartramConfig(retrieve structs.StartramRetrieve) {
	startramstate.Set(retrieve)
}

func GetStartramConfig() structs.StartramRetrieve {
	return startramstate.Get()
}

func GetStartramConfigSnapshot() StartramConfigSnapshot {
	return startramstate.GetSnapshot()
}
