package importer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"groundseg/auth/tokens"
	"groundseg/docker/events"
	"groundseg/internal/testruntime"
	"groundseg/lifecycle"
	"groundseg/orchestration"
	"groundseg/structs"

	"github.com/gorilla/mux"
)

func importerRuntimeWith(overrides func(*importerRuntime)) importerRuntime {
	return testruntime.Apply(defaultImporterRuntime(), overrides)
}

func drainImportTransitions() {
	for {
		select {
		case <-events.DefaultEventRuntime().ImportShipTransitions():
		default:
			return
		}
	}
}

func readImportTransitions(t *testing.T, count int) []structs.UploadTransition {
	t.Helper()
	capturedEvents := make([]structs.UploadTransition, 0, count)
	for i := 0; i < count; i++ {
		select {
		case evt := <-events.DefaultEventRuntime().ImportShipTransitions():
			capturedEvents = append(capturedEvents, evt)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for import transition %d", i+1)
		}
	}
	return capturedEvents
}

func copyUploadSessionMap(src map[string]uploadSession) map[string]uploadSession {
	out := make(map[string]uploadSession, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func resetUploadSessionsForTest(t *testing.T) {
	t.Helper()
	uploadMu.Lock()
	originalSessions := copyUploadSessionMap(uploadSessions)
	uploadSessions = map[string]uploadSession{}
	uploadMu.Unlock()
	t.Cleanup(func() {
		uploadMu.Lock()
		uploadSessions = originalSessions
		uploadMu.Unlock()
	})
}

func authorizeTokenIDsForTest(t *testing.T, tokenIDs ...string) importerRuntime {
	t.Helper()
	runtime := defaultImporterRuntime()
	uploadOps := uploadRuntime(t, &runtime)
	authorizedTokens := make(map[string]struct{}, len(tokenIDs))
	for _, tokenID := range tokenIDs {
		authorizedTokens[tokenID] = struct{}{}
	}
	uploadOps.ValidateUploadSessionTokenFn = func(sessionToken structs.WsTokenStruct, providedToken structs.WsTokenStruct, request *http.Request) tokens.UploadTokenAuthorizationResult {
		tokenID := providedToken.ID
		tokenValue := providedToken.Token
		if request != nil {
			if headerTokenID := strings.TrimSpace(request.Header.Get("X-Upload-Token-Id")); headerTokenID != "" {
				tokenID = headerTokenID
			}
			if headerToken := strings.TrimSpace(request.Header.Get("X-Upload-Token")); headerToken != "" {
				tokenValue = headerToken
			}
		}
		if sessionToken != (structs.WsTokenStruct{}) {
			if sessionToken.ID != tokenID || sessionToken.Token != tokenValue {
				return tokens.UploadTokenAuthorizationResult{
					Status:           tokens.UploadValidationStatusTokenContract,
					AuthorizedToken:  tokenValue,
					AuthorizationErr: fmt.Errorf("upload token does not match upload session"),
				}
			}
		}

		if _, ok := authorizedTokens[tokenID]; !ok {
			return tokens.UploadTokenAuthorizationResult{
				Status:           tokens.UploadValidationStatusNotAuthorized,
				AuthorizedToken:  tokenValue,
				AuthorizationErr: fmt.Errorf("token id %s is not authorized", tokenID),
			}
		}

		return tokens.UploadTokenAuthorizationResult{
			Status:          tokens.UploadValidationStatusAuthorized,
			AuthorizedToken: tokenValue,
		}
	}
	return runtime
}

func setUploadDirsForTest(t *testing.T) {
	t.Helper()
	originalUploadDir := uploadDir
	originalTempDir := tempDir
	uploadDir = t.TempDir()
	tempDir = t.TempDir()
	t.Cleanup(func() {
		uploadDir = originalUploadDir
		tempDir = originalTempDir
	})
}

func initRuntime(t *testing.T, runtime *importerRuntime) *importerRuntime {
	t.Helper()
	if runtime == nil {
		t.Fatalf("importer init runtime is not configured")
	}
	return runtime
}

func uploadRuntime(t *testing.T, runtime *importerRuntime) *importerRuntime {
	t.Helper()
	if runtime == nil {
		t.Fatalf("importer upload runtime is not configured")
	}
	return runtime
}

func provisionRuntime(t *testing.T, runtime *importerRuntime) *importerRuntime {
	t.Helper()
	if runtime == nil {
		t.Fatalf("importer provision runtime is not configured")
	}
	return runtime
}

func TestResetPublishesBaselineTransitions(t *testing.T) {
	drainImportTransitions()

	if err := Reset(); err != nil {
		t.Fatalf("Reset returned error: %v", err)
	}
	events := readImportTransitions(t, 4)

	expected := []structs.UploadTransition{
		{Type: "status", Event: ""},
		{Type: "patp", Event: ""},
		{Type: "error", Event: ""},
		{Type: "extracted", Value: 0},
	}
	for i, want := range expected {
		if events[i] != want {
			t.Fatalf("unexpected event[%d]: want %+v got %+v", i, want, events[i])
		}
	}
}

func TestFailUploadRequestPublishesFailureTransitions(t *testing.T) {
	drainImportTransitions()

	recorder := httptest.NewRecorder()
	written, err := failUploadRequest(recorder, http.StatusUnauthorized, "bad token")
	if !written {
		t.Fatal("expected failUploadRequest to write a response")
	}
	if err != nil {
		t.Fatalf("failUploadRequest returned error: %v", err)
	}

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: got %d", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"status": "failure"`) {
		t.Fatalf("response body missing failure status: %s", body)
	}
	if !strings.Contains(body, "bad token") {
		t.Fatalf("response body missing error message: %s", body)
	}

	events := readImportTransitions(t, 2)
	if events[0].Type != "error" || events[0].Event != "bad token" {
		t.Fatalf("unexpected error event: %+v", events[0])
	}
	if events[1].Type != "status" || events[1].Event != "aborted" {
		t.Fatalf("unexpected status event: %+v", events[1])
	}
}

func TestFailUploadRequestReturnsPublishError(t *testing.T) {
	publishErr := errors.New("publish transport failure")
	observed := make([]structs.UploadTransition, 0, 2)

	runtime := defaultImporterRuntime()
	runtime.PublishImportTransitionFn = func(_ context.Context, uploadTransition structs.UploadTransition) error {
		observed = append(observed, uploadTransition)
		return publishErr
	}

	recorder := httptest.NewRecorder()
	written, err := failUploadRequest(recorder, http.StatusInternalServerError, "failed", runtime)
	if !written {
		t.Fatal("expected failUploadRequest to write a response")
	}
	if err == nil {
		t.Fatalf("expected failUploadRequest to return publish failure, observed=%d transitions, err=%v", len(observed), err)
	}
	if !errors.Is(err, publishErr) {
		t.Fatalf("expected wrapped publish failure, got %v", err)
	}
	if len(observed) == 0 {
		t.Fatal("expected publish transition to be attempted")
	}
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: got %d", recorder.Code)
	}
}

