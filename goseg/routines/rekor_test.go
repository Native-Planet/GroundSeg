package routines

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func resetRekorSeamsForTest(t *testing.T) {
	t.Helper()
	origFetchTimestamp := fetchTimestampForRekor
	origFetchSnapshot := fetchSnapshotForRekor
	origFetchTargets := fetchTargetsForRekor
	origDownload := downloadMetadataForRekor
	origReadCached := readCachedKeyForRekor
	origHome := userHomeDirForRekor
	origMkdir := mkdirAllForRekor
	origWrite := writeFileForRekor
	origTimeUntil := timeUntilForRekor

	t.Cleanup(func() {
		fetchTimestampForRekor = origFetchTimestamp
		fetchSnapshotForRekor = origFetchSnapshot
		fetchTargetsForRekor = origFetchTargets
		downloadMetadataForRekor = origDownload
		readCachedKeyForRekor = origReadCached
		userHomeDirForRekor = origHome
		mkdirAllForRekor = origMkdir
		writeFileForRekor = origWrite
		timeUntilForRekor = origTimeUntil
	})
}

func makeTargetsWithStatus(status, hash string, expires time.Time) *targets {
	raw := fmt.Sprintf(`{"signed":{"expires":%q,"targets":{"rekor.pub":{"hashes":{"sha256":%q},"custom":{"sigstore":{"status":%q}}}}}}`, expires.Format(time.RFC3339), hash, status)
	var targ targets
	_ = json.Unmarshal([]byte(raw), &targ)
	return &targ
}

func TestFetchTimestampParsesMetadata(t *testing.T) {
	resetRekorSeamsForTest(t)
	downloadMetadataForRekor = func(url string) ([]byte, error) {
		if !strings.HasSuffix(url, "/timestamp.json") {
			t.Fatalf("unexpected timestamp url: %s", url)
		}
		return []byte(`{"signed":{"meta":{"snapshot.json":{"version":7}},"expires":"2030-01-01T00:00:00Z"}}`), nil
	}

	ts, err := fetchTimestamp()
	if err != nil {
		t.Fatalf("fetchTimestamp returned error: %v", err)
	}
	if got := ts.Signed.Meta["snapshot.json"].Version; got != 7 {
		t.Fatalf("unexpected snapshot version: got %d want 7", got)
	}
}

func TestFetchSnapshotAndTargetsParseMetadata(t *testing.T) {
	resetRekorSeamsForTest(t)
	downloadMetadataForRekor = func(url string) ([]byte, error) {
		switch {
		case strings.HasSuffix(url, "/5.snapshot.json"):
			return []byte(`{"signed":{"meta":{"targets.json":{"version":9,"length":123}},"expires":"2030-01-01T00:00:00Z"}}`), nil
		case strings.HasSuffix(url, "/9.targets.json"):
			return []byte(`{"signed":{"expires":"2030-01-01T00:00:00Z","targets":{"rekor.pub":{"hashes":{"sha256":"abc"},"custom":{"sigstore":{"status":"Active"}}}}}}`), nil
		default:
			return nil, fmt.Errorf("unexpected url: %s", url)
		}
	}

	snap, err := fetchSnapshot(5)
	if err != nil {
		t.Fatalf("fetchSnapshot returned error: %v", err)
	}
	if snap.Signed.Meta["targets.json"].Version != 9 {
		t.Fatalf("unexpected targets version: %+v", snap.Signed.Meta["targets.json"])
	}

	targ, err := fetchTargets(9)
	if err != nil {
		t.Fatalf("fetchTargets returned error: %v", err)
	}
	if targ.Signed.Targets["rekor.pub"].Custom.Sigstore.Status != "Active" {
		t.Fatalf("unexpected key status: %+v", targ.Signed.Targets["rekor.pub"].Custom.Sigstore)
	}
}

func TestReadCachedKeyFreshAndStale(t *testing.T) {
	file := filepath.Join(t.TempDir(), "rekor.pub")
	if err := os.WriteFile(file, []byte("fresh"), 0o644); err != nil {
		t.Fatalf("failed to write cache file: %v", err)
	}

	data, err := readCachedKey(file)
	if err != nil {
		t.Fatalf("readCachedKey returned error for fresh file: %v", err)
	}
	if string(data) != "fresh" {
		t.Fatalf("unexpected fresh cache data: %q", data)
	}

	old := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(file, old, old); err != nil {
		t.Fatalf("failed to age cache file: %v", err)
	}
	if _, err := readCachedKey(file); err == nil {
		t.Fatal("expected stale cache error")
	}
}

