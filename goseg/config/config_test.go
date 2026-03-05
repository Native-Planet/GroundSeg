package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
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
	defaultCfg := structs.SysConfig{}
	defaultCfg.Runtime.Setup = "complete"
	defaultCfg.Connectivity.EndpointURL = "default.endpoint"
	customCfg := structs.SysConfig{}
	customCfg.AuthSession.PwHash = ""

	merged := mergeConfigs(defaultCfg, customCfg)
	if merged.Runtime.Setup != "start" {
		t.Fatalf("expected setup=start for new install, got %q", merged.Runtime.Setup)
	}
	if merged.AuthSession.Salt == "" {
		t.Fatal("expected salt to be generated for new install")
	}
}

func TestMergeConfigsPrefersCustomEndpoint(t *testing.T) {
	defaultCfg := structs.SysConfig{}
	defaultCfg.Connectivity.EndpointURL = "default.endpoint"
	customCfg := structs.SysConfig{}
	customCfg.Connectivity.EndpointURL = "custom.endpoint"
	customCfg.AuthSession.PwHash = "hash"

	merged := mergeConfigs(defaultCfg, customCfg)
	if merged.Connectivity.EndpointURL != "custom.endpoint" {
		t.Fatalf("expected custom endpoint to win, got %q", merged.Connectivity.EndpointURL)
	}
}

