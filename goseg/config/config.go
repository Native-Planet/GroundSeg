package config

// code for managing groundseg and container configurations

import (
	"groundseg/defaults"
	"sync/atomic"
	"time"

	"groundseg/structs"
)

type StartramConfigSnapshot struct {
	Value     structs.StartramRetrieve
	UpdatedAt time.Time
	Fresh     bool
}

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
	C2cInterval int
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

type RuntimeSettings struct {
	BasePath     string
	Architecture string
	DebugMode    bool
}

// RuntimeContext holds immutable process-level execution context values that
// orchestrators and handlers can pass explicitly through seams.
type RuntimeContext struct {
	BasePath     string
	Architecture string
	DebugMode    bool
	DockerDir    string
}

var (
	basePathDefault     = getBasePath
	architectureDefault = getArchitecture
	defaultDebugMode    = func() bool { return false }
	dockerDirDefault    = func() string { return defaults.DockerData("volumes") + "/" }
	runtimeContextValue atomic.Pointer[RuntimeContext]
)

// RuntimeContextSnapshot returns an immutable runtime context for the current
// process state. Callers should pass this into orchestration boundaries instead
// of reaching into package-level mutable config state.
func RuntimeContextSnapshot() RuntimeContext {
	ctx := runtimeContextValue.Load()
	if ctx == nil {
		return RuntimeContext{
			BasePath:     basePathDefault(),
			Architecture: architectureDefault(),
			DebugMode:    defaultDebugMode(),
			DockerDir:    dockerDirDefault(),
		}
	}
	return *ctx
}

func applyRuntimeContext(context RuntimeContext) {
	updated := context
	runtimeContextValue.Store(&updated)
}

func setRuntimeContext(context RuntimeContext) {
	applyRuntimeContext(context)
}

// BasePath returns the active process base path used by config-owned filesystem paths.
func BasePath() string {
	ctx := RuntimeContextSnapshot()
	return ctx.BasePath
}

// Architecture returns the active process architecture used by config-owned configuration.
func Architecture() string {
	ctx := RuntimeContextSnapshot()
	return ctx.Architecture
}

// DebugMode returns whether runtime debug mode is enabled.
func DebugMode() bool {
	ctx := RuntimeContextSnapshot()
	return ctx.DebugMode
}

// DockerDir returns the active docker volume root used by config-owned paths.
func DockerDir() string {
	ctx := RuntimeContextSnapshot()
	return ctx.DockerDir
}

// SetBasePath updates process context state in one location.
func SetBasePath(basePath string) {
	setRuntimeContextBasePath(basePath)
}

// SetArchitecture updates process context architecture in one location.
func SetArchitecture(architecture string) {
	current := RuntimeContextSnapshot()
	current.Architecture = architecture
	setRuntimeContext(current)
}

// SetDebugMode updates process context debug mode in one location.
func SetDebugMode(debugMode bool) {
	setRuntimeContextDebugMode(debugMode)
}

// SetDockerDir updates process context docker directory in one location.
func SetDockerDir(dockerDir string) {
	current := RuntimeContextSnapshot()
	current.DockerDir = dockerDir
	setRuntimeContext(current)
}

func setRuntimeContextBasePath(basePath string) {
	current := RuntimeContextSnapshot()
	current.BasePath = basePath
	setRuntimeContext(current)
}

func setRuntimeContextDebugMode(debugMode bool) {
	current := RuntimeContextSnapshot()
	current.DebugMode = debugMode
	setRuntimeContext(current)
}

func init() {
	setRuntimeContext(RuntimeContext{
		BasePath:     getBasePath(),
		Architecture: getArchitecture(),
		DebugMode:    false,
		DockerDir:    defaults.DockerData("volumes") + "/",
	})
}

func SetStartramConfig(retrieve structs.StartramRetrieve) {
	startramMu.Lock()
	defer startramMu.Unlock()
	startramConfig = retrieve
	startramConfigUpdated = time.Now()
}

func GetStartramConfig() structs.StartramRetrieve {
	startramMu.RLock()
	defer startramMu.RUnlock()
	return startramConfig
}

func GetStartramConfigSnapshot() StartramConfigSnapshot {
	startramMu.RLock()
	defer startramMu.RUnlock()
	return StartramConfigSnapshot{
		Value:     startramConfig,
		UpdatedAt: startramConfigUpdated,
		Fresh:     !startramConfigUpdated.IsZero(),
	}
}
