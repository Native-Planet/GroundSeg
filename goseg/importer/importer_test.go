package importer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"groundseg/auth/tokens"
	"groundseg/docker/events"
	"groundseg/lifecycle"
	"groundseg/orchestration"
	"groundseg/shipworkflow"
	"groundseg/structs"

	"github.com/docker/docker/errdefs"
	"github.com/gorilla/mux"
)

func importerRuntimeWith(overrides func(*importerRuntime)) importerRuntime {
	runtime := defaultImporterRuntime()
	if overrides != nil {
		overrides(&runtime)
	}
	return runtime
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
	uploadOps.ValidateUploadSessionTokenFn = func(sessionToken structs.WsTokenStruct, providedToken structs.WsTokenStruct, _ *http.Request) tokens.UploadTokenAuthorizationResult {
		if sessionToken != (structs.WsTokenStruct{}) {
			if sessionToken.ID != providedToken.ID || sessionToken.Token != providedToken.Token {
				return tokens.UploadTokenAuthorizationResult{
					Status:           tokens.UploadValidationStatusTokenContract,
					AuthorizedToken:  providedToken.Token,
					AuthorizationErr: fmt.Errorf("upload token does not match upload session"),
				}
			}
		}

		if _, ok := authorizedTokens[providedToken.ID]; !ok {
			return tokens.UploadTokenAuthorizationResult{
				Status:           tokens.UploadValidationStatusNotAuthorized,
				AuthorizedToken:  providedToken.Token,
				AuthorizationErr: fmt.Errorf("token id %s is not authorized", providedToken.ID),
			}
		}

		return tokens.UploadTokenAuthorizationResult{
			Status:          tokens.UploadValidationStatusAuthorized,
			AuthorizedToken: providedToken.Token,
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
	if err := failUploadRequest(recorder, http.StatusUnauthorized, "bad token"); err != nil {
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

	if err := failUploadRequest(writer, http.StatusBadRequest, "oops"); err == nil || !errors.Is(err, writeErr) {
		t.Fatalf("expected write error to propagate, got %v", err)
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

func TestOpenUploadEndpointRejectsInvalidSessionKey(t *testing.T) {
	err := OpenUploadEndpoint(OpenUploadEndpointCmd{
		Endpoint: "not-a-valid-key",
		Token: structs.WsTokenStruct{
			ID:    "token-id",
			Token: "token-value",
		},
		SelectedDrive: "system-drive",
	})
	if err == nil {
		t.Fatal("expected invalid session key error")
	}
	if !strings.Contains(err.Error(), "invalid upload session key format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenUploadEndpointCreatesAndUpdatesSystemDriveSession(t *testing.T) {
	resetUploadSessionsForTest(t)
	runtime := authorizeTokenIDsForTest(t, "token-id")

	endpoint := "0123456789abcdef0123456789abcdef"
	createCmd := OpenUploadEndpointCmd{
		Endpoint: endpoint,
		Token: structs.WsTokenStruct{
			ID:    "token-id",
			Token: "token-value",
		},
		Remote:        false,
		Fix:           false,
		SelectedDrive: "system-drive",
	}
	if err := OpenUploadEndpoint(createCmd, runtime); err != nil {
		t.Fatalf("OpenUploadEndpoint(create) returned error: %v", err)
	}

	uploadMu.Lock()
	createdSession, exists := uploadSessions[endpoint]
	uploadMu.Unlock()
	if !exists {
		t.Fatalf("expected upload session %s to be created", endpoint)
	}
	if createdSession.NeedsFormatting {
		t.Fatal("system-drive session should not require formatting")
	}
	if createdSession.Remote || createdSession.Fix {
		t.Fatalf("unexpected initial flags in created session: %+v", createdSession)
	}

	updateCmd := createCmd
	updateCmd.Remote = true
	updateCmd.Fix = true
	if err := OpenUploadEndpoint(updateCmd, runtime); err != nil {
		t.Fatalf("OpenUploadEndpoint(update) returned error: %v", err)
	}

	uploadMu.Lock()
	updated := uploadSessions[endpoint]
	uploadMu.Unlock()
	if !updated.Remote || !updated.Fix {
		t.Fatalf("expected updated session flags to be true, got %+v", updated)
	}
}

func TestOpenUploadEndpointRejectsTokenMismatchOnExistingSession(t *testing.T) {
	resetUploadSessionsForTest(t)
	runtime := authorizeTokenIDsForTest(t, "token-id", "other-token-id")

	endpoint := "0123456789abcdef0123456789abcdef"
	if err := OpenUploadEndpoint(OpenUploadEndpointCmd{
		Endpoint: endpoint,
		Token: structs.WsTokenStruct{
			ID:    "token-id",
			Token: "token-value",
		},
		SelectedDrive: "system-drive",
	}, runtime); err != nil {
		t.Fatalf("OpenUploadEndpoint(create) returned error: %v", err)
	}

	err := OpenUploadEndpoint(OpenUploadEndpointCmd{
		Endpoint: endpoint,
		Token: structs.WsTokenStruct{
			ID:    "other-token-id",
			Token: "other-token-value",
		},
		SelectedDrive: "system-drive",
	}, runtime)
	if err == nil {
		t.Fatal("expected token mismatch error")
	}
	if !strings.Contains(err.Error(), "token mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetUploadSessionWrapsOpenUploadEndpoint(t *testing.T) {
	resetUploadSessionsForTest(t)
	runtime := authorizeTokenIDsForTest(t, "token-id")

	endpoint := "0123456789abcdef0123456789abcdef"
	payload := structs.WsUploadPayload{
		Token: structs.WsTokenStruct{
			ID:    "token-id",
			Token: "token-value",
		},
		Payload: structs.WsUploadAction{
			Endpoint:      endpoint,
			Remote:        true,
			Fix:           true,
			SelectedDrive: "system-drive",
		},
	}
	if err := SetUploadSession(payload, runtime); err != nil {
		t.Fatalf("SetUploadSession returned error: %v", err)
	}

	uploadMu.Lock()
	sesh, exists := uploadSessions[endpoint]
	uploadMu.Unlock()
	if !exists {
		t.Fatalf("expected upload session %s to be present", endpoint)
	}
	if !sesh.Remote || !sesh.Fix {
		t.Fatalf("expected remote+fix true in session, got %+v", sesh)
	}
}

func TestParseUploadChunkMetadata(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/upload?dzchunkindex=2&dztotalchunkcount=5", nil)
	index, total, err := parseUploadChunkMetadata(req, "ship.tar.gz")
	if err != nil {
		t.Fatalf("parseUploadChunkMetadata returned error: %v", err)
	}
	if index != 2 || total != 5 {
		t.Fatalf("unexpected parsed values: index=%d total=%d", index, total)
	}

	badIndexReq := httptest.NewRequest(http.MethodPost, "/upload?dzchunkindex=bad&dztotalchunkcount=5", nil)
	if _, _, err := parseUploadChunkMetadata(badIndexReq, "ship.tar.gz"); err == nil {
		t.Fatal("expected invalid chunk index error")
	}
}

func TestPersistChunkAndCombineChunks(t *testing.T) {
	setUploadDirsForTest(t)

	filename := "ship.tar"
	parts := []string{"abc", "123", "xyz"}
	for index, part := range parts {
		if err := persistChunkToTemp(strings.NewReader(part), filename, index); err != nil {
			t.Fatalf("persistChunkToTemp(%d) returned error: %v", index, err)
		}
	}
	allChunks, err := allChunksReceived(filename, len(parts))
	if err != nil {
		t.Fatalf("allChunksReceived returned error: %v", err)
	}
	if !allChunks {
		t.Fatal("expected allChunksReceived to be true after writing all parts")
	}

	if err := combineChunks(filename, len(parts)); err != nil {
		t.Fatalf("combineChunks returned error: %v", err)
	}
	combinedPath := filepath.Join(uploadDir, filename)
	data, err := os.ReadFile(combinedPath)
	if err != nil {
		t.Fatalf("read combined file: %v", err)
	}
	if string(data) != "abc123xyz" {
		t.Fatalf("unexpected combined output: %q", string(data))
	}
	for i := range parts {
		partPath := filepath.Join(tempDir, filename+"-part-"+strconv.Itoa(i))
		if _, err := os.Stat(partPath); !os.IsNotExist(err) {
			t.Fatalf("expected chunk file %s to be removed, stat err=%v", partPath, err)
		}
	}
}

func TestPersistChunkToTempPropagatesCloseError(t *testing.T) {
	setUploadDirsForTest(t)
	closeErr := errors.New("close failed")
	runtime := importerRuntimeWith(func(runtime *importerRuntime) {
		uploadOps := uploadRuntime(t, runtime)
		uploadOps.CloseTempFileFn = func(*os.File) error {
			return closeErr
		}
	})

	err := persistChunkToTemp(strings.NewReader("payload"), "ship.tgz", 0, runtime)
	if err == nil || !errors.Is(err, closeErr) {
		t.Fatalf("expected close error wrapped, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(tempDir, "ship.tgz-part-0")); !os.IsNotExist(statErr) {
		t.Fatalf("expected temp chunk file to be removed when close fails, got %v", statErr)
	}
}

func TestAllChunksReceivedPropagatesFilesystemError(t *testing.T) {
	setUploadDirsForTest(t)
	statErr := errors.New("stat access denied")
	runtime := importerRuntimeWith(func(runtime *importerRuntime) {
		uploadOps := uploadRuntime(t, runtime)
		uploadOps.StatFn = func(string) (os.FileInfo, error) {
			return nil, statErr
		}
	})

	if _, err := allChunksReceived("ship.tgz", 1, runtime); err == nil || !errors.Is(err, statErr) {
		t.Fatalf("expected stat error to propagate, got %v", err)
	}
}

func TestCombineChunksReturnsErrorWhenChunkMissing(t *testing.T) {
	setUploadDirsForTest(t)

	filename := "ship.tar"
	if err := persistChunkToTemp(strings.NewReader("only-first"), filename, 0); err != nil {
		t.Fatalf("persistChunkToTemp returned error: %v", err)
	}

	err := combineChunks(filename, 2)
	if err == nil {
		t.Fatal("expected combineChunks to fail when a chunk is missing")
	}
	if !strings.Contains(err.Error(), "failed to open chunk file 1") {
		t.Fatalf("unexpected combine error: %v", err)
	}
}

func TestConfigureUploadedPierRunsPostImportWorkflowAndPropagatesErrors(t *testing.T) {
	setUploadDirsForTest(t)
	runtime := defaultImporterRuntime()
	initOps := initRuntime(t, &runtime)
	uploadPath := uploadDir
	tempPath := tempDir
	initOps.StoragePathForFn = func(pathType string) (string, error) {
		switch pathType {
		case "uploads":
			return uploadPath, nil
		case "temp":
			return tempPath, nil
		default:
			return "", fmt.Errorf("unexpected path type: %s", pathType)
		}
	}

	configureErr := errors.New("configure failed")
	postErr := errors.New("post-process failed")

	workflowCalled := false
	postCalled := false
	provisionOps := provisionRuntime(t, &runtime)
	provisionOps.RunImportedPierWorkflowFn = func(_ importedPierContext, _ importerRuntime) error {
		workflowCalled = true
		t.Logf("configure workflow called")
		return nil
	}
	provisionOps.RunImportedPierPostImportWorkflowFn = func(_ importedPierContext, _ importerRuntime) error {
		postCalled = true
		t.Logf("post-import workflow called")
		return nil
	}
	provisionOps.CleanupMultipartFn = func(string) error { return nil }

	if err := configureUploadedPier(context.Background(), shipworkflow.UploadImportCommand{
		ArchivePath: filepath.Join(uploadDir, "ship.tgz"),
		Filename:    "ship.tgz",
		Patp:        "~zod",
	}, runtime); err != nil {
		t.Fatalf("configureUploadedPier returned unexpected error: %v", err)
	}
	if !workflowCalled || !postCalled {
		t.Fatalf("expected both imported-pier workflows to be called, got workflow=%v post=%v", workflowCalled, postCalled)
	}

	workflowCalled = false
	postCalled = false
	provisionOps.RunImportedPierWorkflowFn = func(_ importedPierContext, _ importerRuntime) error {
		workflowCalled = true
		return configureErr
	}
	provisionOps.RunImportedPierPostImportWorkflowFn = func(_ importedPierContext, _ importerRuntime) error {
		postCalled = false
		return nil
	}
	if err := configureUploadedPier(context.Background(), shipworkflow.UploadImportCommand{
		ArchivePath: filepath.Join(uploadDir, "ship.tgz"),
		Filename:    "ship.tgz",
		Patp:        "~zod",
	}, runtime); err == nil || !errors.Is(err, configureErr) {
		t.Fatalf("expected configure phase failure to be returned, got: %v", err)
	}
	if !workflowCalled || postCalled {
		t.Fatalf("expected only configure workflow to run on configure failure, got workflow=%v post=%v", workflowCalled, postCalled)
	}

	workflowCalled = false
	postCalled = false
	provisionOps.RunImportedPierWorkflowFn = func(_ importedPierContext, _ importerRuntime) error {
		workflowCalled = true
		return nil
	}
	provisionOps.RunImportedPierPostImportWorkflowFn = func(_ importedPierContext, _ importerRuntime) error {
		postCalled = true
		return postErr
	}
	if err := configureUploadedPier(context.Background(), shipworkflow.UploadImportCommand{
		ArchivePath: filepath.Join(uploadDir, "ship.tgz"),
		Filename:    "ship.tgz",
		Patp:        "~zod",
	}, runtime); err == nil || !errors.Is(err, postErr) {
		t.Fatalf("expected post-import workflow failure to be returned, got: %v", err)
	}
	if !workflowCalled || !postCalled {
		t.Fatalf("expected both workflows to run, got workflow=%v post=%v", workflowCalled, postCalled)
	}
}

func TestFinalizeUploadOnCompletionDispatchesUploadImportCommand(t *testing.T) {
	setUploadDirsForTest(t)

	runtime := defaultImporterRuntime()
	provisionOps := provisionRuntime(t, &runtime)

	requestCtx := context.WithValue(context.Background(), "test-key", "test-value")
	var observedCmd shipworkflow.UploadImportCommand
	provisionOps.UploadCoordinatorFn = func(ctx context.Context, cmd shipworkflow.UploadImportCommand) error {
		if ctx != requestCtx {
			t.Fatalf("unexpected dispatch context: %v", ctx)
		}
		observedCmd = cmd
		return nil
	}

	if err := persistChunkToTemp(strings.NewReader("payload"), "ship.tgz", 0); err != nil {
		t.Fatalf("persistChunkToTemp returned error: %v", err)
	}

	completed, err := finalizeUploadOnCompletion(
		uploadChunkProgress{
			Filename:  "ship.tgz",
			Total:     1,
			AllChunks: true,
		},
		validatedUploadRequest{
			Context: requestCtx,
			Patp:    "~zod",
			Session: uploadSession{
				Remote:      true,
				Fix:         true,
				CustomDrive: "/custom/drive",
			},
		},
		runtime,
	)
	if err != nil {
		t.Fatalf("finalizeUploadOnCompletion returned unexpected error: %v", err)
	}
	if !completed {
		t.Fatalf("expected completion to be true after final chunk")
	}
	if observedCmd.ArchivePath != filepath.Join(uploadDir, "ship.tgz") {
		t.Fatalf("unexpected archive path: %q", observedCmd.ArchivePath)
	}
	if observedCmd.Patp != "~zod" {
		t.Fatalf("unexpected patp: %q", observedCmd.Patp)
	}
	if !observedCmd.Remote || !observedCmd.Fix {
		t.Fatalf("expected flags to be propagated, got remote=%v fix=%v", observedCmd.Remote, observedCmd.Fix)
	}
	if observedCmd.CustomDrive != "/custom/drive" {
		t.Fatalf("unexpected custom drive: %q", observedCmd.CustomDrive)
	}
}

func TestFinalizeUploadOnCompletionWrapsDispatchFailureAndImportError(t *testing.T) {
	setUploadDirsForTest(t)

	dispatchErr := errors.New("dispatch failed")
	runtime := defaultImporterRuntime()
	provisionOps := provisionRuntime(t, &runtime)
	provisionOps.UploadCoordinatorFn = func(context.Context, shipworkflow.UploadImportCommand) error {
		return dispatchErr
	}

	if err := persistChunkToTemp(strings.NewReader("payload"), "ship.tgz", 0); err != nil {
		t.Fatalf("persistChunkToTemp returned error: %v", err)
	}
	completed, err := finalizeUploadOnCompletion(
		uploadChunkProgress{
			Filename:  "ship.tgz",
			Total:     1,
			AllChunks: true,
		},
		validatedUploadRequest{
			Patp: "~zod",
		},
		runtime,
	)
	if err == nil {
		t.Fatal("expected finalizeUploadOnCompletion to return an error")
	}
	if !completed {
		t.Fatalf("expected completion to be true after final chunk")
	}
	if !errors.Is(err, errImportPierConfig) {
		t.Fatalf("expected import config sentinel in error chain, got %v", err)
	}
	if !errors.Is(err, dispatchErr) {
		t.Fatalf("expected dispatch error in error chain, got %v", err)
	}
}

func TestFinalizeImportedPierReadinessReturnsAcmeFixErrorWhenFixIsEnabled(t *testing.T) {
	runtime := importerRuntimeWith(func(runtime *importerRuntime) {
		provisionOps := provisionRuntime(t, runtime)
		provisionOps.ShipworkflowWaitForBootCodeFn = func(string, time.Duration) {}
		provisionOps.ShipworkflowWaitForRemoteReadyFn = func(string, time.Duration) {}
		provisionOps.ShipworkflowSwitchToWireguardFn = func(string, bool) error {
			return nil
		}
	})
	expectedErr := errors.New("acme fix failed")
	provisionOps := provisionRuntime(t, &runtime)
	provisionOps.AcmeFixFn = func(string) error {
		return expectedErr
	}

	err := finalizeImportedPierReadiness(importedPierContext{
		Patp: "~zod",
		Fix:  true,
	}, runtime)
	if err == nil {
		t.Fatal("expected finalizeImportedPierReadiness to return acme fix error")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error chain to include acme failure, got %v", err)
	}
}

func TestPrepareImportedPierEnvironmentPropagatesRealCleanupErrors(t *testing.T) {
	runtime := importerRuntimeWith(func(runtime *importerRuntime) {
		provisionOps := provisionRuntime(t, runtime)
		provisionOps.ShipcreatorCreateUrbitConfigFn = func(string, string) error { return nil }
		provisionOps.ShipcreatorAppendSysConfigPierFn = func(string) error { return nil }

		provisionOps.DeleteContainerFn = func(string) error {
			return errors.New("failed to delete container: permission denied")
		}
		provisionOps.DeleteVolumeFn = func(string) error {
			return errors.New("volume cleanup should be ignored when container cleanup fails")
		}
		provisionOps.CreateVolumeFn = func(string) error { return nil }

		provisionOps.MkdirAllFn = func(string, os.FileMode) error { return nil }
	})

	err := prepareImportedPierEnvironment(importedPierContext{
		Patp:        "~zod",
		CustomDrive: "",
	}, runtime)
	if err == nil {
		t.Fatal("expected prepareImportedPierEnvironment to fail for non-ignorable container delete error")
	}
}

func TestPrepareImportedPierEnvironmentIgnoresNotFoundCleanupErrors(t *testing.T) {
	runtime := importerRuntimeWith(func(runtime *importerRuntime) {
		provisionOps := provisionRuntime(t, runtime)
		provisionOps.ShipcreatorCreateUrbitConfigFn = func(string, string) error { return nil }
		provisionOps.ShipcreatorAppendSysConfigPierFn = func(string) error { return nil }

		provisionOps.DeleteContainerFn = func(string) error {
			return errdefs.NotFound(errors.New("container not found"))
		}
		provisionOps.DeleteVolumeFn = func(string) error {
			return errors.New("no such volume")
		}
		provisionOps.CreateVolumeFn = func(string) error { return nil }

		provisionOps.MkdirAllFn = func(string, os.FileMode) error { return nil }
	})

	if err := prepareImportedPierEnvironment(importedPierContext{
		Patp:        "~zod",
		CustomDrive: "",
	}, runtime); err != nil {
		t.Fatalf("expected prepareImportedPierEnvironment to ignore cleanup not found errors: %v", err)
	}
}

func TestIgnorableCleanupDeleteError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "not found",
			err:  errdefs.NotFound(errors.New("container not found")),
			want: true,
		},
		{
			name: "permission denied",
			err:  errors.New("permission denied"),
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isIgnorableCleanupDeleteError(tc.err); got != tc.want {
				t.Fatalf("isIgnorableCleanupDeleteError(%q) = %v, want %v", tc.err.Error(), got, tc.want)
			}
		})
	}
}