type responseWriterWithWriteFailure struct {
	statusCode int
	writeErr   error
	headers    http.Header
}

func (w *responseWriterWithWriteFailure) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *responseWriterWithWriteFailure) WriteHeader(code int) {
	w.statusCode = code
}

func (w *responseWriterWithWriteFailure) Write(body []byte) (int, error) {
	return 0, w.writeErr
}

func TestSendUploadResponsePropagatesWriteError(t *testing.T) {
	writeErr := errors.New("write failed")
	writer := &responseWriterWithWriteFailure{writeErr: writeErr}

	if err := sendUploadResponse(writer, http.StatusOK, "success", "ok"); err == nil || !errors.Is(err, writeErr) {
		t.Fatalf("expected write error to propagate, got %v", err)
	}
}

func TestFailUploadRequestPropagatesWriteError(t *testing.T) {
	writeErr := errors.New("write failed")
	writer := &responseWriterWithWriteFailure{writeErr: writeErr}

	_, err := failUploadRequest(writer, http.StatusBadRequest, "oops")
	if err == nil || !errors.Is(err, writeErr) {
		t.Fatalf("expected write error to propagate, got %v", err)
	}
}

func TestSetUploadResponseHeadersUsesTrustedRequestOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://groundseg.local/upload", nil)
	req.Host = "groundseg.local"
	req.Header.Set("Origin", "http://groundseg.local")
	recorder := httptest.NewRecorder()

	setUploadResponseHeaders(recorder, req)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://groundseg.local" {
		t.Fatalf("expected allowed origin header, got %q", got)
	}
	if got := recorder.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("expected Vary header set to Origin, got %q", got)
	}
}

func TestSetUploadResponseHeadersRejectsMismatchedOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://groundseg.local/upload", nil)
	req.Host = "groundseg.local"
	req.Header.Set("Origin", "http://evil.local")
	recorder := httptest.NewRecorder()

	setUploadResponseHeaders(recorder, req)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected mismatched origin to be rejected, got %q", got)
	}
}

