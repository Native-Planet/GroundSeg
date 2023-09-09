package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"goseg/defaults"
	"goseg/structs"
	"os"
	"path/filepath"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

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
	conf := Conf()
	releaseChannel := conf.UpdateBranch
	wgRepo := VersionInfo.Wireguard.Repo
	amdHash := VersionInfo.Wireguard.Amd64Sha256
	armHash := VersionInfo.Wireguard.Arm64Sha256
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
	encoded := base64.StdEncoding.EncodeToString([]byte(privateKey.PublicKey().String() + "\n"))
	publicKey := base64.StdEncoding.EncodeToString([]byte(encoded))
	return privateKey.String(), publicKey, nil
}

// cycle on re-reg
func CycleWgKey() error {
	pub, priv, err := WgKeyGen()
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Couldn't reset WG keys: %v",err))
	}
	if err := UpdateConf(map[string]interface{}{
		"pubkey":  pub,
		"privkey": priv,
	}); err != nil {
		return fmt.Errorf(fmt.Sprintf("Couldn't update new WG keys: %v",err))
	}
	return nil
}