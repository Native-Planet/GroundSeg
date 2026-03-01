package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"groundseg/defaults"
	"groundseg/structs"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var (
	confForWG               = Conf
	getVersionChannelForWG  = GetVersionChannel
	wgKeyGenForCycle        = WgKeyGen
	updateConfTypedForCycle = UpdateConfTyped
)

// retrieve struct corresponding with urbit json file
func GetWgConf() (structs.WgConfig, error) {
	var wgConf structs.WgConfig
	path := filepath.Join(BasePath, "settings", "wireguard.json")
	configFile, err := os.Open(path)
	if err != nil {
		return wgConf, err
	}
	defer configFile.Close()

	// Read file contents into byte slice
	byteValue, err := ioutil.ReadAll(configFile)
	if err != nil {
		return wgConf, err
	}

	if err := json.Unmarshal(byteValue, &wgConf); err != nil {
		return wgConf, err
	}
	return wgConf, nil
}

// write a hardcoded default container conf to disk
func CreateDefaultWGConf() error {
	defaultConfig := defaults.WgConfig
	path := filepath.Join(BasePath, "settings", "wireguard.json")
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(&defaultConfig); err != nil {
		return err
	}
	return nil
}

// write a container conf to disk from version server info
func UpdateWGConf() error {
	conf := confForWG()
	versionInfo := getVersionChannelForWG()
	releaseChannel := conf.UpdateBranch
	wgRepo := versionInfo.Wireguard.Repo
	amdHash := versionInfo.Wireguard.Amd64Sha256
	armHash := versionInfo.Wireguard.Arm64Sha256
	newConfig := structs.WgConfig{
		WireguardName:    "wireguard",
		WireguardVersion: releaseChannel,
		Repo:             wgRepo,
		Amd64Sha256:      amdHash,
		Arm64Sha256:      armHash,
		CapAdd:           []string{"NET_ADMIN", "SYS_MODULE"},
		Volumes:          []string{"/lib/modules:/lib/modules"},
		Sysctls: struct {
			NetIpv4ConfAllSrcValidMark int `json:"net.ipv4.conf.all.src_valid_mark"`
		}{
			NetIpv4ConfAllSrcValidMark: 1,
		},
	}
	path := filepath.Join(BasePath, "settings", "wireguard.json")
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(&newConfig); err != nil {
		return err
	}
	return nil
}

// wireguard keypair gen
func WgKeyGen() (privateKeyStr string, publicKeyStr string, err error) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %v", err)
	}
	// derive pubkey and use startram encoding
	publicKey := base64.StdEncoding.EncodeToString([]byte(privateKey.PublicKey().String() + "\n"))
	return privateKey.String(), publicKey, nil
}

// cycle on re-reg
func CycleWgKey() error {
	priv, pub, err := wgKeyGenForCycle()
	if err != nil {
		return fmt.Errorf("Couldn't reset WG keys: %v", err)
	}
	if err := updateConfTypedForCycle(
		WithPubkey(pub),
		WithPrivkey(priv),
	); err != nil {
		return fmt.Errorf("Couldn't update new WG keys: %v", err)
	}
	return nil
}
