package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"groundseg/defaults"
	"groundseg/structs"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func resetWGSeams() {
	confForWG = Conf
	getVersionChannelForWG = GetVersionChannel
	wgKeyGenForCycle = WgKeyGen
	updateConfTypedForCycle = UpdateConfTyped
}

func TestCreateDefaultWGConfAndGetWgConf(t *testing.T) {
	t.Cleanup(resetWGSeams)
	oldBasePath := BasePath
	BasePath = t.TempDir()
	t.Cleanup(func() { BasePath = oldBasePath })

	if err := CreateDefaultWGConf(); err != nil {
		t.Fatalf("CreateDefaultWGConf failed: %v", err)
	}
	got, err := GetWgConf()
	if err != nil {
		t.Fatalf("GetWgConf failed: %v", err)
	}
	if !reflect.DeepEqual(got, defaults.WgConfig) {
		t.Fatalf("unexpected default wireguard config: got %+v want %+v", got, defaults.WgConfig)
	}
}

func TestUpdateWGConfWritesVersionData(t *testing.T) {
	t.Cleanup(resetWGSeams)
	oldBasePath := BasePath
	BasePath = t.TempDir()
	t.Cleanup(func() { BasePath = oldBasePath })

	confForWG = func() structs.SysConfig { return structs.SysConfig{UpdateBranch: "latest"} }
	getVersionChannelForWG = func() structs.Channel {
		return structs.Channel{
			Wireguard: structs.VersionDetails{
				Repo:        "ghcr.io/nativeplanet/wireguard",
				Amd64Sha256: "amd-hash",
				Arm64Sha256: "arm-hash",
			},
		}
	}

	if err := UpdateWGConf(); err != nil {
		t.Fatalf("UpdateWGConf failed: %v", err)
	}
	path := filepath.Join(BasePath, "settings", "wireguard.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read wireguard.json failed: %v", err)
	}
	var got structs.WgConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal wireguard.json failed: %v", err)
	}
	if got.WireguardName != "wireguard" || got.WireguardVersion != "latest" || got.Repo != "ghcr.io/nativeplanet/wireguard" || got.Amd64Sha256 != "amd-hash" || got.Arm64Sha256 != "arm-hash" {
		t.Fatalf("unexpected wireguard config: %+v", got)
	}
	if got.Sysctls.NetIpv4ConfAllSrcValidMark != 1 {
		t.Fatalf("unexpected sysctl value: %+v", got.Sysctls)
	}
}

func TestWgKeyGenProducesValidKeys(t *testing.T) {
	priv, pub, err := WgKeyGen()
	if err != nil {
		t.Fatalf("WgKeyGen failed: %v", err)
	}
	if _, err := wgtypes.ParseKey(priv); err != nil {
		t.Fatalf("invalid private key generated: %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(pub)
	if err != nil {
		t.Fatalf("public key should be base64 encoded: %v", err)
	}
	if !strings.HasSuffix(string(decoded), "\n") {
		t.Fatalf("expected public key payload to end with newline, got %q", string(decoded))
	}
}

func TestCycleWgKeyUpdatesConfFromGeneratedKeys(t *testing.T) {
	t.Cleanup(resetWGSeams)

	wgKeyGenForCycle = func() (string, string, error) {
		return "private-key", "public-key", nil
	}

	var capturedPub, capturedPriv string
	updateConfTypedForCycle = func(opts ...ConfUpdateOption) error {
		patch := &ConfPatch{}
		for _, opt := range opts {
			opt(patch)
		}
		if patch.Pubkey != nil {
			capturedPub = *patch.Pubkey
		}
		if patch.Privkey != nil {
			capturedPriv = *patch.Privkey
		}
		return nil
	}

	if err := CycleWgKey(); err != nil {
		t.Fatalf("CycleWgKey failed: %v", err)
	}
	if capturedPub != "public-key" || capturedPriv != "private-key" {
		t.Fatalf("unexpected captured keys: pub=%s priv=%s", capturedPub, capturedPriv)
	}
}

func TestCycleWgKeyErrors(t *testing.T) {
	t.Cleanup(resetWGSeams)

	wgKeyGenForCycle = func() (string, string, error) {
		return "", "", errors.New("gen failed")
	}
	if err := CycleWgKey(); err == nil || !strings.Contains(err.Error(), "Couldn't reset WG keys") {
		t.Fatalf("expected keygen failure, got %v", err)
	}

	wgKeyGenForCycle = func() (string, string, error) {
		return "priv", "pub", nil
	}
	updateConfTypedForCycle = func(...ConfUpdateOption) error {
		return errors.New("update failed")
	}
	if err := CycleWgKey(); err == nil || !strings.Contains(err.Error(), "Couldn't update new WG keys") {
		t.Fatalf("expected update failure, got %v", err)
	}
}
