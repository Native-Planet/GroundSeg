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
	defaultCfg := structs.SysConfig{}
	defaultCfg.Setup = "complete"
	defaultCfg.EndpointUrl = "default.endpoint"
	customCfg := structs.SysConfig{}
	customCfg.PwHash = ""

	merged := mergeConfigs(defaultCfg, customCfg)
	if merged.Setup != "start" {
		t.Fatalf("expected setup=start for new install, got %q", merged.Setup)
	}
	if merged.Salt == "" {
		t.Fatal("expected salt to be generated for new install")
	}
}

func TestMergeConfigsPrefersCustomEndpoint(t *testing.T) {
	defaultCfg := structs.SysConfig{}
	defaultCfg.EndpointUrl = "default.endpoint"
	customCfg := structs.SysConfig{}
	customCfg.EndpointUrl = "custom.endpoint"
	customCfg.PwHash = "hash"

	merged := mergeConfigs(defaultCfg, customCfg)
	if merged.EndpointUrl != "custom.endpoint" {
		t.Fatalf("expected custom endpoint to win, got %q", merged.EndpointUrl)
	}
}

func TestMergeConfigsMigrationUsesDefaultsAndGeneratedSaltWhenMissing(t *testing.T) {
	defaultCfg := structs.SysConfig{}
	defaultCfg.GracefulExit = true
	defaultCfg.Setup = "complete"
	defaultCfg.LastKnownMDNS = "default-mdns"
	defaultCfg.EndpointUrl = "https://default.endpoint"
	defaultCfg.ApiVersion = "v2"
	defaultCfg.NetCheck = "default-netcheck"
	defaultCfg.UpdateMode = "default-mode"
	defaultCfg.UpdateUrl = "https://default-update"
	defaultCfg.UpdateBranch = "main"
	defaultCfg.SwapVal = 99
	defaultCfg.SwapFile = "/default/swap"
	defaultCfg.KeyFile = "/default/key"
	defaultCfg.DockerData = "/default/docker-data"
	defaultCfg.WgOn = true
	defaultCfg.WgRegistered = true
	defaultCfg.UpdateInterval = 15
	defaultCfg.C2cInterval = 2000
	defaultCfg.GsVersion = "v2.0"
	defaultCfg.CfgDir = "/default/cfg"
	defaultCfg.BinHash = "bin-default"
	defaultCfg.Pubkey = "default-pub"
	defaultCfg.Privkey = "default-priv"
	defaultCfg.PwHash = "old-hash"
	defaultCfg.DiskWarning = map[string]structs.DiskWarning{
		"data": {Eighty: true},
	}
	defaultCfg.PenpaiModels = []structs.Penpai{
		{ModelName: "default-a"},
		{ModelName: "default-b"},
	}
	defaultCfg.PenpaiActive = "default-a"
	defaultCfg.Disable502 = true
	defaultCfg.SnapTime = 14
	defaultCfg.PenpaiAllow = true
	defaultCfg.PenpaiRunning = true
	defaultCfg.PenpaiCores = 8
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
	customCfg := structs.SysConfig{}
	customCfg.PwHash = "legacy-pwhash"
	customCfg.Salt = "custom-salt"

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
	defaultCfg := structs.SysConfig{}
	defaultCfg.GracefulExit = true
	defaultCfg.LastKnownMDNS = "default-mdns"
	defaultCfg.Setup = "default-setup"
	defaultCfg.EndpointUrl = "https://default.endpoint"
	defaultCfg.ApiVersion = "v2"
	defaultCfg.Piers = []string{"default"}
	defaultCfg.NetCheck = "default-netcheck"
	defaultCfg.UpdateMode = "default-mode"
	defaultCfg.UpdateUrl = "https://default-update"
	defaultCfg.UpdateBranch = "default"
	defaultCfg.SwapVal = 10
	defaultCfg.SwapFile = "/default/swap"
	defaultCfg.KeyFile = "/default/key"
	defaultCfg.DockerData = "/default/docker"
	defaultCfg.WgOn = true
	defaultCfg.WgRegistered = true
	defaultCfg.C2cInterval = 1200
	defaultCfg.GsVersion = "default-gs"
	defaultCfg.CfgDir = "/default/cfg"
	defaultCfg.UpdateInterval = 30
	defaultCfg.BinHash = "default-bin"
	defaultCfg.Pubkey = "default-pub"
	defaultCfg.Privkey = "default-priv"
	defaultCfg.PenpaiModels = []structs.Penpai{
		{ModelName: "alpha"},
		{ModelName: "beta"},
	}
	defaultCfg.PenpaiActive = "alpha"
	defaultCfg.Disable502 = false
	defaultCfg.SnapTime = 7
	defaultCfg.PenpaiAllow = true
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

	customCfg := structs.SysConfig{}
	customCfg.GracefulExit = false
	customCfg.LastKnownMDNS = "custom-mdns"
	customCfg.Setup = "custom-setup"
	customCfg.PwHash = "present"
	customCfg.EndpointUrl = "https://custom.endpoint"
	customCfg.ApiVersion = "v10"
	customCfg.Piers = []string{"a", "b", "c"}
	customCfg.NetCheck = "custom-netcheck"
	customCfg.UpdateMode = "poll"
	customCfg.UpdateUrl = "https://custom-update"
	customCfg.UpdateBranch = "stable"
	customCfg.SwapVal = 20
	customCfg.SwapFile = "/custom/swap"
	customCfg.KeyFile = "/custom/key"
	customCfg.DockerData = "/custom/docker"
	customCfg.WgOn = false
	customCfg.WgRegistered = false
	customCfg.Sessions = structs.AuthSessionBag{
		Authorized:   map[string]structs.SessionInfo{"custom": {Hash: "auth"}},
		Unauthorized: map[string]structs.SessionInfo{"custom": {Hash: "unauth"}},
	}
	customCfg.LinuxUpdates = struct {
		Value    int    `json:"value"`
		Interval string `json:"interval"`
	}{
		Value:    2,
		Interval: "3m",
	}
	customCfg.C2cInterval = 3000
	customCfg.GsVersion = "v10"
	customCfg.CfgDir = "/custom/cfg"
	customCfg.UpdateInterval = 60
	customCfg.BinHash = "custom-bin"
	customCfg.Pubkey = "custom-pub"
	customCfg.Privkey = "custom-priv"
	customCfg.PenpaiRunning = true
	customCfg.PenpaiCores = 4
	customCfg.Salt = "custom-salt"
	customCfg.PenpaiAllow = true
	customCfg.PenpaiActive = "beta"
	customCfg.Disable502 = true
	customCfg.SnapTime = 42
	customCfg.RemoteBackupPassword = "rpw"

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
	cfg := structs.SysConfig{}
	cfg.Setup = "counted"
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

	defaultConfig := structs.SysConfig{}
	defaultConfig.GracefulExit = true
	defaultConfig.LastKnownMDNS = "default.mdns"
	defaultConfig.Setup = "complete"
	defaultConfig.PwHash = "default-hash"
	defaultConfig.EndpointUrl = "https://default"
	defaultConfig.UpdateMode = "poll"
	defaultConfig.PenpaiModels = []structs.Penpai{{ModelName: "default-a"}}
	defaultConfig.PenpaiActive = "default-a"
	defaultConfig.Sessions.Authorized = map[string]structs.SessionInfo{"default": {Hash: "hash"}}
	defaultConfig.Sessions.Unauthorized = map[string]structs.SessionInfo{"default-unauth": {Hash: "hash"}}

	customConfig := structs.SysConfig{}
	customConfig.GracefulExit = false
	customConfig.UpdateMode = ""
	customConfig.PwHash = ""

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
	defaultConfig := structs.SysConfig{}
	defaultConfig.GracefulExit = false
	defaultConfig.LastKnownMDNS = "default-mdns"
	defaultConfig.Setup = "migrated"
	defaultConfig.PwHash = "default-hash"
	defaultConfig.EndpointUrl = "https://default-endpoint"
	defaultConfig.ApiVersion = "1.0"
	defaultConfig.Piers = []string{"default-pier"}
	defaultConfig.NetCheck = "default-netcheck"
	defaultConfig.UpdateMode = "default"
	defaultConfig.UpdateUrl = "https://updates.example.com"
	defaultConfig.UpdateBranch = "main"
	defaultConfig.SwapVal = 1
	defaultConfig.SwapFile = "/tmp/default-swap"
	defaultConfig.KeyFile = "/tmp/default-key"
	defaultConfig.DockerData = "default-docker"
	defaultConfig.WgOn = false
	defaultConfig.WgRegistered = false
	defaultConfig.DiskWarning = map[string]structs.DiskWarning{"default": {Eighty: true}}
	defaultConfig.C2cInterval = 1000
	defaultConfig.GsVersion = "v1"
	defaultConfig.CfgDir = "/default/cfg"
	defaultConfig.UpdateInterval = 20
	defaultConfig.BinHash = "default-bin"
	defaultConfig.Pubkey = "default-pub"
	defaultConfig.Privkey = "default-priv"
	defaultConfig.Salt = "default-salt"
	defaultConfig.PenpaiAllow = true
	defaultConfig.PenpaiRunning = false
	defaultConfig.PenpaiCores = 4
	defaultConfig.PenpaiModels = []structs.Penpai{{ModelName: "mymodel"}}
	defaultConfig.PenpaiActive = "mymodel"
	defaultConfig.DisableSlsa = true
	defaultConfig.Disable502 = true
	defaultConfig.SnapTime = 12
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

	customConfig := structs.SysConfig{}
	customConfig.GracefulExit = true
	customConfig.LastKnownMDNS = "" // force default
	customConfig.EndpointUrl = "https://custom-endpoint"
	customConfig.PwHash = "" // triggers new install logic
	customConfig.SwapVal = 0 // fallback to default
	customConfig.WgOn = true
	customConfig.WgRegistered = true
	customConfig.DiskWarning = map[string]structs.DiskWarning{"custom": {}}
	customConfig.PenpaiAllow = true
	customConfig.PenpaiRunning = true
	customConfig.PenpaiCores = 0          // fallback
	customConfig.PenpaiModels = nil       // ignored
	customConfig.PenpaiActive = "invalid" // should fallback
	customConfig.Disable502 = false       // default should stay true
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