func TestMergeConfigsMigrationUsesDefaultsAndGeneratedSaltWhenMissing(t *testing.T) {
	defaultCfg := structs.SysConfig{}
	defaultCfg.Runtime.GracefulExit = true
	defaultCfg.Runtime.Setup = "complete"
	defaultCfg.Runtime.LastKnownMDNS = "default-mdns"
	defaultCfg.Connectivity.EndpointURL = "https://default.endpoint"
	defaultCfg.Connectivity.ApiVersion = "v2"
	defaultCfg.Connectivity.NetCheck = "default-netcheck"
	defaultCfg.Connectivity.UpdateMode = "default-mode"
	defaultCfg.Connectivity.UpdateURL = "https://default-update"
	defaultCfg.Connectivity.UpdateBranch = "main"
	defaultCfg.Runtime.SwapVal = 99
	defaultCfg.Runtime.SwapFile = "/default/swap"
	defaultCfg.AuthSession.KeyFile = "/default/key"
	defaultCfg.Runtime.DockerData = "/default/docker-data"
	defaultCfg.Connectivity.WgOn = true
	defaultCfg.Connectivity.WgRegistered = true
	defaultCfg.Runtime.UpdateInterval = 15
	defaultCfg.Connectivity.C2CInterval = 2000
	defaultCfg.Runtime.GsVersion = "v2.0"
	defaultCfg.Runtime.CfgDir = "/default/cfg"
	defaultCfg.Runtime.BinHash = "bin-default"
	defaultCfg.Startram.Pubkey = "default-pub"
	defaultCfg.Startram.Privkey = "default-priv"
	defaultCfg.AuthSession.PwHash = "old-hash"
	defaultCfg.Connectivity.DiskWarning = map[string]structs.DiskWarning{
		"data": {Eighty: true},
	}
	defaultCfg.Penpai.PenpaiModels = []structs.Penpai{
		{ModelName: "default-a"},
		{ModelName: "default-b"},
	}
	defaultCfg.Penpai.PenpaiActive = "default-a"
	defaultCfg.Runtime.Disable502 = true
	defaultCfg.Runtime.SnapTime = 14
	defaultCfg.Penpai.PenpaiAllow = true
	defaultCfg.Penpai.PenpaiRunning = true
	defaultCfg.Penpai.PenpaiCores = 8
	defaultCfg.Startram.StartramSetReminder = struct {
		One   bool `json:"one"`
		Three bool `json:"three"`
		Seven bool `json:"seven"`
	}{
		One:   true,
		Three: true,
		Seven: true,
	}
	defaultCfg.AuthSession.Sessions = struct {
		Authorized   map[string]structs.SessionInfo `json:"authorized"`
		Unauthorized map[string]structs.SessionInfo `json:"unauthorized"`
	}{
		Authorized:   map[string]structs.SessionInfo{"default": {Hash: "authorized-default"}},
		Unauthorized: map[string]structs.SessionInfo{"default": {Hash: "unauthorized-default"}},
	}
	defaultCfg.Runtime.LinuxUpdates = struct {
		Value    int    `json:"value"`
		Interval string `json:"interval"`
	}{
		Value:    1,
		Interval: "1m",
	}

	// Ensure migration path is exercised and defaults are selected for empty custom fields.
	customCfg := structs.SysConfig{}
	customCfg.AuthSession.PwHash = "legacy-pwhash"
	customCfg.AuthSession.Salt = "custom-salt"

	merged := MergeConfigs(defaultCfg, customCfg)
	if merged.Runtime.Setup != "complete" {
		t.Fatalf("expected setup=complete for migration path, got %q", merged.Runtime.Setup)
	}
	if merged.Runtime.LastKnownMDNS != defaultCfg.Runtime.LastKnownMDNS {
		t.Fatalf("expected last known mdns to use default when custom is empty")
	}
	if merged.Connectivity.EndpointURL != defaultCfg.Connectivity.EndpointURL || merged.Connectivity.ApiVersion != defaultCfg.Connectivity.ApiVersion {
		t.Fatalf("expected defaults for empty custom scalar fields")
	}
	if merged.AuthSession.PwHash != customCfg.AuthSession.PwHash {
		t.Fatalf("expected custom pwhash to be preserved when provided")
	}
	if merged.Startram.StartramSetReminder != defaultCfg.Startram.StartramSetReminder {
		t.Fatalf("expected startram reminders to merge using OR semantics")
	}
	if merged.AuthSession.Sessions.Authorized == nil || merged.AuthSession.Sessions.Unauthorized == nil {
		t.Fatalf("expected session maps to fall back to defaults when custom is nil")
	}
	if merged.Runtime.CfgDir != defaultCfg.Runtime.CfgDir || merged.Runtime.BinHash != defaultCfg.Runtime.BinHash {
		t.Fatalf("expected default config values when custom values are empty")
	}
	if merged.Runtime.Disable502 != defaultCfg.Runtime.Disable502 {
		t.Fatalf("expected default Disable502 when custom is false")
	}
	if merged.Penpai.PenpaiAllow != true {
		t.Fatalf("expected custom false penpai allow to inherit from default true")
	}
	if merged.Penpai.PenpaiRunning != false {
		t.Fatalf("expected custom false penpai running to remain false")
	}
	if merged.Penpai.PenpaiCores != defaultCfg.Penpai.PenpaiCores {
		t.Fatalf("expected default penpai cores when custom is zero")
	}
	if merged.Penpai.PenpaiActive != defaultCfg.Penpai.PenpaiActive {
		t.Fatalf("expected default penpai active when custom value empty")
	}
	if merged.Connectivity.DiskWarning != nil {
		t.Fatalf("expected custom nil disk warning to be assigned as-is")
	}
	if merged.Penpai.PenpaiModels == nil || len(merged.Penpai.PenpaiModels) != len(defaultCfg.Penpai.PenpaiModels) {
		t.Fatalf("expected latest penpai model list to come from default config")
	}
	if merged.AuthSession.Salt != customCfg.AuthSession.Salt {
		t.Fatalf("expected custom salt to be propagated when default salt is not set and pwhash present")
	}
}

