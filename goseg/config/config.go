package config

// code for managing groundseg and container configurations

import (
	"groundseg/config/runtimecontext"
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

type RuntimeContext = runtimecontext.RuntimeContext

// SetRuntimeContext replaces the process runtime context snapshot in one write.
func SetRuntimeContext(context RuntimeContext) {
	runtimecontext.Set(context)
}

// RuntimeContextSnapshot returns an immutable runtime context for the current
// process state. Callers should pass this into orchestration boundaries instead
// of reaching into package-level mutable config state.
func RuntimeContextSnapshot() RuntimeContext {
	return runtimecontext.Snapshot()
}

// BasePath returns the active process base path used by config-owned filesystem paths.
func BasePath() string {
	return runtimecontext.Snapshot().BasePath
}

// Architecture returns the active process architecture used by config-owned configuration.
func Architecture() string {
	return runtimecontext.Snapshot().Architecture
}

// DebugMode returns whether runtime debug mode is enabled.
func DebugMode() bool {
	return runtimecontext.Snapshot().DebugMode
}

// DockerDir returns the active docker volume root used by config-owned paths.
func DockerDir() string {
	return runtimecontext.Snapshot().DockerDir
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
	runtimecontext.SetBasePath(basePath)
}

// SetArchitecture updates process context architecture in one location.
func SetArchitecture(architecture string) {
	runtimecontext.SetArchitecture(architecture)
}

// SetDebugMode updates process context debug mode in one location.
func SetDebugMode(debugMode bool) {
	runtimecontext.SetDebugMode(debugMode)
}

// SetDockerDir updates process context docker directory in one location.
func SetDockerDir(dockerDir string) {
	runtimecontext.SetDockerDir(dockerDir)
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
