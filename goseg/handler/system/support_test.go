package system

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"groundseg/logger"
)

func TestCopyFileCopiesSourceContent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("groundseg-support"), 0o644); err != nil {
		t.Fatalf("write src file: %v", err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile returned error: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst file: %v", err)
	}
	if string(data) != "groundseg-support" {
		t.Fatalf("unexpected copied content: %q", string(data))
	}
}

func TestZipDirCreatesArchiveWithRelativePaths(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "nested")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir nested dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "root.txt"), []byte("root"), 0o644); err != nil {
		t.Fatalf("write root file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "leaf.txt"), []byte("leaf"), 0o644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	zipPath := filepath.Join(t.TempDir(), "report.zip")
	if err := zipDir(dir, zipPath); err != nil {
		t.Fatalf("zipDir returned error: %v", err)
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer reader.Close()

	var names []string
	for _, file := range reader.File {
		names = append(names, file.Name)
	}
	if !slices.Contains(names, "root.txt") {
		t.Fatalf("zip missing root file; got %v", names)
	}
	if !slices.Contains(names, "nested/leaf.txt") {
		t.Fatalf("zip missing nested file; got %v", names)
	}
}

func TestZipDirReturnsErrorForMissingDirectory(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "missing.zip")
	if err := zipDir(filepath.Join(t.TempDir(), "does-not-exist"), zipPath); err == nil {
		t.Fatal("expected zipDir to fail for missing source directory")
	}
}

func TestSanitizeJSONRemovesRequestedKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "system.json")
	content := map[string]any{
		"sessions": "secret",
		"salt":     "remove-me",
		"keep":     "value",
	}
	blob, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshal content: %v", err)
	}
	if err := os.WriteFile(path, blob, 0o644); err != nil {
		t.Fatalf("write json file: %v", err)
	}

	if err := sanitizeJSON(path, "sessions", "salt"); err != nil {
		t.Fatalf("sanitizeJSON returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read sanitized json: %v", err)
	}
	var sanitized map[string]any
	if err := json.Unmarshal(data, &sanitized); err != nil {
		t.Fatalf("unmarshal sanitized json: %v", err)
	}
	if _, exists := sanitized["sessions"]; exists {
		t.Fatalf("sessions key should be removed, got %+v", sanitized)
	}
	if _, exists := sanitized["salt"]; exists {
		t.Fatalf("salt key should be removed, got %+v", sanitized)
	}
	if sanitized["keep"] != "value" {
		t.Fatalf("keep key changed unexpectedly: %+v", sanitized)
	}
}

func TestSendBugReportIncludesServerBodyOnFailure(t *testing.T) {
	var gotPayload []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		gotPayload, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read multipart body: %v", err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	t.Cleanup(srv.Close)

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "report.zip")
	if err := os.WriteFile(zipPath, []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write bug report zip: %v", err)
	}

	err := sendBugReportWithEndpoint(zipPath, "test@example.com", "desc", srv.URL)
	if err == nil {
		t.Fatal("expected sendBugReport to return error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected status in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected response body in error, got: %v", err)
	}
	if len(gotPayload) == 0 {
		t.Fatal("expected request body")
	}
}

func TestLastTwoLogsReturnsNewestByDateAndPart(t *testing.T) {
	originalLogPath := logger.LogPath
	t.Cleanup(func() {
		logger.LogPath = originalLogPath
	})

	logDir := t.TempDir()
	logger.LogPath = logDir
	files := []string{
		"2026-01-part-2.log",
		"2026-02-part-1.log",
		"2026-02-part-3.log",
		"not-a-log.txt",
		"broken-part-name.log",
	}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(logDir, name), []byte(name), 0o644); err != nil {
			t.Fatalf("write log %s: %v", name, err)
		}
	}

	got := lastTwoLogs()
	if len(got) != 2 {
		t.Fatalf("expected two logs, got %d (%v)", len(got), got)
	}
	wantFirst := filepath.Join(logDir, "2026-02-part-3.log")
	wantSecond := filepath.Join(logDir, "2026-02-part-1.log")
	if got[0] != wantFirst || got[1] != wantSecond {
		t.Fatalf("unexpected log order: got %v want [%s %s]", got, wantFirst, wantSecond)
	}
}
