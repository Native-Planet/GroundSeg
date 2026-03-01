package startram

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"groundseg/config"
	"groundseg/structs"
)

type stubAPIClient struct {
	getFn  func(url string) (*http.Response, error)
	postFn func(url, contentType string, body io.Reader) (*http.Response, error)
}

func (s stubAPIClient) Get(url string) (*http.Response, error) {
	if s.getFn != nil {
		return s.getFn(url)
	}
	return nil, errors.New("unexpected get call")
}

func (s stubAPIClient) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	if s.postFn != nil {
		return s.postFn(url, contentType, body)
	}
	return nil, errors.New("unexpected post call")
}

type stubBackupInfrastructureService struct {
	getBackupFn func(ship, timestamp, backupPassword, pubkey, endpointURL string) (string, error)
	restoreFn   func(req RestoreBackupRequest) error
}

func (s stubBackupInfrastructureService) GetBackup(ship, timestamp, backupPassword, pubkey, endpointURL string) (string, error) {
	if s.getBackupFn != nil {
		return s.getBackupFn(ship, timestamp, backupPassword, pubkey, endpointURL)
	}
	return "", errors.New("unexpected get backup call")
}

func (s stubBackupInfrastructureService) UploadBackup(ship, privateKey, filePath string) error {
	return errors.New("unexpected upload backup call")
}

func (s stubBackupInfrastructureService) Restore(req RestoreBackupRequest) error {
	if s.restoreFn != nil {
		return s.restoreFn(req)
	}
	return errors.New("unexpected restore call")
}

type stubConfigService struct {
	settings config.StartramSettings
	basePath string
}

func (s stubConfigService) StartramSettingsSnapshot() config.StartramSettings {
	return s.settings
}

func (s stubConfigService) IsWgRegistered() bool {
	return s.settings.WgRegistered
}

func (s stubConfigService) SetWgRegistered(bool) error {
	return nil
}

func (s stubConfigService) SetStartramConfig(structs.StartramRetrieve) {}

func (s stubConfigService) BasePath() string {
	return s.basePath
}

func httpResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func TestRestoreBackupMapsRequestFlagsToRestoreRequest(t *testing.T) {
	origService := defaultBackupInfrastructureService
	t.Cleanup(func() {
		defaultBackupInfrastructureService = origService
	})

	var captured []RestoreBackupRequest
	defaultBackupInfrastructureService = stubBackupInfrastructureService{
		restoreFn: func(req RestoreBackupRequest) error {
			captured = append(captured, req)
			return nil
		},
	}

	if err := RestoreBackup("~zod", false, 11, "md5-1", false, "daily"); err != nil {
		t.Fatalf("RestoreBackup local/prod returned error: %v", err)
	}
	if err := RestoreBackup("~bus", true, 22, "md5-2", true, "weekly"); err != nil {
		t.Fatalf("RestoreBackup remote/dev returned error: %v", err)
	}
	if len(captured) != 2 {
		t.Fatalf("expected 2 captured requests, got %d", len(captured))
	}

	localProd := captured[0]
	if localProd.Mode != RestoreBackupModeProduction || localProd.Source != RestoreBackupSourceLocal || localProd.LocalBackupType != "daily" {
		t.Fatalf("unexpected local/prod mapping: %+v", localProd)
	}

	remoteDev := captured[1]
	if remoteDev.Mode != RestoreBackupModeDevelopment || remoteDev.Source != RestoreBackupSourceRemote || remoteDev.LocalBackupType != "weekly" {
		t.Fatalf("unexpected remote/dev mapping: %+v", remoteDev)
	}
}

func TestGetBackupHandlesSuccessAndErrors(t *testing.T) {
	origClient := defaultAPIClient
	t.Cleanup(func() {
		defaultAPIClient = origClient
	})

	defaultAPIClient = stubAPIClient{
		postFn: func(url, contentType string, body io.Reader) (*http.Response, error) {
			if url != "https://api.startram.test/v1/backup/get" {
				t.Fatalf("unexpected URL: %s", url)
			}
			if contentType != "application/json" {
				t.Fatalf("unexpected content type: %s", contentType)
			}
			raw, _ := io.ReadAll(body)
			var req structs.GetBackupRequest
			if err := json.Unmarshal(raw, &req); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			if req.Ship != "~zod" || req.Pubkey != "pubkey" || req.Timestamp != "1234" {
				t.Fatalf("unexpected request payload: %+v", req)
			}
			return httpResponse(http.StatusOK, []byte(`{"result":"https://download/link"}`)), nil
		},
	}

	link, err := getBackup("~zod", "1234", "pw", "pubkey", "api.startram.test")
	if err != nil {
		t.Fatalf("getBackup returned error: %v", err)
	}
	if link != "https://download/link" {
		t.Fatalf("unexpected backup link: %s", link)
	}

	defaultAPIClient = stubAPIClient{
		postFn: func(string, string, io.Reader) (*http.Response, error) {
			return httpResponse(http.StatusBadRequest, []byte("bad request")), nil
		},
	}
	if _, err := getBackup("~zod", "1234", "pw", "pubkey", "api.startram.test"); err == nil || !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected non-200 status error, got %v", err)
	}

	defaultAPIClient = stubAPIClient{
		postFn: func(string, string, io.Reader) (*http.Response, error) {
			return httpResponse(http.StatusOK, []byte(`{"result":`)), nil
		},
	}
	if _, err := getBackup("~zod", "1234", "pw", "pubkey", "api.startram.test"); err == nil || !strings.Contains(err.Error(), "failed to unmarshal response data") {
		t.Fatalf("expected unmarshal error, got %v", err)
	}
}

