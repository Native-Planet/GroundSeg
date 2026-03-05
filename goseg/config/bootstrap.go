package config

// code for managing groundseg and container configurations

import (
	"fmt"
	"groundseg/config/runtimecontext"
	"groundseg/defaults"
	"groundseg/structs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	// global settings config (accessed via funcs)
	globalConfig structs.SysConfig
	// struct of minio passwords
	minIOPasswords = make(map[string]string)
	Ready          = false
	BackupTime     time.Time
	// representation of desired/actual container states
	GSContainers  = make(map[string]structs.ContainerState)
	isEMMCMachine bool
	confMutex     sync.RWMutex
	contMutex     sync.RWMutex
	minioPwdMutex sync.Mutex
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
	if err := defaults.DefaultVersionParseError(); err != nil {
		return fmt.Errorf("default version metadata is invalid: %w", err)
	}

	initializeDebugMode()
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
	cachedConfig.AuthSession.Sessions.Unauthorized = make(map[string]structs.SessionInfo)
	cachedConfig.Runtime.BinHash = readCurrentBinaryHash()

	if err := persistConfig(cachedConfig); err != nil {
		return fmt.Errorf("persist config: %w", err)
	}

	if err := ensureSessionKeyExists(); err != nil {
		return fmt.Errorf("ensure session key: %w", err)
	}

	currentConf := Config()
	if currentConf.AuthSession.KeyFile == "" {
		keyPath := SessionKeyPath()
		if err := ensureSessionKeyExists(); err != nil {
			return fmt.Errorf("ensure session key: %w", err)
		}
		if err := UpdateConfigTyped(WithKeyfile(keyPath)); err != nil {
			return fmt.Errorf("set session key path: %w", err)
		}
	}

	Ready = true
	return nil
}

func initializePaths() {
	SetBasePath(runtimecontext.BasePathFromEnv())
}

// Config returns a snapshot of the package-level config state.
func Config() structs.SysConfig {
	confMutex.RLock()
	defer confMutex.RUnlock()
	return globalConfig
}

func GetWgPrivkey() string {
	return Config().Startram.Privkey
}

func initializeDebugMode() {
	debugMode := runtimecontext.DebugModeFromArgs(os.Args[1:])
	if debugMode {
		zap.L().Info("Starting GroundSeg in debug mode")
	}
	SetDebugMode(debugMode)
}

// ConfigFilePath returns the current system config path computed from the active context.
func ConfigFilePath() string {
	return filepath.Join(BasePath(), "settings", "system.json")
}

// SessionKeyPath returns the active session key path computed from the active context.
func SessionKeyPath() string {
	return filepath.Join(BasePath(), "settings", "session.key")
}
