package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"groundseg/defaults"
	"groundseg/structs"
)

func resetMcSeams() {
	confForMc = Conf
	getVersionChannelForMc = GetVersionChannel
}

func TestCreateDefaultMcConfWritesDefaults(t *testing.T) {
	t.Cleanup(resetMcSeams)
	oldBasePath := BasePath()
	SetBasePath(t.TempDir())
	t.Cleanup(func() { SetBasePath(oldBasePath) })

	if err := CreateDefaultMcConf(); err != nil {
		t.Fatalf("CreateDefaultMcConf failed: %v", err)
	}

	path := filepath.Join(BasePath(), "settings", "mc.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mc.json failed: %v", err)
	}
	var got structs.McConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal mc.json failed: %v", err)
	}
	if got != defaults.McConfig {
		t.Fatalf("unexpected mc defaults: got %+v want %+v", got, defaults.McConfig)
	}
}

func TestUpdateMcConfWritesVersionChannelData(t *testing.T) {
	t.Cleanup(resetMcSeams)
	oldBasePath := BasePath()
	SetBasePath(t.TempDir())
	t.Cleanup(func() { SetBasePath(oldBasePath) })

	confForMc = func() structs.SysConfig {
		conf := structs.SysConfig{}
		conf.UpdateBranch = "edge"
		return conf
	}
	getVersionChannelForMc = func() structs.Channel {
		return structs.Channel{
			Miniomc: structs.VersionDetails{
				Repo:        "ghcr.io/nativeplanet/mc",
				Amd64Sha256: "amd-hash",
				Arm64Sha256: "arm-hash",
			},
		}
	}

	if err := UpdateMcConf(); err != nil {
		t.Fatalf("UpdateMcConf failed: %v", err)
	}

	path := filepath.Join(BasePath(), "settings", "mc.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mc.json failed: %v", err)
	}
	var got structs.McConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal mc.json failed: %v", err)
	}
	if got.McName != "minio_client" || got.McVersion != "edge" || got.Repo != "ghcr.io/nativeplanet/mc" || got.Amd64Sha256 != "amd-hash" || got.Arm64Sha256 != "arm-hash" {
		t.Fatalf("unexpected mc config: %+v", got)
	}
}
