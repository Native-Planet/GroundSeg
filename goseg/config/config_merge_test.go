package config

import (
	"groundseg/structs"
	"testing"
)

func TestConfigMergeConfigMergeFileIsDirectlyTested(t *testing.T) {
	defaultCfg := structs.SysConfig{
		Setup:        "complete",
		GracefulExit: false,
		EndpointUrl:  "https://default.endpoint",
		ApiVersion:   "v1",
		SwapVal:      64,
		C2cInterval:  1200,
		PenpaiCores:  2,
		PenpaiModels: []structs.Penpai{{ModelName: "groundseg"}},
		PenpaiActive: "groundseg",
	}
	customCfg := structs.SysConfig{
		PwHash:       "existing-hash",
		SwapVal:      0,
		GracefulExit: false,
	}

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
	defaultCfg := structs.SysConfig{
		EndpointUrl:  "https://default.endpoint",
		ApiVersion:   "1.0",
		WgOn:         true,
		Piers:        []string{"alpha"},
		PenpaiCores:  8,
		PenpaiModels: []structs.Penpai{{ModelName: "model-a"}},
		PenpaiActive: "model-a",
	}
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