func mergeConfigsCustomOverrideFixture() (structs.SysConfig, structs.SysConfig, structs.SysConfig) {
	defaultCfg := structs.SysConfig{}
	defaultCfg.Runtime.GracefulExit = true
	defaultCfg.Runtime.LastKnownMDNS = "default-mdns"
	defaultCfg.Runtime.Setup = "default-setup"
	defaultCfg.Connectivity.EndpointURL = "https://default.endpoint"
	defaultCfg.Connectivity.ApiVersion = "v2"
	defaultCfg.Connectivity.Piers = []string{"default"}
	defaultCfg.Connectivity.NetCheck = "default-netcheck"
	defaultCfg.Connectivity.UpdateMode = "default-mode"
	defaultCfg.Connectivity.UpdateURL = "https://default-update"
	defaultCfg.Connectivity.UpdateBranch = "default"
	defaultCfg.Runtime.SwapVal = 10
	defaultCfg.Runtime.SwapFile = "/default/swap"
	defaultCfg.AuthSession.KeyFile = "/default/key"
	defaultCfg.Runtime.DockerData = "/default/docker"
	defaultCfg.Connectivity.WgOn = true
	defaultCfg.Connectivity.WgRegistered = true
	defaultCfg.Connectivity.C2CInterval = 1200
	defaultCfg.Runtime.GsVersion = "default-gs"
	defaultCfg.Runtime.CfgDir = "/default/cfg"
	defaultCfg.Runtime.UpdateInterval = 30
	defaultCfg.Runtime.BinHash = "default-bin"
	defaultCfg.Startram.Pubkey = "default-pub"
	defaultCfg.Startram.Privkey = "default-priv"
	defaultCfg.Penpai.PenpaiModels = []structs.Penpai{
		{ModelName: "alpha"},
		{ModelName: "beta"},
	}
	defaultCfg.Penpai.PenpaiActive = "alpha"
	defaultCfg.Runtime.Disable502 = false
	defaultCfg.Runtime.SnapTime = 7
	defaultCfg.Penpai.PenpaiAllow = true
	defaultCfg.Startram.StartramSetReminder = struct {
		One   bool `json:"one"`
		Three bool `json:"three"`
		Seven bool `json:"seven"`
	}{
		One:   true,
		Three: true,
		Seven: true,
	}
	defaultCfg.Connectivity.DiskWarning = map[string]structs.DiskWarning{
		"default": {Eighty: true},
	}
	defaultCfg.AuthSession.Sessions = struct {
		Authorized   map[string]structs.SessionInfo `json:"authorized"`
		Unauthorized map[string]structs.SessionInfo `json:"unauthorized"`
	}{
		Authorized:   map[string]structs.SessionInfo{"default": {Hash: "default-auth"}},
		Unauthorized: map[string]structs.SessionInfo{"default": {Hash: "default-unauth"}},
	}
	defaultCfg.Runtime.LinuxUpdates = struct {
		Value    int    `json:"value"`
		Interval string `json:"interval"`
	}{
		Value:    10,
		Interval: "1m",
	}

	customCfg := structs.SysConfig{}
	customCfg.Runtime.GracefulExit = false
	customCfg.Runtime.LastKnownMDNS = "custom-mdns"
	customCfg.Runtime.Setup = "custom-setup"
	customCfg.AuthSession.PwHash = "present"
	customCfg.Connectivity.EndpointURL = "https://custom.endpoint"
	customCfg.Connectivity.ApiVersion = "v10"
	customCfg.Connectivity.Piers = []string{"a", "b", "c"}
	customCfg.Connectivity.NetCheck = "custom-netcheck"
	customCfg.Connectivity.UpdateMode = "poll"
	customCfg.Connectivity.UpdateURL = "https://custom-update"
	customCfg.Connectivity.UpdateBranch = "stable"
	customCfg.Runtime.SwapVal = 20
	customCfg.Runtime.SwapFile = "/custom/swap"
	customCfg.AuthSession.KeyFile = "/custom/key"
	customCfg.Runtime.DockerData = "/custom/docker"
	customCfg.Connectivity.WgOn = false
	customCfg.Connectivity.WgRegistered = false
	customCfg.AuthSession.Sessions = structs.AuthSessionBag{
		Authorized:   map[string]structs.SessionInfo{"custom": {Hash: "auth"}},
		Unauthorized: map[string]structs.SessionInfo{"custom": {Hash: "unauth"}},
	}
	customCfg.Runtime.LinuxUpdates = struct {
		Value    int    `json:"value"`
		Interval string `json:"interval"`
	}{
		Value:    2,
		Interval: "3m",
	}
	customCfg.Connectivity.C2CInterval = 3000
	customCfg.Runtime.GsVersion = "v10"
	customCfg.Runtime.CfgDir = "/custom/cfg"
	customCfg.Runtime.UpdateInterval = 60
	customCfg.Runtime.BinHash = "custom-bin"
	customCfg.Startram.Pubkey = "custom-pub"
	customCfg.Startram.Privkey = "custom-priv"
	customCfg.Penpai.PenpaiRunning = true
	customCfg.Penpai.PenpaiCores = 4
	customCfg.AuthSession.Salt = "custom-salt"
	customCfg.Penpai.PenpaiAllow = true
	customCfg.Penpai.PenpaiActive = "beta"
	customCfg.Runtime.Disable502 = true
	customCfg.Runtime.SnapTime = 42
	customCfg.Connectivity.RemoteBackupPassword = "rpw"

	merged := MergeConfigs(defaultCfg, customCfg)
	return defaultCfg, customCfg, merged
}

