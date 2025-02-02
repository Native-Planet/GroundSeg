package routines

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

const (
	tufUrl         = "https://tuf-repo-cdn.sigstore.dev"
	defaultTimeout = 30 * time.Second
	keyPath        = ".sigstore/root/targets/rekor.pub"
)

type timestampMetadata struct {
	Signed struct {
		Meta map[string]struct {
			Version int `json:"version"`
		} `json:"meta"`
		Expires time.Time `json:"expires"`
	} `json:"signed"`
}

type snapshotMetadata struct {
	Signed struct {
		Meta map[string]struct {
			Version int `json:"version"`
		} `json:"meta"`
		Expires time.Time `json:"expires"`
	} `json:"signed"`
}

type targetsMetadata struct {
	Signed struct {
		Expires time.Time             `json:"expires"`
		Targets map[string]targetInfo `json:"targets"`
		Version int                   `json:"version"`
	} `json:"signed"`
}

type targetInfo struct {
	Length int               `json:"length"`
	Hashes map[string]string `json:"hashes"`
	Custom struct {
		Sigstore struct {
			Status string `json:"status"`
			URI    string `json:"uri"`
			Usage  string `json:"usage"`
		} `json:"sigstore"`
	} `json:"custom"`
}

func init() {
	_, err := rekorKey()
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to retrieve rekor pubkey: %v", err))
	}
}

func rekorKey() (string, error) {
	timestamp, err := fetchTimestamp()
	if err != nil {
		return "", fmt.Errorf("fetching timestamp: %w", err)
	}
	snapshotVersion := timestamp.Signed.Meta["snapshot.json"].Version
	snapshot, err := fetchSnapshot(snapshotVersion)
	if err != nil {
		return "", fmt.Errorf("fetching snapshot: %w", err)
	}
	targetsVersion := snapshot.Signed.Meta["targets.json"].Version
	targets, err := fetchTargets(targetsVersion)
	if err != nil {
		return "", fmt.Errorf("fetching targets: %w", err)
	}
	rekorKeyInfo, ok := targets.Signed.Targets["rekor.pub"]
	if !ok {
		return "", fmt.Errorf("rekor.pub not found in targets metadata")
	}
	if rekorKeyInfo.Custom.Sigstore.Status != "Active" {
		return "", fmt.Errorf("rekor.pub key is not active: %s", rekorKeyInfo.Custom.Sigstore.Status)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	fullKeyPath := filepath.Join(home, keyPath)
	needsUpdate := true
	if time.Until(targets.Signed.Expires) > 0 {
		if cachedKey, err := readCachedKey(fullKeyPath); err == nil {
			h := sha256.Sum256(cachedKey)
			if hex.EncodeToString(h[:]) == rekorKeyInfo.Hashes["sha256"] {
				needsUpdate = false
			}
		}
	}
	if needsUpdate {
		if err := os.MkdirAll(filepath.Dir(fullKeyPath), 0755); err != nil {
			return "", fmt.Errorf("creating cache directory: %w", err)
		}
		keyData := []byte(rekorKeyInfo.Custom.Sigstore.URI)
		h := sha256.Sum256(keyData)
		if hex.EncodeToString(h[:]) != rekorKeyInfo.Hashes["sha256"] {
			return "", fmt.Errorf("key data hash mismatch")
		}
		if err := os.WriteFile(fullKeyPath, keyData, 0644); err != nil {
			return "", fmt.Errorf("caching rekor key: %w", err)
		}
	}
	return fullKeyPath, nil
}

func fetchTimestamp() (*timestampMetadata, error) {
	client := &http.Client{Timeout: defaultTimeout}
	resp, err := client.Get(fmt.Sprintf("%s/timestamp.json", tufUrl))
	if err != nil {
		return nil, fmt.Errorf("fetching timestamp: %w", err)
	}
	defer resp.Body.Close()
	var ts timestampMetadata
	if err := json.NewDecoder(resp.Body).Decode(&ts); err != nil {
		return nil, fmt.Errorf("parsing timestamp: %w", err)
	}
	return &ts, nil
}

func fetchSnapshot(version int) (*snapshotMetadata, error) {
	client := &http.Client{Timeout: defaultTimeout}
	resp, err := client.Get(fmt.Sprintf("%s/%d.snapshot.json", tufUrl, version))
	if err != nil {
		return nil, fmt.Errorf("fetching snapshot: %w", err)
	}
	defer resp.Body.Close()
	var snap snapshotMetadata
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		return nil, fmt.Errorf("parsing snapshot: %w", err)
	}
	return &snap, nil
}

func fetchTargets(version int) (*targetsMetadata, error) {
	client := &http.Client{Timeout: defaultTimeout}
	resp, err := client.Get(fmt.Sprintf("%s/%d.targets.json", tufUrl, version))
	if err != nil {
		return nil, fmt.Errorf("fetching targets: %w", err)
	}
	defer resp.Body.Close()
	var targets targetsMetadata
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, fmt.Errorf("parsing targets: %w", err)
	}
	return &targets, nil
}

func readCachedKey(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if time.Since(info.ModTime()) > 24*time.Hour {
		return nil, fmt.Errorf("cached key is too old")
	}
	return os.ReadFile(path)
}
