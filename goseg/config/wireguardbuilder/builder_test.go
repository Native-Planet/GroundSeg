package wireguardbuilder

import (
	"testing"

	"groundseg/defaults"
	"groundseg/structs"
)

func TestBuildConfigUsesDefaultsAndVersionOverrides(t *testing.T) {
	conf := structs.SysConfig{UpdateBranch: "canary"}
	version := structs.Channel{
		Wireguard: structs.VersionDetails{
			Repo:        "wireguard.repo/example",
			Amd64Sha256: "amd-sum",
			Arm64Sha256: "arm-sum",
		},
	}

	config := BuildConfig(conf, version)
	expected := defaults.WgConfig
	expected.WireguardVersion = conf.UpdateBranch
	expected.Repo = version.Wireguard.Repo
	expected.Amd64Sha256 = version.Wireguard.Amd64Sha256
	expected.Arm64Sha256 = version.Wireguard.Arm64Sha256

	if config.WireguardVersion != expected.WireguardVersion ||
		config.Repo != expected.Repo ||
		config.Amd64Sha256 != expected.Amd64Sha256 ||
		config.Arm64Sha256 != expected.Arm64Sha256 {
		t.Fatalf("BuildConfig did not apply version overrides correctly: got %+v want %+v", config, expected)
	}
	if config.WireguardName != defaults.WgConfig.WireguardName ||
		config.Repo != expected.Repo {
		t.Fatalf("BuildConfig changed invariant defaults: got %+v want wireguard name %s", config.WireguardName, defaults.WgConfig.WireguardName)
	}
}

func TestBuildConfigCopiesDefaultSlices(t *testing.T) {
	conf := structs.SysConfig{UpdateBranch: "stable"}
	version := structs.Channel{
		Wireguard: structs.VersionDetails{
			Repo:        "wireguard.repo/example",
			Amd64Sha256: "amd-sum",
			Arm64Sha256: "arm-sum",
		},
	}

	config := BuildConfig(conf, version)
	config.CapAdd[0] = "CHANGED"
	config.Volumes[0] = "CHANGED"

	if defaults.WgConfig.CapAdd[0] == "CHANGED" || defaults.WgConfig.Volumes[0] == "CHANGED" {
		t.Fatal("BuildConfig mutates default wireguard slices")
	}
}
