package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"groundseg/structs"
)

func TestRandStringReturnsEncodedData(t *testing.T) {
	got := RandString(32)
	if got == "" {
		t.Fatal("expected non-empty random string")
	}
	// base64 output should be longer than input bytes.
	if len(got) <= 32 {
		t.Fatalf("expected encoded string longer than 32, got %d", len(got))
	}
}

func TestGetSHA256(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hash.txt")
	if err := os.WriteFile(path, []byte("groundseg"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	got, err := GetSHA256(path)
	if err != nil {
		t.Fatalf("GetSHA256 returned error: %v", err)
	}
	const want = "62bdeec0e24a451b75e5f518384e1ec2856cb5dc2366913d65bb9313899bc2e2"
	if got != want {
		t.Fatalf("unexpected hash: want %s got %s", want, got)
	}
}

func TestMergeConfigsNewInstallSetupAndSalt(t *testing.T) {
	defaultCfg := structs.SysConfig{
		Setup:       "complete",
		EndpointUrl: "default.endpoint",
	}
	customCfg := structs.SysConfig{
		PwHash: "",
	}

	merged := mergeConfigs(defaultCfg, customCfg)
	if merged.Setup != "start" {
		t.Fatalf("expected setup=start for new install, got %q", merged.Setup)
	}
	if merged.Salt == "" {
		t.Fatal("expected salt to be generated for new install")
	}
}

func TestMergeConfigsPrefersCustomEndpoint(t *testing.T) {
	defaultCfg := structs.SysConfig{EndpointUrl: "default.endpoint"}
	customCfg := structs.SysConfig{EndpointUrl: "custom.endpoint", PwHash: "hash"}

	merged := mergeConfigs(defaultCfg, customCfg)
	if merged.EndpointUrl != "custom.endpoint" {
		t.Fatalf("expected custom endpoint to win, got %q", merged.EndpointUrl)
	}
}

func TestMergeConfigsMigrationUsesDefaultsAndGeneratedSaltWhenMissing(t *testing.T) {
	defaultCfg := structs.SysConfig{
		GracefulExit:   true,
		Setup:          "complete",
		LastKnownMDNS:  "default-mdns",
		EndpointUrl:    "https://default.endpoint",
		ApiVersion:     "v2",
		NetCheck:       "default-netcheck",
		UpdateMode:     "default-mode",
		UpdateUrl:      "https://default-update",
		UpdateBranch:   "main",
		SwapVal:        99,
		SwapFile:       "/default/swap",
		KeyFile:        "/default/key",
		DockerData:     "/default/docker-data",
		WgOn:           true,
		WgRegistered:   true,
		UpdateInterval: 15,
		C2cInterval:    2000,
		GsVersion:      "v2.0",
		CfgDir:         "/default/cfg",
		BinHash:        "bin-default",
		Pubkey:         "default-pub",
		Privkey:        "default-priv",
		PwHash:         "old-hash",
		DiskWarning: map[string]structs.DiskWarning{
			"data": {Eighty: true},
		},
		PenpaiModels: []structs.Penpai{
			{ModelName: "default-a"},
			{ModelName: "default-b"},
		},
		PenpaiActive:  "default-a",
		Disable502:    true,
		SnapTime:      14,
		PenpaiAllow:   true,
		PenpaiRunning: true,
		PenpaiCores:   8,
	}
	defaultCfg.StartramSetReminder = struct {
		One   bool `json:"one"`
		Three bool `json:"three"`
		Seven bool `json:"seven"`
	}{
		One:   true,
		Three: true,
		Seven: true,
	}
	defaultCfg.Sessions = struct {
		Authorized   map[string]structs.SessionInfo `json:"authorized"`
		Unauthorized map[string]structs.SessionInfo `json:"unauthorized"`
	}{
		Authorized:   map[string]structs.SessionInfo{"default": {Hash: "authorized-default"}},
		Unauthorized: map[string]structs.SessionInfo{"default": {Hash: "unauthorized-default"}},
	}
	defaultCfg.LinuxUpdates = struct {
		Value    int    `json:"value"`
		Interval string `json:"interval"`
	}{
		Value:    1,
		Interval: "1m",
	}

	// Ensure migration path is exercised and defaults are selected for empty custom fields.
	customCfg := structs.SysConfig{
		PwHash: "legacy-pwhash",
		Salt:   "custom-salt",
	}

	merged := MergeConfigs(defaultCfg, customCfg)
	if merged.Setup != "complete" {
		t.Fatalf("expected setup=complete for migration path, got %q", merged.Setup)
	}
	if merged.LastKnownMDNS != defaultCfg.LastKnownMDNS {
		t.Fatalf("expected last known mdns to use default when custom is empty")
	}
	if merged.EndpointUrl != defaultCfg.EndpointUrl || merged.ApiVersion != defaultCfg.ApiVersion {
		t.Fatalf("expected defaults for empty custom scalar fields")
	}
	if merged.PwHash != customCfg.PwHash {
		t.Fatalf("expected custom pwhash to be preserved when provided")
	}
	if merged.StartramSetReminder != defaultCfg.StartramSetReminder {
		t.Fatalf("expected startram reminders to merge using OR semantics")
	}
	if merged.Sessions.Authorized == nil || merged.Sessions.Unauthorized == nil {
		t.Fatalf("expected session maps to fall back to defaults when custom is nil")
	}
	if merged.CfgDir != defaultCfg.CfgDir || merged.BinHash != defaultCfg.BinHash {
		t.Fatalf("expected default config values when custom values are empty")
	}
	if merged.Disable502 != defaultCfg.Disable502 {
		t.Fatalf("expected default Disable502 when custom is false")
	}
	if merged.PenpaiAllow != true {
		t.Fatalf("expected custom false penpai allow to inherit from default true")
	}
	if merged.PenpaiRunning != false {
		t.Fatalf("expected custom false penpai running to remain false")
	}
	if merged.PenpaiCores != defaultCfg.PenpaiCores {
		t.Fatalf("expected default penpai cores when custom is zero")
	}
	if merged.PenpaiActive != defaultCfg.PenpaiActive {
		t.Fatalf("expected default penpai active when custom value empty")
	}
	if merged.DiskWarning != nil {
		t.Fatalf("expected custom nil disk warning to be assigned as-is")
	}
	if merged.PenpaiModels == nil || len(merged.PenpaiModels) != len(defaultCfg.PenpaiModels) {
		t.Fatalf("expected latest penpai model list to come from default config")
	}
	if merged.Salt != customCfg.Salt {
		t.Fatalf("expected custom salt to be propagated when default salt is not set and pwhash present")
	}
}

func TestMergeConfigsCustomOverridesAndValidModelSelection(t *testing.T) {
	defaultCfg := structs.SysConfig{
		GracefulExit:   true,
		LastKnownMDNS:  "default-mdns",
		Setup:          "default-setup",
		EndpointUrl:    "https://default.endpoint",
		ApiVersion:     "v2",
		Piers:          []string{"default"},
		NetCheck:       "default-netcheck",
		UpdateMode:     "default-mode",
		UpdateUrl:      "https://default-update",
		UpdateBranch:   "default",
		SwapVal:        10,
		SwapFile:       "/default/swap",
		KeyFile:        "/default/key",
		DockerData:     "/default/docker",
		WgOn:           true,
		WgRegistered:   true,
		C2cInterval:    1200,
		GsVersion:      "default-gs",
		CfgDir:         "/default/cfg",
		UpdateInterval: 30,
		BinHash:        "default-bin",
		Pubkey:         "default-pub",
		Privkey:        "default-priv",
		PenpaiModels: []structs.Penpai{
			{ModelName: "alpha"},
			{ModelName: "beta"},
		},
		PenpaiActive: "alpha",
		Disable502:   false,
		SnapTime:     7,
		PenpaiAllow:  true,
	}
	defaultCfg.StartramSetReminder = struct {
		One   bool `json:"one"`
		Three bool `json:"three"`
		Seven bool `json:"seven"`
	}{
		One:   true,
		Three: true,
		Seven: true,
	}
	defaultCfg.DiskWarning = map[string]structs.DiskWarning{
		"default": {Eighty: true},
	}
	defaultCfg.Sessions = struct {
		Authorized   map[string]structs.SessionInfo `json:"authorized"`
		Unauthorized map[string]structs.SessionInfo `json:"unauthorized"`
	}{
		Authorized:   map[string]structs.SessionInfo{"default": {Hash: "default-auth"}},
		Unauthorized: map[string]structs.SessionInfo{"default": {Hash: "default-unauth"}},
	}
	defaultCfg.LinuxUpdates = struct {
		Value    int    `json:"value"`
		Interval string `json:"interval"`
	}{
		Value:    10,
		Interval: "1m",
	}

	customCfg := structs.SysConfig{
		GracefulExit:  false,
		LastKnownMDNS: "custom-mdns",
		Setup:         "custom-setup",
		PwHash:        "present",
		EndpointUrl:   "https://custom.endpoint",
		ApiVersion:    "v10",
		Piers:         []string{"a", "b", "c"},
		NetCheck:      "custom-netcheck",
		UpdateMode:    "poll",
		UpdateUrl:     "https://custom-update",
		UpdateBranch:  "stable",
		SwapVal:       20,
		SwapFile:      "/custom/swap",
		KeyFile:       "/custom/key",
		DockerData:    "/custom/docker",
		WgOn:          false,
		WgRegistered:  false,
		Sessions: struct {
			Authorized   map[string]structs.SessionInfo `json:"authorized"`
			Unauthorized map[string]structs.SessionInfo `json:"unauthorized"`
		}{
			Authorized:   map[string]structs.SessionInfo{"custom": {Hash: "auth"}},
			Unauthorized: map[string]structs.SessionInfo{"custom": {Hash: "unauth"}},
		},
		LinuxUpdates: struct {
			Value    int    `json:"value"`
			Interval string `json:"interval"`
		}{Value: 2, Interval: "3m"},
		C2cInterval:          3000,
		GsVersion:            "v10",
		CfgDir:               "/custom/cfg",
		UpdateInterval:       60,
		BinHash:              "custom-bin",
		Pubkey:               "custom-pub",
		Privkey:              "custom-priv",
		PenpaiRunning:        true,
		PenpaiCores:          4,
		Salt:                 "custom-salt",
		PenpaiAllow:          true,
		PenpaiActive:         "beta",
		Disable502:           true,
		SnapTime:             42,
		RemoteBackupPassword: "rpw",
	}

	merged := MergeConfigs(defaultCfg, customCfg)

	if !merged.GracefulExit {
		t.Fatal("expected merged graceful exit from custom false or default true to be true")
	}
	if merged.LastKnownMDNS != customCfg.LastKnownMDNS || merged.EndpointUrl != customCfg.EndpointUrl {
		t.Fatalf("expected non-empty custom string fields to override defaults")
	}
	if !reflect.DeepEqual(merged.Piers, customCfg.Piers) {
		t.Fatalf("expected custom piers to override defaults")
	}
	if merged.SwapVal != customCfg.SwapVal || merged.SwapFile != customCfg.SwapFile {
		t.Fatalf("expected custom swap fields to override defaults")
	}
	if merged.StartramSetReminder != defaultCfg.StartramSetReminder {
		t.Fatal("expected startram reminders to keep default false/true semantics with fallback")
	}
	if !reflect.DeepEqual(merged.Sessions.Authorized, customCfg.Sessions.Authorized) {
		t.Fatalf("expected custom session maps to override defaults")
	}
	if !reflect.DeepEqual(merged.LinuxUpdates, customCfg.LinuxUpdates) {
		t.Fatalf("expected custom linux update settings to override defaults")
	}
	if merged.PenpaiCores != customCfg.PenpaiCores {
		t.Fatalf("expected non-zero custom penpai cores to override default")
	}
	if merged.PenpaiModels[0].ModelName != defaultCfg.PenpaiModels[0].ModelName {
		t.Fatalf("expected default penpai models to remain source of truth")
	}
	if merged.PenpaiActive != customCfg.PenpaiActive {
		t.Fatalf("expected valid custom active penpai model to be selected")
	}
	if merged.PwHash != customCfg.PwHash || merged.CfgDir != customCfg.CfgDir {
		t.Fatalf("expected custom scalar overrides to propagate")
	}
	if merged.Disable502 != customCfg.Disable502 || merged.SnapTime != customCfg.SnapTime {
		t.Fatalf("expected custom booleans and snap time to propagate when set")
	}
	if merged.RemoteBackupPassword != customCfg.RemoteBackupPassword {
		t.Fatalf("expected remote backup password to propagate")
	}
}

func TestGetStoragePathReturnsErrorForUnknownOperation(t *testing.T) {
	_, err := GetStoragePath("unknown")
	if err == nil {
		t.Fatal("expected error for unknown storage operation")
	}
	if !strings.Contains(err.Error(), "invalid storage operation") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestGetStoragePathReturnsErrorWhenDirectoryCreationFails(t *testing.T) {
	restoreIsEMMC := isEMMCMachine
	restoreBasePath := os.Getenv("GS_BASE_PATH")
	restoreMkdir := mkdirAllFn

	isEMMCMachine = false
	t.Setenv("GS_BASE_PATH", t.TempDir())
	mkdirAllFn = func(_ string, _ os.FileMode) error {
		return errors.New("permission denied")
	}
	t.Cleanup(func() {
		isEMMCMachine = restoreIsEMMC
		_ = os.Setenv("GS_BASE_PATH", restoreBasePath)
		mkdirAllFn = restoreMkdir
	})

	_, err := GetStoragePath("logs")
	if err == nil {
		t.Fatal("expected GetStoragePath to fail when MkdirAll fails")
	}
	if !strings.Contains(err.Error(), "failed to create storage directory") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestCreateDefaultConfPropagatesCloseError(t *testing.T) {
	oldBasePath := BasePath()
	closeErr := errors.New("close failed")

	SetBasePath(t.TempDir())
	t.Cleanup(func() {
		SetBasePath(oldBasePath)
	})

	runtime := NewCloseFileRuntime()
	runtime.CloseConfigFileFn = func(_ *os.File) error {
		return closeErr
	}
	if err := createDefaultConfWithRuntime(runtime); err == nil || !errors.Is(err, closeErr) {
		t.Fatalf("expected close error wrapped, got: %v", err)
	}
}

type countingMerger struct {
	configMerged bool
}

func (m *countingMerger) Merge(_ structs.SysConfig, _ structs.SysConfig) structs.SysConfig {
	m.configMerged = true
	return structs.SysConfig{Setup: "counted"}
}

func TestMergeConfigsDelegatesToConfiguredMerger(t *testing.T) {
	merger := &countingMerger{}
	restoreMerger := configMerger
	t.Cleanup(func() {
		configMerger = restoreMerger
	})

	SetConfigMerger(merger)
	merged := MergeConfigs(structs.SysConfig{}, structs.SysConfig{})
	if !merger.configMerged {
		t.Fatalf("expected custom merger to be invoked")
	}
	if merged.Setup != "counted" {
		t.Fatalf("unexpected merged config: got %q", merged.Setup)
	}
}

func TestSetConfigMergerIgnoresNilMerger(t *testing.T) {
	restoreMerger := configMerger
	t.Cleanup(func() {
		configMerger = restoreMerger
	})

	SetConfigMerger(nil)
	if configMerger != restoreMerger {
		t.Fatalf("expected nil merger to be ignored")
	}
}

func TestMergeConfigsFallbackWhenMergerNilUsesDefaultMerger(t *testing.T) {
	restoreMerger := configMerger
	t.Cleanup(func() {
		configMerger = restoreMerger
	})

	configMerger = nil

	defaultConfig := structs.SysConfig{
		GracefulExit:  true,
		LastKnownMDNS: "default.mdns",
		Setup:         "complete",
		PwHash:        "default-hash",
		EndpointUrl:   "https://default",
		UpdateMode:    "poll",
		PenpaiModels:  []structs.Penpai{{ModelName: "default-a"}},
		PenpaiActive:  "default-a",
	}
	defaultConfig.Sessions.Authorized = map[string]structs.SessionInfo{"default": {Hash: "hash"}}
	defaultConfig.Sessions.Unauthorized = map[string]structs.SessionInfo{"default-unauth": {Hash: "hash"}}

	customConfig := structs.SysConfig{
		GracefulExit: false,
		UpdateMode:   "",
		PwHash:       "",
	}

	merged := MergeConfigs(defaultConfig, customConfig)
	if merged.GracefulExit != true {
		t.Fatalf("expected graceful exit to keep default true")
	}
	if merged.LastKnownMDNS != defaultConfig.LastKnownMDNS {
		t.Fatalf("expected default mdns when custom value missing")
	}
	if merged.EndpointUrl != defaultConfig.EndpointUrl {
		t.Fatalf("expected default endpoint when custom missing")
	}
	if merged.Salt == "" {
		t.Fatalf("expected generated salt when default config has empty salt and custom pw hash is empty")
	}
}

func TestMergeConfigsAppliesDefaultsAndCustomOverrides(t *testing.T) {
	defaultConfig := structs.SysConfig{
		GracefulExit:   false,
		LastKnownMDNS:  "default-mdns",
		Setup:          "migrated",
		PwHash:         "default-hash",
		EndpointUrl:    "https://default-endpoint",
		ApiVersion:     "1.0",
		Piers:          []string{"default-pier"},
		NetCheck:       "default-netcheck",
		UpdateMode:     "default",
		UpdateUrl:      "https://updates.example.com",
		UpdateBranch:   "main",
		SwapVal:        1,
		SwapFile:       "/tmp/default-swap",
		KeyFile:        "/tmp/default-key",
		DockerData:     "default-docker",
		WgOn:           false,
		WgRegistered:   false,
		DiskWarning:    map[string]structs.DiskWarning{"default": {Eighty: true}},
		C2cInterval:    1000,
		GsVersion:      "v1",
		CfgDir:         "/default/cfg",
		UpdateInterval: 20,
		BinHash:        "default-bin",
		Pubkey:         "default-pub",
		Privkey:        "default-priv",
		Salt:           "default-salt",
		PenpaiAllow:    true,
		PenpaiRunning:  false,
		PenpaiCores:    4,
		PenpaiModels:   []structs.Penpai{{ModelName: "mymodel"}},
		PenpaiActive:   "mymodel",
		DisableSlsa:    true,
		Disable502:     true,
		SnapTime:       12,
	}
	defaultConfig.Sessions.Authorized = map[string]structs.SessionInfo{"user": {Hash: "old"}}
	defaultConfig.Sessions.Unauthorized = map[string]structs.SessionInfo{"anon": {Hash: "anon-old"}}
	defaultConfig.LinuxUpdates = struct {
		Value    int    `json:"value"`
		Interval string `json:"interval"`
	}{
		Value:    1,
		Interval: "2m",
	}
	defaultConfig.StartramSetReminder = struct {
		One   bool `json:"one"`
		Three bool `json:"three"`
		Seven bool `json:"seven"`
	}{
		One:   false,
		Three: false,
		Seven: false,
	}

	customConfig := structs.SysConfig{
		GracefulExit:  true,
		LastKnownMDNS: "", // force default
		EndpointUrl:   "https://custom-endpoint",
		PwHash:        "", // triggers new install logic
		SwapVal:       0,  // fallback to default
		WgOn:          true,
		WgRegistered:  true,
		DiskWarning:   map[string]structs.DiskWarning{"custom": {}},
		PenpaiAllow:   true,
		PenpaiRunning: true,
		PenpaiCores:   0,         // fallback
		PenpaiModels:  nil,       // ignored
		PenpaiActive:  "invalid", // should fallback
		Disable502:    false,     // default should stay true
	}
	customConfig.StartramSetReminder = struct {
		One   bool `json:"one"`
		Three bool `json:"three"`
		Seven bool `json:"seven"`
	}{
		One:   true,
		Three: true,
		Seven: false,
	}
	customConfig.Sessions.Unauthorized = map[string]structs.SessionInfo{"new": {Hash: "new"}}

	merged := mergeConfigs(defaultConfig, customConfig)
	if merged.GracefulExit != true {
		t.Fatalf("expected merged graceful exit true")
	}
	if merged.LastKnownMDNS != defaultConfig.LastKnownMDNS {
		t.Fatalf("expected default mdns on empty custom value")
	}
	if merged.EndpointUrl != customConfig.EndpointUrl {
		t.Fatalf("expected custom endpoint to win")
	}
	if merged.ApiVersion != defaultConfig.ApiVersion {
		t.Fatalf("expected default API version when custom empty")
	}
	if merged.Piers[0] != defaultConfig.Piers[0] {
		t.Fatalf("expected default piers for empty custom list")
	}
	if merged.NetCheck != defaultConfig.NetCheck {
		t.Fatalf("expected default netcheck when custom empty")
	}
	if !merged.WgOn || !merged.WgRegistered {
		t.Fatalf("expected bool-or flags to include custom true values")
	}
	if merged.StartramSetReminder.One != true || !merged.StartramSetReminder.Three || merged.StartramSetReminder.Seven {
		t.Fatalf("unexpected startram reminder merge state")
	}
	if merged.PenpaiCores != defaultConfig.PenpaiCores {
		t.Fatalf("expected default penpai cores when custom is zero")
	}
	if merged.PenpaiActive != defaultConfig.PenpaiActive {
		t.Fatalf("expected default penpai active when custom active is invalid")
	}
	if merged.Disable502 != true {
		t.Fatalf("expected default disable502 when custom is false")
	}
	if merged.Setup != "start" {
		t.Fatalf("expected setup=start when custom pw hash is missing")
	}
	if merged.PwHash != defaultConfig.PwHash {
		t.Fatalf("expected default pwhash when custom missing")
	}
	if merged.Salt == "" {
		t.Fatalf("expected generated salt when starting setup")
	}
	if !merged.PenpaiAllow {
		t.Fatalf("expected penpai allow true due custom override")
	}
	if !merged.PenpaiRunning {
		t.Fatalf("expected penpai running from custom")
	}
	if len(merged.PenpaiModels) != len(defaultConfig.PenpaiModels) || merged.PenpaiModels[0].ModelName != defaultConfig.PenpaiModels[0].ModelName {
		t.Fatalf("expected default penpai models")
	}
	if !reflect.DeepEqual(merged.Sessions.Unauthorized, customConfig.Sessions.Unauthorized) {
		t.Fatalf("expected custom unauthorized sessions")
	}
	if !reflect.DeepEqual(merged.Sessions.Authorized, defaultConfig.Sessions.Authorized) {
		t.Fatalf("expected default authorized sessions")
	}
	if !reflect.DeepEqual(merged.DiskWarning, customConfig.DiskWarning) {
		t.Fatalf("expected custom disk warning")
	}
	if merged.C2cInterval != defaultConfig.C2cInterval || merged.GsVersion != defaultConfig.GsVersion || merged.CfgDir != defaultConfig.CfgDir {
		t.Fatalf("expected default scalar fields when custom zero/empty")
	}
}
