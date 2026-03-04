package config

import (
	"groundseg/structs"
	"testing"
)

func TestConfigMergeConfigMergeFileIsDirectlyTested(t *testing.T) {
	defaultCfg := structs.SysConfig{}
	defaultCfg.Runtime.Setup = "complete"
	defaultCfg.Runtime.GracefulExit = false
	defaultCfg.Connectivity.EndpointUrl = "https://default.endpoint"
	defaultCfg.Connectivity.ApiVersion = "v1"
	defaultCfg.Runtime.SwapVal = 64
	defaultCfg.Connectivity.C2cInterval = 1200
	defaultCfg.Penpai.PenpaiCores = 2
	defaultCfg.Penpai.PenpaiModels = []structs.Penpai{{ModelName: "groundseg"}}
	defaultCfg.Penpai.PenpaiActive = "groundseg"

	customCfg := structs.SysConfig{}
	customCfg.AuthSession.PwHash = "existing-hash"
	customCfg.Runtime.SwapVal = 0
	customCfg.Runtime.GracefulExit = false

	merged := MergeConfigs(defaultCfg, customCfg)
	if merged.Runtime.Setup != "complete" {
		t.Fatalf("expected setup to stay as source setup, got %q", merged.Runtime.Setup)
	}
	if merged.AuthSession.PwHash != customCfg.AuthSession.PwHash {
		t.Fatalf("expected custom pw hash to override, got %q", merged.AuthSession.PwHash)
	}
	if merged.Connectivity.EndpointUrl != defaultCfg.Connectivity.EndpointUrl {
		t.Fatalf("expected endpoint default fallback, got %q", merged.Connectivity.EndpointUrl)
	}
}

func TestMergeConfigsFallsBackToDefaultsWhenCustomMissing(t *testing.T) {
	defaultCfg := structs.SysConfig{}
	defaultCfg.Connectivity.EndpointUrl = "https://default.endpoint"
	defaultCfg.Connectivity.ApiVersion = "1.0"
	defaultCfg.Connectivity.WgOn = true
	defaultCfg.Connectivity.Piers = []string{"alpha"}
	defaultCfg.Penpai.PenpaiCores = 8
	defaultCfg.Penpai.PenpaiModels = []structs.Penpai{{ModelName: "model-a"}}
	defaultCfg.Penpai.PenpaiActive = "model-a"
	customCfg := structs.SysConfig{}

	merged := MergeConfigs(defaultCfg, customCfg)
	if merged.Connectivity.EndpointUrl != defaultCfg.Connectivity.EndpointUrl {
		t.Fatalf("expected endpoint default fallback, got %q", merged.Connectivity.EndpointUrl)
	}
	if merged.Connectivity.ApiVersion != defaultCfg.Connectivity.ApiVersion {
		t.Fatalf("expected version fallback, got %q", merged.Connectivity.ApiVersion)
	}
	if merged.Penpai.PenpaiActive != defaultCfg.Penpai.PenpaiActive {
		t.Fatalf("expected active model fallback, got %q", merged.Penpai.PenpaiActive)
	}
	if merged.Penpai.PenpaiCores != defaultCfg.Penpai.PenpaiCores {
		t.Fatalf("expected penpai cores fallback, got %d", merged.Penpai.PenpaiCores)
	}
}
