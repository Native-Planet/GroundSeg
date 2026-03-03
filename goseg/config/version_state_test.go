package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"groundseg/structs"
)

func preserveVersionGlobals(t *testing.T) {
	t.Helper()
	preserveVersionFetchRuntimeState(t)
	originalConfig := globalConfig
	originalBasePath := BasePath()
	t.Cleanup(func() {
		globalConfig = originalConfig
		SetBasePath(originalBasePath)
	})
}

func TestInMemoryVersionStoreSetters(t *testing.T) {
	store := newInMemoryVersionStore()
	first := structs.Channel{Groundseg: structs.VersionDetails{Repo: "repo-a"}}
	second := structs.Channel{Groundseg: structs.VersionDetails{Repo: "repo-b"}}

	store.SetState(first, true)
	snap := store.Snapshot()
	if !snap.ServerReady || snap.Channel.Groundseg.Repo != "repo-a" {
		t.Fatalf("unexpected snapshot after SetState: %+v", snap)
	}

	store.SetChannel(second)
	store.SetServerReady(false)
	snap = store.Snapshot()
	if snap.ServerReady || snap.Channel.Groundseg.Repo != "repo-b" {
		t.Fatalf("unexpected snapshot after SetChannel/SetServerReady: %+v", snap)
	}
}

func TestResolveLatestChannelAndPublishVersionMetadata(t *testing.T) {
	preserveVersionGlobals(t)

	setVersionFetchPolicy(1, time.Millisecond)
	setVersionFetchSleep(func(time.Duration) {})
	globalConfig.UpdateUrl = "https://updates.example/version"
	setVersionHTTPClient(&stubVersionHTTPClient{
		results: []stubVersionHTTPResult{
			{resp: newHTTPResponse(200, `{"groundseg":{"beta":{"groundseg":{"repo":"repo-beta"}}}}`)},
		},
	})

	conf := structs.SysConfig{}
	conf.GsVersion = "1.0.0"
	conf.UpdateBranch = "beta"
	metadata, channel, err := ResolveLatestChannel(conf)
	if err != nil {
		t.Fatalf("ResolveLatestChannel failed: %v", err)
	}
	if channel.Groundseg.Repo != "repo-beta" {
		t.Fatalf("unexpected channel: %+v", channel)
	}

	SetBasePath(t.TempDir())
	if err := os.MkdirAll(filepath.Join(BasePath(), "settings"), 0o755); err != nil {
		t.Fatalf("mkdir settings failed: %v", err)
	}
	setVersionStore(newInMemoryVersionStore())
	if err := PublishVersionMetadata(metadata, channel); err != nil {
		t.Fatalf("PublishVersionMetadata failed: %v", err)
	}
	if !IsVersionServerReady() {
		t.Fatalf("expected version server ready after publish")
	}
	if GetVersionChannel().Groundseg.Repo != "repo-beta" {
		t.Fatalf("unexpected published channel: %+v", GetVersionChannel())
	}
	if _, err := os.Stat(filepath.Join(BasePath(), "settings", "version_info.json")); err != nil {
		t.Fatalf("expected persisted version_info.json: %v", err)
	}
}

func TestCheckVersionReturnsStoredChannelOnFailure(t *testing.T) {
	preserveVersionGlobals(t)

	existing := structs.Channel{Groundseg: structs.VersionDetails{Repo: "existing-repo"}}
	store := newInMemoryVersionStore()
	store.SetState(existing, true)
	setVersionStore(store)

	globalConfig.UpdateUrl = "https://updates.example/version"
	globalConfig.GsVersion = "1.0.0"
	globalConfig.UpdateBranch = "latest"
	setVersionFetchPolicy(1, time.Millisecond)
	setVersionFetchSleep(func(time.Duration) {})
	setVersionHTTPClient(&stubVersionHTTPClient{
		results: []stubVersionHTTPResult{{err: os.ErrDeadlineExceeded}},
	})

	got, ok := CheckVersion()
	if ok {
		t.Fatalf("expected CheckVersion to report failure")
	}
	if got.Groundseg.Repo != "existing-repo" {
		t.Fatalf("expected existing channel fallback, got %+v", got)
	}
}

func TestCheckVersionWithErrorReturnsCauseOnFailure(t *testing.T) {
	preserveVersionGlobals(t)

	existing := structs.Channel{Groundseg: structs.VersionDetails{Repo: "existing-repo"}}
	store := newInMemoryVersionStore()
	store.SetState(existing, true)
	setVersionStore(store)

	globalConfig.UpdateUrl = "https://updates.example/version"
	globalConfig.GsVersion = "1.0.0"
	globalConfig.UpdateBranch = "latest"
	setVersionFetchPolicy(1, time.Millisecond)
	setVersionFetchSleep(func(time.Duration) {})
	setVersionHTTPClient(&stubVersionHTTPClient{
		results: []stubVersionHTTPResult{{err: os.ErrDeadlineExceeded}},
	})

	got, err := CheckVersionWithError()
	if err == nil {
		t.Fatalf("expected CheckVersionWithError to fail")
	}
	if got.Groundseg.Repo != "existing-repo" {
		t.Fatalf("expected fallback channel on failure, got %+v", got)
	}
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("expected wrapped cause, got %v", err)
	}
}

