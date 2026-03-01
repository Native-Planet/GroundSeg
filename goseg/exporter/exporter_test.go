package exporter

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"groundseg/config"
	"groundseg/defaults"
	"groundseg/docker"
	"groundseg/structs"

	"github.com/gorilla/mux"
)

func resetExporterState() {
	exportMu.Lock()
	whitelist = make(map[string]structs.WsTokenStruct)
	exportMu.Unlock()
	mkdirAllForExporter = os.MkdirAll
	getStoragePathForExporter = config.GetStoragePath
	createTempForExporter = os.CreateTemp
	removeForExporter = os.Remove
	walkForExporter = filepath.Walk
	dockerDataForExporter = defaults.DockerData
	urbitConfForExporter = config.UrbitConf
	publishUrbitTransitionForExporter = docker.PublishUrbitTransition
	getShipStatusForExporter = docker.GetShipStatus
}

func TestInitializeAndWhitelist(t *testing.T) {
	t.Cleanup(resetExporterState)
	targetDir := filepath.Join(t.TempDir(), "exports")
	getStoragePathForExporter = func(string) string { return targetDir }

	if err := Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if exportDir != targetDir {
		t.Fatalf("unexpected exportDir: %s", exportDir)
	}
	if _, err := os.Stat(targetDir); err != nil {
		t.Fatalf("expected export directory to exist: %v", err)
	}

	token := structs.WsTokenStruct{ID: "1", Token: "abc"}
	if err := WhitelistContainer("minio_~zod", token); err != nil {
		t.Fatalf("WhitelistContainer failed: %v", err)
	}
	if err := RemoveContainerFromWhitelist("minio_~zod"); err != nil {
		t.Fatalf("RemoveContainerFromWhitelist failed: %v", err)
	}
}

func TestExportHandlerOptionsAndRejects(t *testing.T) {
	t.Cleanup(resetExporterState)

	req := httptest.NewRequest(http.MethodOptions, "/export/minio_~zod", nil)
	req = mux.SetURLVars(req, map[string]string{"container": "minio_~zod"})
	rr := httptest.NewRecorder()
	ExportHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected OPTIONS 200, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/export/minio_~zod", bytes.NewBufferString(`{"id":"x","token":"y"}`))
	req = mux.SetURLVars(req, map[string]string{"container": "minio_~zod"})
	rr = httptest.NewRecorder()
	ExportHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected non-whitelisted request to fail, got %d", rr.Code)
	}
}

func TestExportHandlerInvalidTokenCleansUpWhitelist(t *testing.T) {
	t.Cleanup(resetExporterState)
	exportDir = t.TempDir()
	container := "minio_~zod"
	token := structs.WsTokenStruct{ID: "ok", Token: "ok"}
	if err := WhitelistContainer(container, token); err != nil {
		t.Fatalf("whitelist failed: %v", err)
	}

	transitionCalls := 0
	publishUrbitTransitionForExporter = func(structs.UrbitTransition) {
		transitionCalls++
	}

	badToken, _ := json.Marshal(structs.WsTokenStruct{ID: "bad", Token: "bad"})
	req := httptest.NewRequest(http.MethodPost, "/export/"+container, bytes.NewReader(badToken))
	req = mux.SetURLVars(req, map[string]string{"container": container})
	rr := httptest.NewRecorder()
	ExportHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected token mismatch status 400, got %d", rr.Code)
	}

	exportMu.Lock()
	_, exists := whitelist[container]
	exportMu.Unlock()
	if exists {
		t.Fatalf("expected whitelist cleanup on failure")
	}
	if transitionCalls < 2 {
		t.Fatalf("expected cleanup transitions to be published, calls=%d", transitionCalls)
	}
}

func TestExportHandlerMinioSuccessStreamsZip(t *testing.T) {
	t.Cleanup(resetExporterState)
	exportDir = t.TempDir()
	volumesDir := t.TempDir()
	dockerDataForExporter = func(string) string { return volumesDir }
	urbitConfForExporter = func(string) structs.UrbitDocker { return structs.UrbitDocker{} }
	publishUrbitTransitionForExporter = func(structs.UrbitTransition) {}

	container := "minio_~zod"
	dataDir := filepath.Join(volumesDir, container, "_data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "conn.sock"), []byte("socket"), 0o644); err != nil {
		t.Fatalf("write conn.sock failed: %v", err)
	}

	token := structs.WsTokenStruct{ID: "1", Token: "t"}
	if err := WhitelistContainer(container, token); err != nil {
		t.Fatalf("whitelist failed: %v", err)
	}
	body, _ := json.Marshal(token)
	req := httptest.NewRequest(http.MethodPost, "/export/"+container, bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"container": container})
	rr := httptest.NewRecorder()
	ExportHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected success status, got %d body=%s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("expected zip content type, got %q", rr.Header().Get("Content-Type"))
	}

	payload := rr.Body.Bytes()
	zr, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatalf("zip parse failed: %v", err)
	}
	if len(zr.File) != 1 || zr.File[0].Name != "file.txt" {
		t.Fatalf("expected only exported file.txt in archive, got %+v", zr.File)
	}

	exportMu.Lock()
	_, exists := whitelist[container]
	exportMu.Unlock()
	if exists {
		t.Fatalf("expected whitelist cleanup after success")
	}
}
