package transport

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"groundseg/startram/backup/crypto"
)

type stubTransportClient struct {
	getFn  func(string) (*http.Response, error)
	postFn func(string, string, io.Reader) (*http.Response, error)
}

func (stub stubTransportClient) Get(url string) (*http.Response, error) {
	if stub.getFn == nil {
		return nil, errors.New("unexpected GET request")
	}
	return stub.getFn(url)
}

func (stub stubTransportClient) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	if stub.postFn == nil {
		return nil, errors.New("unexpected POST request")
	}
	return stub.postFn(url, contentType, body)
}

func testResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestGetBackupSuccess(t *testing.T) {
	t.Parallel()

	var gotURL string
	client := stubTransportClient{
		postFn: func(url, contentType string, body io.Reader) (*http.Response, error) {
			gotURL = url
			if contentType != "application/json" {
				t.Fatalf("unexpected content type: %s", contentType)
			}
			payload, err := io.ReadAll(body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			if !strings.Contains(string(payload), "\"ship\":\"~zod\"") {
				t.Fatalf("expected ship payload in request body, got %s", string(payload))
			}
			return testResponse(http.StatusOK, `{"result":"https://files.example/backup.enc"}`), nil
		},
	}

	link, err := GetBackup("~zod", "123", "ignored-password", "pubkey", "endpoint.example", client)
	if err != nil {
		t.Fatalf("GetBackup returned error: %v", err)
	}
	if link != "https://files.example/backup.enc" {
		t.Fatalf("unexpected backup link: %q", link)
	}
	if gotURL != "https://endpoint.example/v1/backup/get" {
		t.Fatalf("unexpected POST URL: %s", gotURL)
	}
}

func TestGetBackupReturnsStatusFailure(t *testing.T) {
	t.Parallel()

	client := stubTransportClient{
		postFn: func(string, string, io.Reader) (*http.Response, error) {
			return testResponse(http.StatusBadGateway, "gateway unavailable"), nil
		},
	}

	_, err := GetBackup("~zod", "123", "ignored-password", "pubkey", "endpoint.example", client)
	if err == nil || !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("expected HTTP status error, got %v", err)
	}
}

func TestDownloadAndVerifyRejectsChecksumMismatch(t *testing.T) {
	t.Parallel()

	client := stubTransportClient{
		getFn: func(string) (*http.Response, error) {
			return testResponse(http.StatusOK, "payload"), nil
		},
	}

	_, err := DownloadAndVerify("https://files.example/backup.enc", "deadbeef", client)
	if err == nil || !strings.Contains(err.Error(), "MD5 mismatch") {
		t.Fatalf("expected checksum mismatch error, got %v", err)
	}
}

func TestFetchRemoteBackupRejectsEmptyLink(t *testing.T) {
	t.Parallel()

	client := stubTransportClient{
		postFn: func(string, string, io.Reader) (*http.Response, error) {
			return testResponse(http.StatusOK, `{"result":""}`), nil
		},
		getFn: func(string) (*http.Response, error) {
			t.Fatal("unexpected download for empty link")
			return nil, errors.New("unreachable")
		},
	}

	_, err := FetchRemoteBackup("~zod", 123, "hash", "password", "pubkey", "endpoint.example", client)
	if err == nil || !strings.Contains(err.Error(), "backup link is empty") {
		t.Fatalf("expected empty link failure, got %v", err)
	}
}

func TestFetchRemoteBackupDecryptsPayload(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	plainPath := filepath.Join(dir, "payload.bin")
	plaintext := []byte("restore-me")
	if err := os.WriteFile(plainPath, plaintext, 0o600); err != nil {
		t.Fatalf("write plaintext: %v", err)
	}
	encrypted, err := crypto.EncryptFile(plainPath, "password")
	if err != nil {
		t.Fatalf("EncryptFile returned error: %v", err)
	}
	md5hash := fmt.Sprintf("%x", md5.Sum(encrypted))

	client := stubTransportClient{
		postFn: func(string, string, io.Reader) (*http.Response, error) {
			return testResponse(http.StatusOK, `{"result":"https://files.example/encrypted.bin"}`), nil
		},
		getFn: func(string) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(encrypted)),
				Header:     make(http.Header),
			}, nil
		},
	}

	decrypted, err := FetchRemoteBackup("~zod", 123, md5hash, "password", "pubkey", "endpoint.example", client)
	if err != nil {
		t.Fatalf("FetchRemoteBackup returned error: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("unexpected decrypted payload: got %q want %q", string(decrypted), string(plaintext))
	}
}

func TestUploadBackupSuccess(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "backup.payload")
	if err := os.WriteFile(filePath, []byte("payload"), 0o600); err != nil {
		t.Fatalf("write payload file: %v", err)
	}

	var postedURL string
	client := stubTransportClient{
		postFn: func(url, contentType string, body io.Reader) (*http.Response, error) {
			postedURL = url

			payload, err := io.ReadAll(body)
			if err != nil {
				t.Fatalf("read multipart payload: %v", err)
			}
			req, err := http.NewRequest(http.MethodPost, "http://local", bytes.NewReader(payload))
			if err != nil {
				t.Fatalf("create request: %v", err)
			}
			req.Header.Set("Content-Type", contentType)
			if err := req.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("parse multipart form: %v", err)
			}
			if req.FormValue("ship") != "~zod" {
				t.Fatalf("unexpected ship form field: %q", req.FormValue("ship"))
			}
			if req.FormValue("pubkey") != "pubkey" {
				t.Fatalf("unexpected pubkey form field: %q", req.FormValue("pubkey"))
			}
			file, _, err := req.FormFile("file")
			if err != nil {
				t.Fatalf("expected encrypted file payload: %v", err)
			}
			defer file.Close()
			encryptedPayload, err := io.ReadAll(file)
			if err != nil {
				t.Fatalf("read encrypted payload: %v", err)
			}
			if len(encryptedPayload) == 0 {
				t.Fatal("expected encrypted payload bytes")
			}

			return testResponse(http.StatusOK, ""), nil
		},
	}

	if err := UploadBackup("~zod", "private-key", "endpoint.example", "pubkey", filePath, client); err != nil {
		t.Fatalf("UploadBackup returned error: %v", err)
	}
	if postedURL != "https://endpoint.example/v1/backup/upload" {
		t.Fatalf("unexpected upload URL: %s", postedURL)
	}
}

func TestUploadBackupReturnsStatusFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "backup.payload")
	if err := os.WriteFile(filePath, []byte("payload"), 0o600); err != nil {
		t.Fatalf("write payload file: %v", err)
	}

	client := stubTransportClient{
		postFn: func(string, string, io.Reader) (*http.Response, error) {
			return testResponse(http.StatusBadRequest, "nope"), nil
		},
	}

	if err := UploadBackup("~zod", "private-key", "endpoint.example", "pubkey", filePath, client); err == nil {
		t.Fatal("expected UploadBackup to fail on non-200 response")
	}
}