func TestMergeConfigsCustomOverridesScalarAndSliceFields(t *testing.T) {
	_, customCfg, merged := mergeConfigsCustomOverrideFixture()
	if !merged.Runtime.GracefulExit {
		t.Fatal("expected merged graceful exit from custom false or default true to be true")
	}
	if merged.Runtime.LastKnownMDNS != customCfg.Runtime.LastKnownMDNS || merged.Connectivity.EndpointURL != customCfg.Connectivity.EndpointURL {
		t.Fatalf("expected non-empty custom string fields to override defaults")
	}
	if !reflect.DeepEqual(merged.Connectivity.Piers, customCfg.Connectivity.Piers) {
		t.Fatalf("expected custom piers to override defaults")
	}
	if merged.Runtime.SwapVal != customCfg.Runtime.SwapVal || merged.Runtime.SwapFile != customCfg.Runtime.SwapFile {
		t.Fatalf("expected custom swap fields to override defaults")
	}
}

func TestMergeConfigsCustomOverridesSessionAndRuntimeNestedFields(t *testing.T) {
	defaultCfg, customCfg, merged := mergeConfigsCustomOverrideFixture()
	if merged.Startram.StartramSetReminder != defaultCfg.Startram.StartramSetReminder {
		t.Fatal("expected startram reminders to keep default false/true semantics with fallback")
	}
	if !reflect.DeepEqual(merged.AuthSession.Sessions.Authorized, customCfg.AuthSession.Sessions.Authorized) {
		t.Fatalf("expected custom session maps to override defaults")
	}
	if !reflect.DeepEqual(merged.Runtime.LinuxUpdates, customCfg.Runtime.LinuxUpdates) {
		t.Fatalf("expected custom linux update settings to override defaults")
	}
}

func TestMergeConfigsCustomOverridesPenpaiModelSelection(t *testing.T) {
	defaultCfg, customCfg, merged := mergeConfigsCustomOverrideFixture()
	if merged.Penpai.PenpaiCores != customCfg.Penpai.PenpaiCores {
		t.Fatalf("expected non-zero custom penpai cores to override default")
	}
	if merged.Penpai.PenpaiModels[0].ModelName != defaultCfg.Penpai.PenpaiModels[0].ModelName {
		t.Fatalf("expected default penpai models to remain source of truth")
	}
	if merged.Penpai.PenpaiActive != customCfg.Penpai.PenpaiActive {
		t.Fatalf("expected valid custom active penpai model to be selected")
	}
}