func TestSyncVersionInfoSuccessThenMissingChannelFailure(t *testing.T) {
	preserveVersionGlobals(t)

	SetBasePath(t.TempDir())
	if err := os.MkdirAll(filepath.Join(BasePath(), "settings"), 0o755); err != nil {
		t.Fatalf("mkdir settings failed: %v", err)
	}
	setVersionStore(newInMemoryVersionStore())
	setVersionFetchPolicy(1, time.Millisecond)
	setVersionFetchSleep(func(time.Duration) {})
	globalConfig.UpdateUrl = "https://updates.example/version"
	globalConfig.GsVersion = "1.0.0"
	globalConfig.UpdateBranch = "latest"

	setVersionHTTPClient(&stubVersionHTTPClient{
		results: []stubVersionHTTPResult{
			{resp: newHTTPResponse(200, `{"groundseg":{"latest":{"groundseg":{"repo":"repo-latest"}}}}`)},
		},
	})
	channel, ok := SyncVersionInfo()
	if !ok || channel.Groundseg.Repo != "repo-latest" {
		t.Fatalf("expected SyncVersionInfo success, got channel=%+v ok=%v", channel, ok)
	}
	if !IsVersionServerReady() {
		t.Fatalf("expected server ready after successful sync")
	}

	globalConfig.UpdateBranch = "missing"
	setVersionHTTPClient(&stubVersionHTTPClient{
		results: []stubVersionHTTPResult{
			{resp: newHTTPResponse(200, `{"groundseg":{"latest":{"groundseg":{"repo":"repo-latest"}}}}`)},
		},
	})
	channel, ok = SyncVersionInfo()
	if ok {
		t.Fatalf("expected SyncVersionInfo failure for missing channel")
	}
	if channel.Groundseg.Repo != "repo-latest" {
		t.Fatalf("expected previous channel fallback, got %+v", channel)
	}
	if IsVersionServerReady() {
		t.Fatalf("expected server ready false after failure")
	}
}

func TestSyncVersionInfoWithErrorReturnsCauseOnFailure(t *testing.T) {
	preserveVersionGlobals(t)

	SetBasePath(t.TempDir())
	if err := os.MkdirAll(filepath.Join(BasePath(), "settings"), 0o755); err != nil {
		t.Fatalf("mkdir settings failed: %v", err)
	}
	setVersionStore(newInMemoryVersionStore())
	setVersionFetchPolicy(1, time.Millisecond)
	setVersionFetchSleep(func(time.Duration) {})
	globalConfig.UpdateUrl = "https://updates.example/version"
	globalConfig.GsVersion = "1.0.0"
	globalConfig.UpdateBranch = "latest"

	setVersionHTTPClient(&stubVersionHTTPClient{
		results: []stubVersionHTTPResult{
			{resp: newHTTPResponse(200, `{"groundseg":{"latest":{"groundseg":{"repo":"repo-latest"}}}}`)},
		},
	})
	channel, err := SyncVersionInfoWithError()
	if err != nil {
		t.Fatalf("expected SyncVersionInfoWithError success, got %v", err)
	}
	if channel.Groundseg.Repo != "repo-latest" {
		t.Fatalf("expected initial channel, got %+v", channel)
	}

	globalConfig.UpdateBranch = "missing"
	setVersionHTTPClient(&stubVersionHTTPClient{
		results: []stubVersionHTTPResult{
			{resp: newHTTPResponse(200, `{"groundseg":{"latest":{"groundseg":{"repo":"repo-latest"}}}}`)},
		},
	})
	_, err = SyncVersionInfoWithError()
	if err == nil {
		t.Fatalf("expected SyncVersionInfoWithError failure")
	}
	if !strings.Contains(err.Error(), "resolve latest version channel") {
		t.Fatalf("expected wrapped resolution error, got %v", err)
	}
}

func TestCreateDefaultVersionAndLocalVersionFallback(t *testing.T) {
	preserveVersionGlobals(t)

	SetBasePath(t.TempDir())
	if err := os.MkdirAll(filepath.Join(BasePath(), "settings"), 0o755); err != nil {
		t.Fatalf("mkdir settings failed: %v", err)
	}

	if err := CreateDefaultVersion(); err != nil {
		t.Fatalf("CreateDefaultVersion failed: %v", err)
	}
	loaded := LocalVersion()
	if len(loaded.Groundseg) == 0 {
		t.Fatalf("expected non-empty default local version metadata")
	}

	if err := os.WriteFile(filepath.Join(BasePath(), "settings", "version_info.json"), []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("write invalid version file failed: %v", err)
	}
	fallback := LocalVersion()
	if len(fallback.Groundseg) == 0 {
		t.Fatalf("expected fallback metadata for invalid file")
	}
}
