package routines

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"groundseg/httpx"
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

var (
	fetchTimestampForRekor   = fetchTimestamp
	fetchSnapshotForRekor    = fetchSnapshot
	fetchTargetsForRekor     = fetchTargets
	downloadMetadataForRekor = downloadMetadata
	readCachedKeyForRekor    = readCachedKey
	userHomeDirForRekor      = os.UserHomeDir
	mkdirAllForRekor         = os.MkdirAll
	writeFileForRekor        = os.WriteFile
	timeUntilForRekor        = time.Until
)

type timestamp struct {
	Signed struct {
		Meta map[string]struct {
			Version int `json:"version"`
		} `json:"meta"`
		Expires time.Time `json:"expires"`
	} `json:"signed"`
}

type snapshot struct {
	Signed struct {
		Meta map[string]struct {
			Version int   `json:"version"`
			Length  int64 `json:"length"`
		} `json:"meta"`
		Expires time.Time `json:"expires"`
	} `json:"signed"`
}

type targets struct {
	Signed struct {
		Expires time.Time `json:"expires"`
		Targets map[string]struct {
			Length int               `json:"length"`
			Hashes map[string]string `json:"hashes"`
			Custom struct {
				Sigstore struct {
					Status string `json:"status"`
					URI    string `json:"uri"`
					Usage  string `json:"usage"`
				} `json:"sigstore"`
			} `json:"custom"`
		} `json:"targets"`
		Version int `json:"version"`
	} `json:"signed"`
}

func PrimeRekorKey() {
	_, err := rekorKey()
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to retrieve rekor pubkey: %v", err))
	}
}

func rekorKey() (string, error) {
	ts, err := fetchTimestampForRekor()
	if err != nil {
		return "", fmt.Errorf("fetching timestamp: %v", err)
	}

	snap, err := fetchSnapshotForRekor(ts.Signed.Meta["snapshot.json"].Version)
	if err != nil {
		return "", fmt.Errorf("fetching snapshot: %v", err)
	}

	targ, err := fetchTargetsForRekor(snap.Signed.Meta["targets.json"].Version)
	if err != nil {
		return "", fmt.Errorf("fetching targets: %v", err)
	}

	rekorKeyInfo, ok := targ.Signed.Targets["rekor.pub"]
	if !ok {
		return "", fmt.Errorf("rekor.pub not found in targets metadata")
	}

	if rekorKeyInfo.Custom.Sigstore.Status != "Active" {
		return "", fmt.Errorf("rekor.pub key is not active: %s", rekorKeyInfo.Custom.Sigstore.Status)
	}

	home, err := userHomeDirForRekor()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %v", err)
	}

	fullKeyPath := filepath.Join(home, keyPath)
	needsUpdate := true

	if timeUntilForRekor(targ.Signed.Expires) > 0 {
		if cachedKey, err := readCachedKeyForRekor(fullKeyPath); err == nil {
			h := sha256.Sum256(cachedKey)
			if hex.EncodeToString(h[:]) == rekorKeyInfo.Hashes["sha256"] {
				needsUpdate = false
			}
		}
	}

	if needsUpdate {
		if err := mkdirAllForRekor(filepath.Dir(fullKeyPath), 0755); err != nil {
			return "", fmt.Errorf("creating cache directory: %v", err)
		}

		keyData, err := downloadMetadataForRekor("https://rekor.sigstore.dev/api/v1/log/publicKey")
		if err != nil {
			return "", fmt.Errorf("downloading key data: %v", err)
		}

		h := sha256.Sum256(keyData)
		if hex.EncodeToString(h[:]) != rekorKeyInfo.Hashes["sha256"] {
			return "", fmt.Errorf("key data hash mismatch: got %s, want %s",
				hex.EncodeToString(h[:]), rekorKeyInfo.Hashes["sha256"])
		}

		if err := writeFileForRekor(fullKeyPath, keyData, 0644); err != nil {
			return "", fmt.Errorf("caching rekor key: %v", err)
		}
	}

	return fullKeyPath, nil
}

func fetchTimestamp() (*timestamp, error) {
	data, err := downloadMetadataForRekor(fmt.Sprintf("%s/timestamp.json", tufUrl))
	if err != nil {
		return nil, err
	}
	var ts timestamp
	if err := json.Unmarshal(data, &ts); err != nil {
		return nil, err
	}
	return &ts, nil
}

func fetchSnapshot(version int) (*snapshot, error) {
	data, err := downloadMetadataForRekor(fmt.Sprintf("%s/%d.snapshot.json", tufUrl, version))
	if err != nil {
		return nil, err
	}
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

func fetchTargets(version int) (*targets, error) {
	data, err := downloadMetadataForRekor(fmt.Sprintf("%s/%d.targets.json", tufUrl, version))
	if err != nil {
		return nil, err
	}
	var targ targets
	if err := json.Unmarshal(data, &targ); err != nil {
		return nil, err
	}
	return &targ, nil
}

func downloadMetadata(url string) ([]byte, error) {
	client := &http.Client{Timeout: defaultTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	return httpx.ReadBody(resp, url)
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