func TestMergeConfigsCustomOverridesAdditionalScalarFields(t *testing.T) {
	_, customCfg, merged := mergeConfigsCustomOverrideFixture()
	if merged.AuthSession.PwHash != customCfg.AuthSession.PwHash || merged.Runtime.CfgDir != customCfg.Runtime.CfgDir {
		t.Fatalf("expected custom scalar overrides to propagate")
	}
	if merged.Runtime.Disable502 != customCfg.Runtime.Disable502 || merged.Runtime.SnapTime != customCfg.Runtime.SnapTime {
		t.Fatalf("expected custom booleans and snap time to propagate when set")
	}
	if merged.Connectivity.RemoteBackupPassword != customCfg.Connectivity.RemoteBackupPassword {
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
	cfg := structs.SysConfig{}
	cfg.Runtime.Setup = "counted"
	return cfg
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
	if merged.Runtime.Setup != "counted" {
		t.Fatalf("unexpected merged config: got %q", merged.Runtime.Setup)
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

	defaultConfig := structs.SysConfig{}
	defaultConfig.Runtime.GracefulExit = true
	defaultConfig.Runtime.LastKnownMDNS = "default.mdns"
	defaultConfig.Runtime.Setup = "complete"
	defaultConfig.AuthSession.PwHash = "default-hash"
	defaultConfig.Connectivity.EndpointURL = "https://default"
	defaultConfig.Connectivity.UpdateMode = "poll"
	defaultConfig.Penpai.PenpaiModels = []structs.Penpai{{ModelName: "default-a"}}
	defaultConfig.Penpai.PenpaiActive = "default-a"
	defaultConfig.AuthSession.Sessions.Authorized = map[string]structs.SessionInfo{"default": {Hash: "hash"}}
	defaultConfig.AuthSession.Sessions.Unauthorized = map[string]structs.SessionInfo{"default-unauth": {Hash: "hash"}}

	customConfig := structs.SysConfig{}
	customConfig.Runtime.GracefulExit = false
	customConfig.Connectivity.UpdateMode = ""
	customConfig.AuthSession.PwHash = ""

	merged := MergeConfigs(defaultConfig, customConfig)
	if merged.Runtime.GracefulExit != true {
		t.Fatalf("expected graceful exit to keep default true")
	}
	if merged.Runtime.LastKnownMDNS != defaultConfig.Runtime.LastKnownMDNS {
		t.Fatalf("expected default mdns when custom value missing")
	}
	if merged.Connectivity.EndpointURL != defaultConfig.Connectivity.EndpointURL {
		t.Fatalf("expected default endpoint when custom missing")
	}
	if merged.AuthSession.Salt == "" {
		t.Fatalf("expected generated salt when default config has empty salt and custom pw hash is empty")
	}
}

func TestMergeConfigsAppliesDefaultsAndCustomOverrides(t *testing.T) {
	defaultConfig := structs.SysConfig{}
	defaultConfig.Runtime.GracefulExit = false
	defaultConfig.Runtime.LastKnownMDNS = "default-mdns"
	defaultConfig.Runtime.Setup = "migrated"
	defaultConfig.AuthSession.PwHash = "default-hash"
	defaultConfig.Connectivity.EndpointURL = "https://default-endpoint"
	defaultConfig.Connectivity.ApiVersion = "1.0"
	defaultConfig.Connectivity.Piers = []string{"default-pier"}
	defaultConfig.Connectivity.NetCheck = "default-netcheck"
	defaultConfig.Connectivity.UpdateMode = "default"
	defaultConfig.Connectivity.UpdateURL = "https://updates.example.com"
	defaultConfig.Connectivity.UpdateBranch = "main"
	defaultConfig.Runtime.SwapVal = 1
	defaultConfig.Runtime.SwapFile = "/tmp/default-swap"
	defaultConfig.AuthSession.KeyFile = "/tmp/default-key"
	defaultConfig.Runtime.DockerData = "default-docker"
	defaultConfig.Connectivity.WgOn = false
	defaultConfig.Connectivity.WgRegistered = false
	defaultConfig.Connectivity.DiskWarning = map[string]structs.DiskWarning{"default": {Eighty: true}}
	defaultConfig.Connectivity.C2CInterval = 1000
	defaultConfig.Runtime.GsVersion = "v1"
	defaultConfig.Runtime.CfgDir = "/default/cfg"
	defaultConfig.Runtime.UpdateInterval = 20
	defaultConfig.Runtime.BinHash = "default-bin"
	defaultConfig.Startram.Pubkey = "default-pub"
	defaultConfig.Startram.Privkey = "default-priv"
	defaultConfig.AuthSession.Salt = "default-salt"
	defaultConfig.Penpai.PenpaiAllow = true
	defaultConfig.Penpai.PenpaiRunning = false
	defaultConfig.Penpai.PenpaiCores = 4
	defaultConfig.Penpai.PenpaiModels = []structs.Penpai{{ModelName: "mymodel"}}
	defaultConfig.Penpai.PenpaiActive = "mymodel"
	defaultConfig.Startram.DisableSlsa = true
	defaultConfig.Runtime.Disable502 = true
	defaultConfig.Runtime.SnapTime = 12
	defaultConfig.AuthSession.Sessions.Authorized = map[string]structs.SessionInfo{"user": {Hash: "old"}}
	defaultConfig.AuthSession.Sessions.Unauthorized = map[string]structs.SessionInfo{"anon": {Hash: "anon-old"}}
	defaultConfig.Runtime.LinuxUpdates = struct {
		Value    int    `json:"value"`
		Interval string `json:"interval"`
	}{
		Value:    1,
		Interval: "2m",
	}
	defaultConfig.Startram.StartramSetReminder = struct {
		One   bool `json:"one"`
		Three bool `json:"three"`
		Seven bool `json:"seven"`
	}{
		One:   false,
		Three: false,
		Seven: false,
	}

	customConfig := structs.SysConfig{}
	customConfig.Runtime.GracefulExit = true
	customConfig.Runtime.LastKnownMDNS = "" // force default
	customConfig.Connectivity.EndpointURL = "https://custom-endpoint"
	customConfig.AuthSession.PwHash = "" // triggers new install logic
	customConfig.Runtime.SwapVal = 0     // fallback to default
	customConfig.Connectivity.WgOn = true
	customConfig.Connectivity.WgRegistered = true
	customConfig.Connectivity.DiskWarning = map[string]structs.DiskWarning{"custom": {}}
	customConfig.Penpai.PenpaiAllow = true
	customConfig.Penpai.PenpaiRunning = true
	customConfig.Penpai.PenpaiCores = 0          // fallback
	customConfig.Penpai.PenpaiModels = nil       // ignored
	customConfig.Penpai.PenpaiActive = "invalid" // should fallback
	customConfig.Runtime.Disable502 = false      // default should stay true
	customConfig.Startram.StartramSetReminder = struct {
		One   bool `json:"one"`
		Three bool `json:"three"`
		Seven bool `json:"seven"`
	}{
		One:   true,
		Three: true,
		Seven: false,
	}
	customConfig.AuthSession.Sessions.Unauthorized = map[string]structs.SessionInfo{"new": {Hash: "new"}}

	merged := mergeConfigs(defaultConfig, customConfig)
	if merged.Runtime.GracefulExit != true {
		t.Fatalf("expected merged graceful exit true")
	}
	if merged.Runtime.LastKnownMDNS != defaultConfig.Runtime.LastKnownMDNS {
		t.Fatalf("expected default mdns on empty custom value")
	}
	if merged.Connectivity.EndpointURL != customConfig.Connectivity.EndpointURL {
		t.Fatalf("expected custom endpoint to win")
	}
	if merged.Connectivity.ApiVersion != defaultConfig.Connectivity.ApiVersion {
		t.Fatalf("expected default API version when custom empty")
	}
	if merged.Connectivity.Piers[0] != defaultConfig.Connectivity.Piers[0] {
		t.Fatalf("expected default piers for empty custom list")
	}
	if merged.Connectivity.NetCheck != defaultConfig.Connectivity.NetCheck {
		t.Fatalf("expected default netcheck when custom empty")
	}
	if !merged.Connectivity.WgOn || !merged.Connectivity.WgRegistered {
		t.Fatalf("expected bool-or flags to include custom true values")
	}
	if merged.Startram.StartramSetReminder.One != true || !merged.Startram.StartramSetReminder.Three || merged.Startram.StartramSetReminder.Seven {
		t.Fatalf("unexpected startram reminder merge state")
	}
	if merged.Penpai.PenpaiCores != defaultConfig.Penpai.PenpaiCores {
		t.Fatalf("expected default penpai cores when custom is zero")
	}
	if merged.Penpai.PenpaiActive != defaultConfig.Penpai.PenpaiActive {
		t.Fatalf("expected default penpai active when custom active is invalid")
	}
	if merged.Runtime.Disable502 != true {
		t.Fatalf("expected default disable502 when custom is false")
	}
	if merged.Runtime.Setup != "start" {
		t.Fatalf("expected setup=start when custom pw hash is missing")
	}
	if merged.AuthSession.PwHash != defaultConfig.AuthSession.PwHash {
		t.Fatalf("expected default pwhash when custom missing")
	}
	if merged.AuthSession.Salt == "" {
		t.Fatalf("expected generated salt when starting setup")
	}
	if !merged.Penpai.PenpaiAllow {
		t.Fatalf("expected penpai allow true due custom override")
	}
	if !merged.Penpai.PenpaiRunning {
		t.Fatalf("expected penpai running from custom")
	}
	if len(merged.Penpai.PenpaiModels) != len(defaultConfig.Penpai.PenpaiModels) || merged.Penpai.PenpaiModels[0].ModelName != defaultConfig.Penpai.PenpaiModels[0].ModelName {
		t.Fatalf("expected default penpai models")
	}
	if !reflect.DeepEqual(merged.AuthSession.Sessions.Unauthorized, customConfig.AuthSession.Sessions.Unauthorized) {
		t.Fatalf("expected custom unauthorized sessions")
	}
	if !reflect.DeepEqual(merged.AuthSession.Sessions.Authorized, defaultConfig.AuthSession.Sessions.Authorized) {
		t.Fatalf("expected default authorized sessions")
	}
	if !reflect.DeepEqual(merged.Connectivity.DiskWarning, customConfig.Connectivity.DiskWarning) {
		t.Fatalf("expected custom disk warning")
	}
	if merged.Connectivity.C2CInterval != defaultConfig.Connectivity.C2CInterval || merged.Runtime.GsVersion != defaultConfig.Runtime.GsVersion || merged.Runtime.CfgDir != defaultConfig.Runtime.CfgDir {
		t.Fatalf("expected default scalar fields when custom zero/empty")
	}
}

func TestRuntimeContextSettersSerializeUpdates(t *testing.T) {
	const iterations = 200
	var wg sync.WaitGroup
	wg.Add(iterations * 2)

	originalBasePath := BasePath()
	originalArchitecture := Architecture()
	originalDebugMode := DebugMode()
	t.Cleanup(func() {
		SetBasePath(originalBasePath)
		SetArchitecture(originalArchitecture)
		SetDebugMode(originalDebugMode)
	})

	for i := 0; i < iterations; i++ {
		go func(i int) {
			defer wg.Done()
			SetBasePath(filepath.Join("/tmp", "base", t.Name(), "a", string(rune('a'+(i%26)))))
		}(i)
		go func(i int) {
			defer wg.Done()
			SetArchitecture(filepath.Join("amd", "arch", string(rune('a'+(i%26)))))
			SetDebugMode(i%2 == 0)
		}(i)
	}
	wg.Wait()

	ctx := RuntimeContextSnapshot()
	if ctx.BasePath == "" || ctx.Architecture == "" {
		t.Fatal("runtime context fields should remain set after concurrent updates")
	}
}
