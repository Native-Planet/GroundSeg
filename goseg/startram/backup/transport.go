package backup

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// TransportClient abstracts backup transport calls for injection in tests.
type TransportClient interface {
	Get(url string) (*http.Response, error)
	Post(url, contentType string, body io.Reader) (*http.Response, error)
}

type getBackupRequest struct {
	Ship      string `json:"ship"`
	Pubkey    string `json:"pubkey"`
	Timestamp string `json:"timestamp"`
}

type getBackupResponse struct {
	Result string `json:"result"`
}

// GetBackup retrieves a download link for an encrypted backup payload.
func GetBackup(ship, timestamp, backupPassword, pubkey, endpointURL string, client TransportClient) (string, error) {
	reqData := getBackupRequest{
		Ship:      ship,
		Pubkey:    pubkey,
		Timestamp: timestamp,
	}
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request data: %w", err)
	}

	url := "https://" + endpointURL + "/v1/backup/get"
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to make POST request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response getBackupResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response data: %w", err)
	}
	return response.Result, nil
}

// DownloadAndVerify downloads and validates a backup blob by MD5 checksum.
func DownloadAndVerify(link, md5hash string, client TransportClient) ([]byte, error) {
	resp, err := client.Get(link)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	computedMD5 := fmt.Sprintf("%x", md5.Sum(data))
	if computedMD5 != md5hash {
		return nil, fmt.Errorf("MD5 mismatch: expected %s, got %s", md5hash, computedMD5)
	}
	return data, nil
}

// FetchRemoteBackup coordinates transport + integrity checks for a remote backup.
func FetchRemoteBackup(ship string, timestamp int, md5hash, password, pubkey, endpoint string, client TransportClient) ([]byte, error) {
	link, err := GetBackup(ship, fmt.Sprintf("%d", timestamp), password, pubkey, endpoint, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup: %w", err)
	}
	if link == "" {
		return nil, fmt.Errorf("backup link is empty")
	}
	data, err := DownloadAndVerify(link, md5hash, client)
	if err != nil {
		return nil, fmt.Errorf("failed to download and verify backup: %w", err)
	}
	decryptedData, err := DecryptFile(data, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt backup: %w", err)
	}
	return decryptedData, nil
}
