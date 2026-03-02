package config

// code for managing groundseg and container configurations

import (
	"fmt"
	"groundseg/defaults"
	"groundseg/structs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	// global settings config (accessed via funcs)
	globalConfig structs.SysConfig
	// cached /retrieve blob
	startramConfig        structs.StartramRetrieve
	startramConfigUpdated time.Time
	// struct of minio passwords
	minIOPasswords = make(map[string]string)
	Ready          = false
	BackupTime     time.Time
	// representation of desired/actual container states
	GSContainers  = make(map[string]structs.ContainerState)
	isEMMCMachine bool
	confMutex     sync.RWMutex
	contMutex     sync.RWMutex
	versMutex     sync.Mutex
	minioPwdMutex sync.Mutex
	startramMu    sync.RWMutex
	initOnce      sync.Once
	initErr       error
	mkdirAllFn    = os.MkdirAll
)

// Initialize loads and validates configuration from disk.
func Initialize() error {
	initOnce.Do(func() {
		initErr = initializeConfig()
	})
	return initErr
}

func initializeConfig() error {
	setDebugMode()
	initializePaths()
	ctx := RuntimeContextSnapshot()
	isEMMCMachine = checkIsEMMCMachine()
	zap.L().Info(fmt.Sprintf("Loading configs from %s", ctx.BasePath))

	file, created, err := openOrCreateConfigFile()
	if err != nil {
		return fmt.Errorf("initialize config file: %w", err)
	}
	defer file.Close()

	if created {
		if err := seedDefaultConfigRuntimeState(); err != nil {
			return fmt.Errorf("seed default runtime state: %w", err)
		}
	}
	cachedConfig, err := loadSystemConfigOrDefault(file)
	if err != nil {
		return fmt.Errorf("load system config: %w", err)
	}

	cachedConfig = mergeConfigs(defaults.SysConfig(ctx.BasePath), cachedConfig)
	cachedConfig.Sessions.Unauthorized = make(map[string]structs.SessionInfo)
	cachedConfig.BinHash = readCurrentBinaryHash()

	if err := persistConfig(cachedConfig); err != nil {
		return fmt.Errorf("persist config: %w", err)
	}

	if err := ensureSessionKeyExists(); err != nil {
		return fmt.Errorf("ensure session key: %w", err)
	}

	currentConf := Conf()
	if currentConf.KeyFile == "" {
		keyPath := SessionKeyPath()
		if err := ensureSessionKeyExists(); err != nil {
			return fmt.Errorf("ensure session key: %w", err)
		}
		if err := UpdateConfTyped(WithKeyfile(keyPath)); err != nil {
			return fmt.Errorf("set session key path: %w", err)
		}
	}

	Ready = true
	return nil
}

func initializePaths() {
	setRuntimeContextBasePath(getBasePath())
}

// return the global conf var
func Conf() structs.SysConfig {
	confMutex.RLock()
	defer confMutex.RUnlock()
	return globalConfig
}

func GetWgPrivkey() string {
	return Conf().Privkey
}

// tell if we're amd64 or arm64
func getArchitecture() string {
	switch runtime.GOARCH {
	case "arm64", "aarch64":
		return "arm64"
	default:
		return "amd64"
	}
}

func getBasePath() string {
	switch os.Getenv("GS_BASE_PATH") {
	case "":
		return "/opt/nativeplanet/groundseg"
	default:
		return os.Getenv("GS_BASE_PATH")
	}
}

func setDebugMode() {
	debugMode := false
	for _, arg := range os.Args[1:] {
		// trigger this with `./groundseg dev`
		if arg == "dev" {
			zap.L().Info("Starting GroundSeg in debug mode")
			debugMode = true
		}
	}
	setRuntimeContextDebugMode(debugMode)
}

// ConfigFilePath returns the current system config path computed from the active context.
func ConfigFilePath() string {
	return filepath.Join(BasePath(), "settings", "system.json")
}

// SessionKeyPath returns the active session key path computed from the active context.
func SessionKeyPath() string {
	return filepath.Join(BasePath(), "settings", "session.key")
}