func TestRekorKeyUsesFreshCachedKeyWhenHashMatches(t *testing.T) {
	resetRekorSeamsForTest(t)

	home := t.TempDir()
	keyData := []byte("cached-key")
	hash := sha256.Sum256(keyData)
	hashHex := hex.EncodeToString(hash[:])
	fullKeyPath := filepath.Join(home, keyPath)
	if err := os.MkdirAll(filepath.Dir(fullKeyPath), 0o755); err != nil {
		t.Fatalf("failed to create key dir: %v", err)
	}
	if err := os.WriteFile(fullKeyPath, keyData, 0o644); err != nil {
		t.Fatalf("failed to write cached key: %v", err)
	}

	userHomeDirForRekor = func() (string, error) { return home, nil }
	fetchTimestampForRekor = func() (*timestamp, error) {
		ts := &timestamp{}
		ts.Signed.Meta = map[string]struct {
			Version int `json:"version"`
		}{"snapshot.json": {Version: 1}}
		return ts, nil
	}
	fetchSnapshotForRekor = func(version int) (*snapshot, error) {
		if version != 1 {
			t.Fatalf("unexpected snapshot version: %d", version)
		}
		snap := &snapshot{}
		snap.Signed.Meta = map[string]struct {
			Version int   `json:"version"`
			Length  int64 `json:"length"`
		}{"targets.json": {Version: 2}}
		return snap, nil
	}
	fetchTargetsForRekor = func(version int) (*targets, error) {
		if version != 2 {
			t.Fatalf("unexpected targets version: %d", version)
		}
		return makeTargetsWithStatus("Active", hashHex, time.Now().Add(time.Hour)), nil
	}
	downloadCalled := false
	downloadMetadataForRekor = func(string) ([]byte, error) {
		downloadCalled = true
		return []byte("unexpected"), nil
	}

	path, err := rekorKey()
	if err != nil {
		t.Fatalf("rekorKey returned error: %v", err)
	}
	if path != fullKeyPath {
		t.Fatalf("unexpected key path: got %q want %q", path, fullKeyPath)
	}
	if downloadCalled {
		t.Fatal("expected cache hit to avoid key download")
	}
}

func TestRekorKeyDownloadsAndCachesWhenNeeded(t *testing.T) {
	resetRekorSeamsForTest(t)

	home := t.TempDir()
	keyData := []byte("downloaded-key")
	h := sha256.Sum256(keyData)
	hashHex := hex.EncodeToString(h[:])
	fullKeyPath := filepath.Join(home, keyPath)

	userHomeDirForRekor = func() (string, error) { return home, nil }
	readCachedKeyForRekor = func(string) ([]byte, error) { return nil, errors.New("cache miss") }
	fetchTimestampForRekor = func() (*timestamp, error) {
		ts := &timestamp{}
		ts.Signed.Meta = map[string]struct {
			Version int `json:"version"`
		}{"snapshot.json": {Version: 1}}
		return ts, nil
	}
	fetchSnapshotForRekor = func(int) (*snapshot, error) {
		snap := &snapshot{}
		snap.Signed.Meta = map[string]struct {
			Version int   `json:"version"`
			Length  int64 `json:"length"`
		}{"targets.json": {Version: 2}}
		return snap, nil
	}
	fetchTargetsForRekor = func(int) (*targets, error) {
		return makeTargetsWithStatus("Active", hashHex, time.Now().Add(time.Hour)), nil
	}
	downloadMetadataForRekor = func(url string) ([]byte, error) {
		if !strings.Contains(url, "/api/v1/log/publicKey") {
			t.Fatalf("unexpected download url: %s", url)
		}
		return keyData, nil
	}

	path, err := rekorKey()
	if err != nil {
		t.Fatalf("rekorKey returned error: %v", err)
	}
	if path != fullKeyPath {
		t.Fatalf("unexpected key path: got %q want %q", path, fullKeyPath)
	}
	cached, err := os.ReadFile(fullKeyPath)
	if err != nil {
		t.Fatalf("failed to read cached key: %v", err)
	}
	if string(cached) != string(keyData) {
		t.Fatalf("unexpected cached key data: got %q want %q", cached, keyData)
	}
}

func TestRekorKeyFailsForInactiveTargetOrHashMismatch(t *testing.T) {
	resetRekorSeamsForTest(t)

	home := t.TempDir()
	userHomeDirForRekor = func() (string, error) { return home, nil }
	fetchTimestampForRekor = func() (*timestamp, error) {
		ts := &timestamp{}
		ts.Signed.Meta = map[string]struct {
			Version int `json:"version"`
		}{"snapshot.json": {Version: 1}}
		return ts, nil
	}
	fetchSnapshotForRekor = func(int) (*snapshot, error) {
		snap := &snapshot{}
		snap.Signed.Meta = map[string]struct {
			Version int   `json:"version"`
			Length  int64 `json:"length"`
		}{"targets.json": {Version: 2}}
		return snap, nil
	}

	fetchTargetsForRekor = func(int) (*targets, error) {
		return makeTargetsWithStatus("Expired", "abc", time.Now().Add(time.Hour)), nil
	}
	if _, err := rekorKey(); err == nil || !strings.Contains(err.Error(), "not active") {
		t.Fatalf("expected inactive key error, got %v", err)
	}

	fetchTargetsForRekor = func(int) (*targets, error) {
		return makeTargetsWithStatus("Active", strings.Repeat("0", 64), time.Now().Add(time.Hour)), nil
	}
	downloadMetadataForRekor = func(string) ([]byte, error) { return []byte("different-key"), nil }
	if _, err := rekorKey(); err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("expected hash mismatch error, got %v", err)
	}
}
