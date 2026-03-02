package config

import (
	"fmt"
	"groundseg/config/wireguardbuilder"
	"groundseg/config/wireguardkeys"
	"groundseg/config/wireguardstore"
	"groundseg/defaults"
	"groundseg/structs"
	"path/filepath"
)

type wireguardSpecsSource func() (structs.SysConfig, structs.Channel)
type wireguardKeyGenerator func() (string, string, error)
type wireguardConfigBuilder func(structs.SysConfig, structs.Channel) structs.WgConfig
type wireguardKeyApplier func(string, string) error

type wireguardRuntime struct {
	loadWGSpecs       wireguardSpecsSource
	generateWgKeypair wireguardKeyGenerator
	applyWgKeys       wireguardKeyApplier
	buildConfig       wireguardConfigBuilder
	wgConfigPath      func() string
	configStore       wireguardstore.WireguardConfigStore
}

func defaultWireguardRuntime() wireguardRuntime {
	return wireguardRuntime{
		loadWGSpecs: func() (structs.SysConfig, structs.Channel) {
			return Conf(), GetVersionChannel()
		},
		generateWgKeypair: WgKeyGen,
		applyWgKeys: func(pub, priv string) error {
			return UpdateConfTyped(
				WithPubkey(pub),
				WithPrivkey(priv),
			)
		},
		buildConfig:  wireguardbuilder.BuildConfig,
		wgConfigPath: func() string { return filepath.Join(BasePath(), "settings", "wireguard.json") },
		configStore:  wireguardstore.FileStore{},
	}
}

// retrieve struct corresponding with urbit json file
func GetWgConf() (structs.WgConfig, error) {
	return getWgConf(defaultWireguardRuntime())
}

func getWgConf(runtime wireguardRuntime) (structs.WgConfig, error) {
	path := runtime.wgConfigPath()
	wgConf, err := runtime.configStore.Load(path)
	if err != nil {
		return structs.WgConfig{}, fmt.Errorf("couldn't open WireGuard config %s: %w", path, err)
	}
	return wgConf, nil
}

// write a hardcoded default container conf to disk
func CreateDefaultWGConf() error {
	return createDefaultWGConf(defaultWireguardRuntime())
}

func createDefaultWGConf(runtime wireguardRuntime) error {
	defaultConfig := defaults.WgConfig
	path := runtime.wgConfigPath()
	if err := runtime.configStore.EnsureDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("failed to create WireGuard settings directory %s: %w", filepath.Dir(path), err)
	}
	if err := runtime.configStore.Save(path, defaultConfig); err != nil {
		return err
	}
	return nil
}

// write a container conf to disk from version server info
func UpdateWGConf() error {
	return updateWGConf(defaultWireguardRuntime())
}

func updateWGConf(runtime wireguardRuntime) error {
	conf, versionInfo := runtime.loadWGSpecs()
	newConfig := runtime.buildConfig(conf, versionInfo)
	path := runtime.wgConfigPath()
	if err := runtime.configStore.Save(path, newConfig); err != nil {
		return fmt.Errorf("failed to write updated WireGuard config %s: %w", path, err)
	}
	return nil
}

// write a container conf to disk from version server info
func WgKeyGen() (privateKeyStr string, publicKeyStr string, err error) {
	privateKeyStr, publicKeyStr, err = wireguardkeys.GenerateKeyPair()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}
	return privateKeyStr, publicKeyStr, nil
}

// cycle on re-reg
func CycleWgKey() error {
	return cycleWGKey(defaultWireguardRuntime())
}

func cycleWGKey(runtime wireguardRuntime) error {
	priv, pub, err := runtime.generateWgKeypair()
	if err != nil {
		return fmt.Errorf("Couldn't reset WG keys: %w", err)
	}
	if err := runtime.applyWgKeys(pub, priv); err != nil {
		return fmt.Errorf("Couldn't update new WG keys: %w", err)
	}
	return nil
}
