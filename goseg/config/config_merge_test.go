package config

import (
	"groundseg/structs"
	"testing"
)

func TestConfigMergeConfigMergeFileIsDirectlyTested(t *testing.T) {
	defaultCfg := structs.SysConfig{}
	defaultCfg.Setup = "complete"
	defaultCfg.GracefulExit = false
	defaultCfg.EndpointUrl = "https://default.endpoint"
	defaultCfg.ApiVersion = "v1"
	defaultCfg.SwapVal = 64
	defaultCfg.C2cInterval = 1200
	defaultCfg.PenpaiCores = 2
	defaultCfg.PenpaiModels = []structs.Penpai{{ModelName: "groundseg"}}
	defaultCfg.PenpaiActive = "groundseg"

	customCfg := structs.SysConfig{}
	customCfg.PwHash = "existing-hash"
	customCfg.SwapVal = 0
	customCfg.GracefulExit = false

	merged := MergeConfigs(defaultCfg, customCfg)
	if merged.Setup != "complete" {
		t.Fatalf("expected setup to stay as source setup, got %q", merged.Setup)
	}
	if merged.PwHash != customCfg.PwHash {
		t.Fatalf("expected custom pw hash to override, got %q", merged.PwHash)
	}
	if merged.EndpointUrl != defaultCfg.EndpointUrl {
		t.Fatalf("expected endpoint default fallback, got %q", merged.EndpointUrl)
	}
}

func TestMergeConfigsFallsBackToDefaultsWhenCustomMissing(t *testing.T) {
	defaultCfg := structs.SysConfig{}
	defaultCfg.EndpointUrl = "https://default.endpoint"
	defaultCfg.ApiVersion = "1.0"
	defaultCfg.WgOn = true
	defaultCfg.Piers = []string{"alpha"}
	defaultCfg.PenpaiCores = 8
	defaultCfg.PenpaiModels = []structs.Penpai{{ModelName: "model-a"}}
	defaultCfg.PenpaiActive = "model-a"
	customCfg := structs.SysConfig{}

	merged := MergeConfigs(defaultCfg, customCfg)
	if merged.EndpointUrl != defaultCfg.EndpointUrl {
		t.Fatalf("expected endpoint default fallback, got %q", merged.EndpointUrl)
	}
	if merged.ApiVersion != defaultCfg.ApiVersion {
		t.Fatalf("expected version fallback, got %q", merged.ApiVersion)
	}
	if merged.PenpaiActive != defaultCfg.PenpaiActive {
		t.Fatalf("expected active model fallback, got %q", merged.PenpaiActive)
	}
	if merged.PenpaiCores != defaultCfg.PenpaiCores {
		t.Fatalf("expected penpai cores fallback, got %d", merged.PenpaiCores)
	}
}
