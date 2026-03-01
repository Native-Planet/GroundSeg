package startram

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"groundseg/config"
	"groundseg/structs"
	"net/http"
)

func TestMaskPubkey(t *testing.T) {
	input := "https://api.example/v1/retrieve?pubkey=abc123DEF4560K"
	output := maskPubkey(input)
	if strings.Contains(output, "abc123DEF456") {
		t.Fatalf("expected pubkey body to be masked, got %q", output)
	}
	if !strings.Contains(output, "pubkey=") || !strings.Contains(output, "0K") {
		t.Fatalf("expected pubkey framing to remain, got %q", output)
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backup.bin")
	plain := []byte("groundseg backup payload")
	if err := os.WriteFile(path, plain, 0o644); err != nil {
		t.Fatalf("failed to write plaintext file: %v", err)
	}

	ciphertext, err := encryptFile(path, "test-password")
	if err != nil {
		t.Fatalf("encryptFile returned error: %v", err)
	}
	if len(ciphertext) == 0 {
		t.Fatal("expected non-empty ciphertext")
	}

	decrypted, err := decryptFile(ciphertext, "test-password")
	if err != nil {
		t.Fatalf("decryptFile returned error: %v", err)
	}
	if string(decrypted) != string(plain) {
		t.Fatalf("decrypted text mismatch: want %q got %q", plain, decrypted)
	}
}

type stubStartramAPIClient struct {
	getFn  func(url string) (*http.Response, error)
	postFn func(url, contentType string, body io.Reader) (*http.Response, error)
}

func (client stubStartramAPIClient) Get(url string) (*http.Response, error) {
	return client.getFn(url)
}

func (client stubStartramAPIClient) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	return client.postFn(url, contentType, body)
}

type stubStartramConfigService struct {
	settings             config.StartramSettings
	setWgRegisteredCalls []bool
}

func (service *stubStartramConfigService) StartramSettingsSnapshot() config.StartramSettings {
	return service.settings
}

func (service *stubStartramConfigService) IsWgRegistered() bool {
	return service.settings.WgRegistered
}

func (service *stubStartramConfigService) SetWgRegistered(registered bool) error {
	service.setWgRegisteredCalls = append(service.setWgRegisteredCalls, registered)
	service.settings.WgRegistered = registered
	return nil
}

func (service *stubStartramConfigService) SetStartramConfig(_ structs.StartramRetrieve) {}

func (service *stubStartramConfigService) BasePath() string {
	return ""
}

type stubRetrieveStateSyncer struct {
	calls int
}

func (syncer *stubRetrieveStateSyncer) ApplyRetrieveState(_ structs.StartramRetrieve) error {
	syncer.calls++
	return nil
}

func newStartramHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestRegisterWithInjectedClientAndConfigService(t *testing.T) {
	originalAPIClient := defaultAPIClient
	originalConfigService := defaultConfigService
	originalRetrieveSyncer := defaultRetrieveStateSyncer
	t.Cleanup(func() {
		defaultAPIClient = originalAPIClient
		defaultConfigService = originalConfigService
		defaultRetrieveStateSyncer = originalRetrieveSyncer
	})

	service := &stubStartramConfigService{
		settings: config.StartramSettings{
			EndpointURL: "api.example.com",
			Pubkey:      "abc123DEF4560K",
		},
	}
	syncer := &stubRetrieveStateSyncer{}
	SetConfigService(service)
	SetRetrieveStateSyncer(syncer)
	SetAPIClient(stubStartramAPIClient{
		postFn: func(url, _ string, _ io.Reader) (*http.Response, error) {
			if !strings.Contains(url, "/v1/register") {
				t.Fatalf("unexpected register URL: %s", url)
			}
			return newStartramHTTPResponse(http.StatusOK, `{"error":0}`), nil
		},
		getFn: func(url string) (*http.Response, error) {
			if !strings.Contains(url, "/v1/retrieve") {
				t.Fatalf("unexpected retrieve URL: %s", url)
			}
			return newStartramHTTPResponse(http.StatusOK, `{"status":"active","subdomains":[]}`), nil
		},
	})

	if err := register("reg-code", "us-east"); err != nil {
		t.Fatalf("register returned error: %v", err)
	}
	if len(service.setWgRegisteredCalls) == 0 || !service.setWgRegisteredCalls[0] {
		t.Fatalf("expected registration status to be set true, calls=%v", service.setWgRegisteredCalls)
	}
	if syncer.calls != 1 {
		t.Fatalf("expected retrieve syncer to be called once, got %d", syncer.calls)
	}
}

func TestRegisterReturnsStatusErrorOnNonSuccessHTTP(t *testing.T) {
	originalAPIClient := defaultAPIClient
	originalConfigService := defaultConfigService
	t.Cleanup(func() {
		defaultAPIClient = originalAPIClient
		defaultConfigService = originalConfigService
	})

	service := &stubStartramConfigService{
		settings: config.StartramSettings{
			EndpointURL: "api.example.com",
			Pubkey:      "abc123DEF4560K",
		},
	}
	SetConfigService(service)
	SetAPIClient(stubStartramAPIClient{
		postFn: func(string, string, io.Reader) (*http.Response, error) {
			return newStartramHTTPResponse(http.StatusInternalServerError, "upstream unavailable"), nil
		},
		getFn: func(string) (*http.Response, error) {
			t.Fatal("retrieve should not be called when register response is non-2xx")
			return nil, nil
		},
	})

	err := register("reg-code", "us-east")
	if err == nil {
		t.Fatal("expected register to fail on non-2xx response")
	}
	if !strings.Contains(err.Error(), "unexpected HTTP status 500") {
		t.Fatalf("expected status error message, got %v", err)
	}
}
