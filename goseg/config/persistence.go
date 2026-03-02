package config

import (
	"encoding/json"
	"fmt"
	"groundseg/defaults"
	"groundseg/structs"
	"io/ioutil"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

type closeFileRuntime struct {
	CloseConfigFileFn func(*os.File) error
}

func NewCloseFileRuntime() closeFileRuntime {
	return closeFileRuntime{
		CloseConfigFileFn: func(file *os.File) error {
			return file.Close()
		},
	}
}

func withDefaultsCloseFileRuntime(runtime closeFileRuntime) closeFileRuntime {
	if runtime.CloseConfigFileFn == nil {
		runtime.CloseConfigFileFn = func(file *os.File) error {
			return file.Close()
		}
	}
	return runtime
}

func closeConfigFileWithRuntime(runtime closeFileRuntime, file *os.File) error {
	runtime = withDefaultsCloseFileRuntime(runtime)
	return runtime.CloseConfigFileFn(file)
}

func closeConfigFile(file *os.File) error {
	return closeConfigFileWithRuntime(NewCloseFileRuntime(), file)
}

func loadSystemConfigOrDefault(file *os.File) (structs.SysConfig, error) {
	var cachedConfig structs.SysConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cachedConfig); err != nil {
		return cachedConfig, fmt.Errorf("decode system config: %w", err)
	}
	return cachedConfig, nil
}

func openOrCreateConfigFile() (*os.File, bool, error) {
	file, err := os.Open(ConfigFilePath())
	if err == nil {
		return file, false, nil
	}

	// create a default if it doesn't exist
	if err = createDefaultConf(); err != nil {
		return nil, false, fmt.Errorf("create default config: %w", err)
	}
	file, err = os.Open(ConfigFilePath())
	if err != nil {
		return nil, false, fmt.Errorf("open config file after bootstrap: %w", err)
	}
	return file, true, nil
}

func seedDefaultConfigRuntimeState() error {
	salt := RandString(32)
	wgPriv, wgPub, err := WgKeyGen()
	if err != nil {
		return fmt.Errorf("generate wireguard keypair: %w", err)
	}
	if err = UpdateConfTyped(
		WithPubkey(wgPub),
		WithPrivkey(wgPriv),
		WithSalt(salt),
		WithKeyfile(SessionKeyPath()),
	); err != nil {
		return fmt.Errorf("update default runtime config state: %w", err)
	}
	return nil
}

func readCurrentBinaryHash() string {
	hash, err := GetSHA256(filepath.Join(BasePath(), "groundseg"))
	if err != nil {
		errmsg := fmt.Sprintf("Error getting binary sha256 hash: %v", err)
		zap.L().Error(errmsg)
		return ""
	}
	zap.L().Info(fmt.Sprintf("Binary sha256 hash: %v", hash))
	return hash
}

func ensureSessionKeyExists() error {
	keyfile, err := os.Stat(SessionKeyPath())
	if err == nil && keyfile.Size() > 0 {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("check session key file: %w", err)
	}
	keyContent := RandString(32)
	if err := ioutil.WriteFile(SessionKeyPath(), []byte(keyContent), 0644); err != nil {
		return fmt.Errorf("couldn't write keyfile! %w", err)
	}
	return nil
}

func persistConfig(configStruct structs.SysConfig) error {
	configMap, err := structToMap(configStruct)
	if err != nil {
		return fmt.Errorf("serialize config structure: %w", err)
	}
	return persistConf(configMap)
}

func structToMap(configStruct structs.SysConfig) (map[string]interface{}, error) {
	configBytes, err := json.Marshal(configStruct)
	if err != nil {
		return nil, fmt.Errorf("marshal config struct: %w", err)
	}
	configMap := make(map[string]interface{})
	if err := json.Unmarshal(configBytes, &configMap); err != nil {
		return nil, fmt.Errorf("unmarshal config map: %w", err)
	}
	return configMap, nil
}

func persistConf(configMap map[string]interface{}) error {
	updatedJSON, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return fmt.Errorf("error encoding JSON: %w", err)
	}
	if err := json.Unmarshal(updatedJSON, &globalConfig); err != nil {
		return fmt.Errorf("error updating global config: %w", err)
	}
	if err := persistConfigJSON(ConfigFilePath(), updatedJSON); err != nil {
		return fmt.Errorf("persist configuration file: %w", err)
	}
	return nil
}

func createDefaultConf() error {
	return createDefaultConfWithRuntime(NewCloseFileRuntime())
}

func createDefaultConfWithRuntime(runtime closeFileRuntime) error {
	defaultConfig := defaults.SysConfig(BasePath())
	path := ConfigFilePath()
	raw, err := json.MarshalIndent(defaultConfig, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal default config: %w", err)
	}
	if err := persistConfigJSONWithRuntime(runtime, path, raw); err != nil {
		return fmt.Errorf("persist default config: %w", err)
	}
	return nil
}

func persistConfigJSONWithRuntime(runtime closeFileRuntime, path string, payload []byte) error {
	if len(payload) == 0 {
		return fmt.Errorf("refusing to persist empty configuration file")
	}
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return fmt.Errorf("ensure config directory exists: %w", err)
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("error creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	if _, err := tmpFile.Write(payload); err != nil {
		_ = closeConfigFileWithRuntime(runtime, tmpFile)
		return fmt.Errorf("error writing temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = closeConfigFileWithRuntime(runtime, tmpFile)
		return fmt.Errorf("error syncing temp file: %w", err)
	}
	if err := closeConfigFileWithRuntime(runtime, tmpFile); err != nil {
		return fmt.Errorf("error closing temp file: %w", err)
	}
	fi, err := os.Stat(tmpPath)
	if err != nil {
		return fmt.Errorf("error checking temp file: %w", err)
	}
	if fi.Size() == 0 {
		return fmt.Errorf("refusing to persist empty configuration file")
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("error moving temp file: %w", err)
	}
	return nil
}

func persistConfigJSON(path string, payload []byte) error {
	return persistConfigJSONWithRuntime(NewCloseFileRuntime(), path, payload)
}