func TestRunImportPhasesEmitsLifecycleStatuses(t *testing.T) {
	drainImportTransitions()

	err := runImportPhases(
		"archive.tgz",
		"~zod",
		"",
		orchestration.WorkflowPhases{
			Execute: []lifecycle.Step{
				{
					Phase: lifecycle.Phase("creating"),
					Run: func() error {
						return nil
					},
				},
				{
					Phase: lifecycle.Phase("booting"),
					Run: func() error {
						return nil
					},
				},
			},
		},
		defaultImporterRuntime(),
	)
	if err != nil {
		t.Fatalf("runImportPhases returned error: %v", err)
	}

	events := readImportTransitions(t, 2)
	if events[0].Type != "status" || events[0].Event != "creating" {
		t.Fatalf("unexpected creating event: %+v", events[0])
	}
	if events[1].Type != "status" || events[1].Event != "booting" {
		t.Fatalf("unexpected booting event: %+v", events[1])
	}
}

func TestValidateUploadRequestRejectsMissingHeaders(t *testing.T) {
	uploadMu.Lock()
	originalSessions := copyUploadSessionMap(uploadSessions)
	uploadSessions = map[string]uploadSession{
		"abcdabcdabcdabcdabcdabcdabcdabcd": {
			Token: structs.WsTokenStruct{
				ID:    "token-id",
				Token: "token-value",
			},
			ExpiresAt: time.Now().Add(1 * time.Hour),
		},
	}
	uploadMu.Unlock()
	t.Cleanup(func() {
		uploadMu.Lock()
		uploadSessions = originalSessions
		uploadMu.Unlock()
	})

	req := httptest.NewRequest(http.MethodPost, "/upload", nil)
	req = mux.SetURLVars(req, map[string]string{
		"uploadSession": "abcdabcdabcdabcdabcdabcdabcdabcd",
		"patp":          "~zod",
	})

	_, err := validateUploadRequest(req, "abcdabcdabcdabcdabcdabcdabcdabcd", "~zod")
	if err == nil {
		t.Fatal("expected validateUploadRequest to fail without auth headers")
	}
	if !strings.Contains(err.Error(), "missing upload authorization headers") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateUploadRequestRejectsTokenMismatch(t *testing.T) {
	uploadMu.Lock()
	originalSessions := copyUploadSessionMap(uploadSessions)
	uploadSessions = map[string]uploadSession{
		"abcdabcdabcdabcdabcdabcdabcdabcd": {
			Token: structs.WsTokenStruct{
				ID:    "token-id",
				Token: "token-value",
			},
			ExpiresAt: time.Now().Add(1 * time.Hour),
		},
	}
	uploadMu.Unlock()
	t.Cleanup(func() {
		uploadMu.Lock()
		uploadSessions = originalSessions
		uploadMu.Unlock()
	})

	req := httptest.NewRequest(http.MethodPost, "/upload", nil)
	req = mux.SetURLVars(req, map[string]string{
		"uploadSession": "abcdabcdabcdabcdabcdabcdabcdabcd",
		"patp":          "~zod",
	})
	req.Header.Set("X-Upload-Token-Id", "token-id")
	req.Header.Set("X-Upload-Token", "different")

	_, err := validateUploadRequest(req, "abcdabcdabcdabcdabcdabcdabcdabcd", "~zod")
	if err == nil {
		t.Fatal("expected validateUploadRequest to fail on token mismatch")
	}
	if !strings.Contains(err.Error(), "upload token does not match upload session") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitializeReturnsErrorForStoragePathFailures(t *testing.T) {
	runtime := defaultImporterRuntime()
	initOps := initRuntime(t, &runtime)
	initOps.StoragePathForFn = func(_ string) (string, error) {
		return "", errors.New("storage unavailable")
	}

	if err := Initialize(runtime); err == nil {
		t.Fatal("expected Initialize to fail when storage path cannot be resolved")
	}
}

func TestInitializeSetsUploadAndTempDirectories(t *testing.T) {
	runtime := defaultImporterRuntime()
	base := t.TempDir()
	initOps := initRuntime(t, &runtime)
	initOps.StoragePathForFn = func(pathType string) (string, error) {
		if pathType == "uploads" {
			return filepath.Join(base, "uploads"), nil
		}
		return filepath.Join(base, "temp"), nil
	}

	if err := Initialize(runtime); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	if uploadDir != filepath.Join(base, "uploads") {
		t.Fatalf("unexpected uploadDir %q", uploadDir)
	}
	if tempDir != filepath.Join(base, "temp") {
		t.Fatalf("unexpected tempDir %q", tempDir)
	}
}
