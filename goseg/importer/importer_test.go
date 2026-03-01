package importer

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"groundseg/auth"
	"groundseg/docker"
	"groundseg/lifecycle"
	"groundseg/structs"

	"github.com/gorilla/mux"
)

func drainImportTransitions() {
	for {
		select {
		case <-docker.ImportShipTransitions():
		default:
			return
		}
	}
}

func readImportTransitions(t *testing.T, count int) []structs.UploadTransition {
	t.Helper()
	events := make([]structs.UploadTransition, 0, count)
	for i := 0; i < count; i++ {
		select {
		case evt := <-docker.ImportShipTransitions():
			events = append(events, evt)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for import transition %d", i+1)
		}
	}
	return events
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

func authorizeTokenIDsForTest(t *testing.T, tokenIDs ...string) {
	t.Helper()
	originalClientManager := auth.ClientManager
	clientManager := auth.NewClientManager()
	for _, tokenID := range tokenIDs {
		clientManager.AddAuthClient(tokenID, &structs.MuConn{Active: false})
	}
	auth.ClientManager = clientManager
	t.Cleanup(func() {
		auth.ClientManager = originalClientManager
	})
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
	failUploadRequest(recorder, http.StatusUnauthorized, "bad token")

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

func TestRunImportPhasesEmitsLifecycleStatuses(t *testing.T) {
	drainImportTransitions()

	err := runImportPhases(
		"archive.tgz",
		"~zod",
		"",
		lifecycle.Step{
			Phase: lifecycle.Phase("creating"),
			Run: func() error {
				return nil
			},
		},
		lifecycle.Step{
			Phase: lifecycle.Phase("booting"),
			Run: func() error {
				return nil
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

	_, err := validateUploadRequest(req)
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

	_, err := validateUploadRequest(req)
	if err == nil {
		t.Fatal("expected validateUploadRequest to fail on token mismatch")
	}
	if !strings.Contains(err.Error(), "upload token does not match upload session") {
		t.Fatalf("unexpected error: %v", err)
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
	authorizeTokenIDsForTest(t, "token-id")

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
	if err := OpenUploadEndpoint(createCmd); err != nil {
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
	if err := OpenUploadEndpoint(updateCmd); err != nil {
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
	authorizeTokenIDsForTest(t, "token-id", "other-token-id")

	endpoint := "0123456789abcdef0123456789abcdef"
	if err := OpenUploadEndpoint(OpenUploadEndpointCmd{
		Endpoint: endpoint,
		Token: structs.WsTokenStruct{
			ID:    "token-id",
			Token: "token-value",
		},
		SelectedDrive: "system-drive",
	}); err != nil {
		t.Fatalf("OpenUploadEndpoint(create) returned error: %v", err)
	}

	err := OpenUploadEndpoint(OpenUploadEndpointCmd{
		Endpoint: endpoint,
		Token: structs.WsTokenStruct{
			ID:    "other-token-id",
			Token: "other-token-value",
		},
		SelectedDrive: "system-drive",
	})
	if err == nil {
		t.Fatal("expected token mismatch error")
	}
	if !strings.Contains(err.Error(), "token mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetUploadSessionWrapsOpenUploadEndpoint(t *testing.T) {
	resetUploadSessionsForTest(t)
	authorizeTokenIDsForTest(t, "token-id")

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
	if err := SetUploadSession(payload); err != nil {
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
	if !allChunksReceived(filename, len(parts)) {
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