func TestEncryptDecryptFileRoundTripAndWrongKey(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "backup.bin")
	plaintext := []byte("backup payload bytes")
	if err := os.WriteFile(path, plaintext, 0o644); err != nil {
		t.Fatalf("failed to create plaintext file: %v", err)
	}

	encrypted, err := encryptFile(path, "secret-key")
	if err != nil {
		t.Fatalf("encryptFile returned error: %v", err)
	}
	if bytes.Equal(encrypted, plaintext) {
		t.Fatal("expected encrypted payload to differ from plaintext")
	}

	decrypted, err := decryptFile(encrypted, "secret-key")
	if err != nil {
		t.Fatalf("decryptFile returned error: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("unexpected decrypted payload: got %q want %q", string(decrypted), string(plaintext))
	}

	if _, err := decryptFile(encrypted, "wrong-key"); err == nil || !strings.Contains(err.Error(), "failed to decrypt") {
		t.Fatalf("expected decrypt failure with wrong key, got %v", err)
	}
	if _, err := decryptFile([]byte("short"), "secret-key"); err == nil || !strings.Contains(err.Error(), "ciphertext too short") {
		t.Fatalf("expected short ciphertext error, got %v", err)
	}
}

func TestDownloadAndVerifyChecksMD5(t *testing.T) {
	origClient := defaultAPIClient
	t.Cleanup(func() {
		defaultAPIClient = origClient
	})

	payload := []byte("signed backup content")
	hash := md5.Sum(payload)
	expectedMD5 := fmt.Sprintf("%x", hash)

	defaultAPIClient = stubAPIClient{
		getFn: func(url string) (*http.Response, error) {
			if url != "https://download.startram.test/blob" {
				t.Fatalf("unexpected URL: %s", url)
			}
			return httpResponse(http.StatusOK, payload), nil
		},
	}

	data, err := downloadAndVerify("https://download.startram.test/blob", expectedMD5)
	if err != nil {
		t.Fatalf("downloadAndVerify returned error: %v", err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("unexpected downloaded payload: %q", string(data))
	}

	if _, err := downloadAndVerify("https://download.startram.test/blob", "badmd5"); err == nil || !strings.Contains(err.Error(), "MD5 mismatch") {
		t.Fatalf("expected MD5 mismatch error, got %v", err)
	}
}

func TestRetrieveRemoteBackupEmptyLinkAndSuccess(t *testing.T) {
	origService := defaultBackupInfrastructureService
	origClient := defaultAPIClient
	origConfig := defaultConfigService
	t.Cleanup(func() {
		defaultBackupInfrastructureService = origService
		defaultAPIClient = origClient
		defaultConfigService = origConfig
	})

	defaultConfigService = stubConfigService{
		settings: config.StartramSettings{
			EndpointURL:          "api.startram.test",
			Pubkey:               "pubkey",
			RemoteBackupPassword: "restore-secret",
		},
		basePath: t.TempDir(),
	}

	defaultBackupInfrastructureService = stubBackupInfrastructureService{
		getBackupFn: func(string, string, string, string, string) (string, error) {
			return "", nil
		},
	}
	if _, err := retrieveRemoteBackup("~zod", 100, "ignored-md5"); err == nil || !strings.Contains(err.Error(), "backup link is empty") {
		t.Fatalf("expected empty link error, got %v", err)
	}

	plain := []byte("remote backup payload")
	plainFile := filepath.Join(t.TempDir(), "plain.bin")
	if err := os.WriteFile(plainFile, plain, 0o644); err != nil {
		t.Fatalf("failed to write temp plaintext: %v", err)
	}
	encrypted, err := encryptFile(plainFile, "restore-secret")
	if err != nil {
		t.Fatalf("failed to encrypt test payload: %v", err)
	}
	encryptedMD5 := fmt.Sprintf("%x", md5.Sum(encrypted))

	defaultBackupInfrastructureService = stubBackupInfrastructureService{
		getBackupFn: func(string, string, string, string, string) (string, error) {
			return "https://download.startram.test/blob", nil
		},
	}
	defaultAPIClient = stubAPIClient{
		getFn: func(string) (*http.Response, error) {
			return httpResponse(http.StatusOK, encrypted), nil
		},
	}

	data, err := retrieveRemoteBackup("~zod", 100, encryptedMD5)
	if err != nil {
		t.Fatalf("retrieveRemoteBackup returned error: %v", err)
	}
	if !bytes.Equal(data, plain) {
		t.Fatalf("unexpected decrypted payload: got %q want %q", string(data), string(plain))
	}
}

func TestRestoreBackupProdRejectsUnsupportedSource(t *testing.T) {
	err := restoreBackupProd(RestoreBackupRequest{
		Ship:   "~zod",
		Source: RestoreBackupSource("unsupported"),
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported restore source") {
		t.Fatalf("expected unsupported restore source error, got %v", err)
	}
}
