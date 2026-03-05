package runtimecontext

import (
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	"groundseg/defaults"
)

// RuntimeContext holds immutable process-level execution context values.
type RuntimeContext struct {
	BasePath     string
	Architecture string
	DebugMode    bool
	DockerDir    string
}

var (
	runtimeContextMu    sync.Mutex
	runtimeContextValue atomic.Pointer[RuntimeContext]
)

// Snapshot returns an immutable runtime context for the current process state.
func Snapshot() RuntimeContext {
	ctx := runtimeContextValue.Load()
	if ctx != nil {
		return *ctx
	}
	runtimeContextMu.Lock()
	defer runtimeContextMu.Unlock()
	ctx = runtimeContextValue.Load()
	if ctx != nil {
		return *ctx
	}
	seeded := defaultRuntimeContext()
	applyRuntimeContext(seeded)
	return seeded
}

func defaultRuntimeContext() RuntimeContext {
	return RuntimeContextFromProcessArgs(nil)
}

// RuntimeContextFromProcessArgs builds a runtime context snapshot from process
// defaults (environment + architecture) and parsed startup args.
func RuntimeContextFromProcessArgs(args []string) RuntimeContext {
	return RuntimeContext{
		BasePath:     BasePathFromEnv(),
		Architecture: ArchitectureFromRuntime(),
		DebugMode:    DebugModeFromArgs(args),
		DockerDir:    DockerDirFromDefaults(),
	}
}

func updateRuntimeContext(updateFn func(*RuntimeContext)) {
	runtimeContextMu.Lock()
	defer runtimeContextMu.Unlock()
	current := runtimeContextValue.Load()
	ctx := defaultRuntimeContext()
	if current != nil {
		ctx = *current
	}
	updateFn(&ctx)
	applyRuntimeContext(ctx)
}

func applyRuntimeContext(context RuntimeContext) {
	updated := context
	runtimeContextValue.Store(&updated)
}

// Set stores a runtime context snapshot.
func Set(context RuntimeContext) {
	applyRuntimeContext(context)
}

func SetBasePath(basePath string) {
	updateRuntimeContext(func(context *RuntimeContext) {
		context.BasePath = basePath
	})
}

func SetArchitecture(architecture string) {
	updateRuntimeContext(func(context *RuntimeContext) {
		context.Architecture = architecture
	})
}

func SetDebugMode(debugMode bool) {
	updateRuntimeContext(func(context *RuntimeContext) {
		context.DebugMode = debugMode
	})
}

func SetDockerDir(dockerDir string) {
	updateRuntimeContext(func(context *RuntimeContext) {
		context.DockerDir = dockerDir
	})
}

// DockerDirFromDefaults resolves the default docker data directory.
func DockerDirFromDefaults() string {
	volumesDir := defaults.DockerVolumesDir()
	if volumesDir == "" {
		return ""
	}
	return volumesDir + "/"
}

// ArchitectureFromRuntime resolves the architecture identifier used by config.
func ArchitectureFromRuntime() string {
	switch runtime.GOARCH {
	case "arm64", "aarch64":
		return "arm64"
	default:
		return "amd64"
	}
}

// BasePathFromEnv resolves the configured base path from GS_BASE_PATH.
func BasePathFromEnv() string {
	switch os.Getenv("GS_BASE_PATH") {
	case "":
		return "/opt/nativeplanet/groundseg"
	default:
		return os.Getenv("GS_BASE_PATH")
	}
}

// DebugModeFromArgs checks command arguments for the dev mode flag.
func DebugModeFromArgs(args []string) bool {
	for _, arg := range args {
		if arg == "dev" {
			return true
		}
	}
	return false
}
