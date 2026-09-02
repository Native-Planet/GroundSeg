package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"groundseg/defaults"
	"groundseg/structs"
	"io"

	"os"
	"path/filepath"
	"strings"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// DecodeStartramWireguardConfig decodes and validates the WireGuard template
// returned by StarTram. In particular, an empty string is valid base64, so it
// must be rejected explicitly before it reaches the on-disk configuration.
func DecodeStartramWireguardConfig(encoded string) (string, error) {
	if strings.TrimSpace(encoded) == "" {
		return "", fmt.Errorf("StarTram returned an empty WireGuard configuration")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode remote WireGuard configuration: %w", err)
	}
	configText := string(decoded)
	if err := ValidateWireguardConfig(configText, true); err != nil {
		return "", fmt.Errorf("invalid remote WireGuard configuration: %w", err)
	}
	return configText, nil
}

// ValidateWireguardConfig performs the minimum structural checks needed to
// avoid replacing a working tunnel configuration with empty or partial data.
func ValidateWireguardConfig(configText string, allowPrivateKeyPlaceholder bool) error {
	if strings.TrimSpace(configText) == "" {
		return fmt.Errorf("configuration is empty")
	}

	section := ""
	hasInterface := false
	hasPeer := false
	hasPrivateKey := false
	hasPeerPublicKey := false
	hasPeerEndpoint := false
	for rawLine := range strings.SplitSeq(configText, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			switch section {
			case "interface":
				hasInterface = true
			case "peer":
				hasPeer = true
			}
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		switch {
		case section == "interface" && key == "privatekey":
			hasPrivateKey = value != "" && (allowPrivateKeyPlaceholder || value != "privkey")
		case section == "peer" && key == "publickey":
			hasPeerPublicKey = value != ""
		case section == "peer" && key == "endpoint":
			hasPeerEndpoint = value != ""
		}
	}

	switch {
	case !hasInterface:
		return fmt.Errorf("missing [Interface] section")
	case !hasPrivateKey:
		return fmt.Errorf("missing private key")
	case !hasPeer:
		return fmt.Errorf("missing [Peer] section")
	case !hasPeerPublicKey:
		return fmt.Errorf("missing peer public key")
	case !hasPeerEndpoint:
		return fmt.Errorf("missing peer endpoint")
	default:
		return nil
	}
}

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
	byteValue, _ := io.ReadAll(configFile)

	if err := json.Unmarshal(byteValue, &wgConf); err != nil {
		return wgConf, err
	}
	return wgConf, nil
}

// write a hardcoded default container conf to disk
func CreateDefaultWGConf() error {
	defaultConfig := defaults.WgConfig
	path := filepath.Join(BasePath, "settings", "wireguard.json")
	return writeJSONDurably(path, &defaultConfig, 0o644)
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
	return writeJSONDurably(path, &newConfig, 0o644)
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
	priv, pub, err := WgKeyGen()
	if err != nil {
		return fmt.Errorf("Couldn't reset WG keys: %w", err)
	}
	if err := UpdateConf(map[string]any{
		"pubkey":  pub,
		"privkey": priv,
	}); err != nil {
		return fmt.Errorf("Couldn't update new WG keys: %w", err)
	}
	return nil
}
