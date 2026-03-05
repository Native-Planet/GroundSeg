package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"groundseg/defaults"
	"groundseg/structs"
)

func resetNetdataSeams() {
	confForNetdata = Config
	getVersionChannelForNetdata = GetVersionChannel
}

func TestCreateDefaultNetdataConfWritesDefaults(t *testing.T) {
	t.Cleanup(resetNetdataSeams)
	oldBasePath := BasePath()
	SetBasePath(t.TempDir())
	t.Cleanup(func() { SetBasePath(oldBasePath) })

	if err := CreateDefaultNetdataConf(); err != nil {
		t.Fatalf("CreateDefaultNetdataConf failed: %v", err)
	}

	path := filepath.Join(BasePath(), "settings", "netdata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read netdata.json failed: %v", err)
	}
	var got structs.NetdataConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal netdata.json failed: %v", err)
	}
	if !reflect.DeepEqual(got, defaults.DefaultNetdataConfig()) {
		t.Fatalf("unexpected netdata defaults: got %+v want %+v", got, defaults.DefaultNetdataConfig())
	}
}

func TestUpdateNetdataConfWritesChannelData(t *testing.T) {
	t.Cleanup(resetNetdataSeams)
	oldBasePath := BasePath()
	SetBasePath(t.TempDir())
	t.Cleanup(func() { SetBasePath(oldBasePath) })

	confForNetdata = func() structs.SysConfig {
		conf := structs.SysConfig{}
		conf.Connectivity.UpdateBranch = "beta"
		return conf
	}
	getVersionChannelForNetdata = func() structs.Channel {
		return structs.Channel{
			Netdata: structs.VersionDetails{
				Repo:        "ghcr.io/nativeplanet/netdata",
				Amd64Sha256: "amd64-hash",
				Arm64Sha256: "arm64-hash",
			},
		}
	}

	if err := UpdateNetdataConf(); err != nil {
		t.Fatalf("UpdateNetdataConf failed: %v", err)
	}

	path := filepath.Join(BasePath(), "settings", "netdata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read netdata.json failed: %v", err)
	}
	var got structs.NetdataConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal netdata.json failed: %v", err)
	}
	if got.NetdataName != "netdata" || got.NetdataVersion != "beta" || got.Repo != "ghcr.io/nativeplanet/netdata" || got.Amd64Sha256 != "amd64-hash" || got.Arm64Sha256 != "arm64-hash" {
		t.Fatalf("unexpected netdata config: %+v", got)
	}
	if got.Port != 19999 || got.Restart != "unless-stopped" || got.SecurityOpt != "apparmor=unconfined" {
		t.Fatalf("unexpected static netdata fields: %+v", got)
	}
	if len(got.Volumes) == 0 {
		t.Fatalf("expected host mount volumes to be populated")
	}
}
