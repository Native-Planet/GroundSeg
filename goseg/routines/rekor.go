package routines

import (
	"encoding/json"
	"fmt"
	"io"
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

type targetsMetadata struct {
	Signed struct {
		Expires time.Time             `json:"expires"`
		Version int                   `json:"version"`
		Targets map[string]targetInfo `json:"targets"`
	} `json:"signed"`
}

type timestampMetadata struct {
	Signed struct {
		Type    string    `json:"_type"`
		Expires time.Time `json:"expires"`
		Meta    struct {
			SnapshotJson struct {
				Version int `json:"version"`
			} `json:"snapshot.json"`
		} `json:"meta"`
		Version int `json:"version"`
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
		zap.L().Error(fmt.Sprintf("Failed to get rekor pubkey: %v", err))
	}
}

func rekorKey() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	fullKeyPath := filepath.Join(home, keyPath)
	timestamp, err := fetchTimestamp()
	if err != nil {
		return "", fmt.Errorf("fetching timestamp: %w", err)
	}
	snapshot, err := fetchSnapshot(timestamp.Signed.Meta.SnapshotJson.Version)
	if err != nil {
		return "", fmt.Errorf("fetching snapshot: %w", err)
	}
	targetsMeta, ok := snapshot.Signed.Meta["targets.json"]
	if !ok {
		return "", fmt.Errorf("targets.json not found in snapshot metadata")
	}
	targets, err := fetchTargets(targetsMeta.Version)
	if err != nil {
		return "", fmt.Errorf("fetching targets metadata: %w", err)
	}
	rekorKey, ok := targets.Signed.Targets["rekor.pub"]
	if !ok {
		return "", fmt.Errorf("rekor.pub not found in targets metadata")
	}
	if rekorKey.Custom.Sigstore.Status != "Active" {
		return "", fmt.Errorf("rekor.pub key is not active: %s", rekorKey.Custom.Sigstore.Status)
	}
	needsUpdate := true
	if time.Until(targets.Signed.Expires) > 0 {
		if _, err := readCachedKey(fullKeyPath); err == nil {
			needsUpdate = false
		}
	}
	if needsUpdate {
		if err := os.MkdirAll(filepath.Dir(fullKeyPath), 0755); err != nil {
			return "", fmt.Errorf("creating cache directory: %w", err)
		}
		keyResp, err := http.Get(fmt.Sprintf("%s/targets/rekor.pub", tufUrl))
		if err != nil {
			return "", fmt.Errorf("fetching rekor key: %w", err)
		}
		defer keyResp.Body.Close()
		keyData, err := io.ReadAll(keyResp.Body)
		if err != nil {
			return "", fmt.Errorf("reading key data: %w", err)
		}
		if err := os.WriteFile(fullKeyPath, keyData, 0644); err != nil {
			return "", fmt.Errorf("caching rekor key: %w", err)
		}
	}
	return fullKeyPath, nil
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

func fetchSnapshot(version int) (*snapshotMetadata, error) {
	client := &http.Client{
		Timeout: defaultTimeout,
	}
	resp, err := client.Get(fmt.Sprintf("%s/%d.snapshot.json", tufUrl, version))
	if err != nil {
		return nil, fmt.Errorf("fetching snapshot: %w", err)
	}
	defer resp.Body.Close()
	var snapshot snapshotMetadata
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("parsing snapshot metadata: %w", err)
	}
	return &snapshot, nil
}

func fetchTimestamp() (*timestampMetadata, error) {
	client := &http.Client{
		Timeout: defaultTimeout,
	}
	resp, err := client.Get(tufUrl + "/timestamp.json")
	if err != nil {
		return nil, fmt.Errorf("fetching timestamp: %w", err)
	}
	defer resp.Body.Close()
	var timestamp timestampMetadata
	if err := json.NewDecoder(resp.Body).Decode(&timestamp); err != nil {
		return nil, fmt.Errorf("parsing timestamp metadata: %w", err)
	}
	return &timestamp, nil
}

func fetchTargets(version int) (*targetsMetadata, error) {
	client := &http.Client{
		Timeout: defaultTimeout,
	}
	resp, err := client.Get(fmt.Sprintf("%s/%d.targets.json", tufUrl, version))
	if err != nil {
		return nil, fmt.Errorf("fetching targets: %w", err)
	}
	defer resp.Body.Close()
	var targets targetsMetadata
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, fmt.Errorf("parsing targets metadata: %w", err)
	}
	return &targets, nil
}
