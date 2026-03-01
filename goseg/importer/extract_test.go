package importer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"groundseg/docker"
	"groundseg/structs"
)

func collectImportTransitions(timeout time.Duration) []structs.UploadTransition {
	events := []structs.UploadTransition{}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case evt := <-docker.ImportShipTransitions():
			events = append(events, evt)
		case <-timer.C:
			return events
		}
	}
}

func hasExtracted100(events []structs.UploadTransition) bool {
	for _, evt := range events {
		if evt.Type == "extracted" && evt.Value == 100 {
			return true
		}
	}
	return false
}

func createZipArchive(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	for name, content := range map[string]string{
		"ship/file.txt":        "hello from zip",
		"__MACOSX/ignored.txt": "ignore",
		".DS_Store":            "ignore",
		"ship/conn.sock":       "ignore",
		"ship/nested/keep.txt": "keep me",
	} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("failed to add %s to zip: %v", name, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write %s to zip: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
}

func createTarArchive(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create tar file: %v", err)
	}
	defer file.Close()

	writer := tar.NewWriter(file)
	defer writer.Close()

	entries := []struct {
		name     string
		content  string
		typeFlag byte
	}{
		{name: "ship/", typeFlag: tar.TypeDir},
		{name: "ship/file.txt", content: "hello from tar", typeFlag: tar.TypeReg},
		{name: "ship/nested/", typeFlag: tar.TypeDir},
		{name: "ship/nested/keep.txt", content: "keep me", typeFlag: tar.TypeReg},
		{name: "__MACOSX/ignored.txt", content: "ignore", typeFlag: tar.TypeReg},
		{name: ".DS_Store", content: "ignore", typeFlag: tar.TypeReg},
		{name: "ship/conn.sock", content: "ignore", typeFlag: tar.TypeReg},
	}

	for _, entry := range entries {
		header := &tar.Header{
			Name:     entry.name,
			Mode:     0o755,
			Typeflag: entry.typeFlag,
		}
		if entry.typeFlag == tar.TypeReg {
			header.Size = int64(len(entry.content))
			header.Mode = 0o644
		}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatalf("failed to write tar header for %s: %v", entry.name, err)
		}
		if entry.typeFlag == tar.TypeReg {
			if _, err := writer.Write([]byte(entry.content)); err != nil {
				t.Fatalf("failed to write tar payload for %s: %v", entry.name, err)
			}
		}
	}
}

func createTarGzArchive(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create tar.gz file: %v", err)
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzWriter)

	entries := []struct {
		name     string
		content  string
		typeFlag byte
	}{
		{name: "ship/", typeFlag: tar.TypeDir},
		{name: "ship/file.txt", content: "hello from targz", typeFlag: tar.TypeReg},
		{name: "ship/nested/keep.txt", content: "keep me", typeFlag: tar.TypeReg},
		{name: "__MACOSX/ignored.txt", content: "ignore", typeFlag: tar.TypeReg},
		{name: ".DS_Store", content: "ignore", typeFlag: tar.TypeReg},
		{name: "ship/conn.sock", content: "ignore", typeFlag: tar.TypeReg},
	}

	for _, entry := range entries {
		header := &tar.Header{
			Name:     entry.name,
			Mode:     0o755,
			Typeflag: entry.typeFlag,
		}
		if entry.typeFlag == tar.TypeReg {
			header.Size = int64(len(entry.content))
			header.Mode = 0o644
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("failed to write tar.gz header for %s: %v", entry.name, err)
		}
		if entry.typeFlag == tar.TypeReg {
			if _, err := tarWriter.Write([]byte(entry.content)); err != nil {
				t.Fatalf("failed to write tar.gz payload for %s: %v", entry.name, err)
			}
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
}

func TestCheckExtension(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{filename: "ship.tar.gz", want: ".tar.gz"},
		{filename: "SHIP.TAR.GZ", want: ".gz"},
		{filename: "ship.zip", want: ".zip"},
		{filename: "ship", want: ""},
	}

	for _, tt := range tests {
		if got := checkExtension(tt.filename); got != tt.want {
			t.Fatalf("checkExtension(%q) = %q, want %q", tt.filename, got, tt.want)
		}
	}
}

func TestExtractZipExtractsFilesAndSkipsArtifacts(t *testing.T) {
	drainImportTransitions()

	tempDir := t.TempDir()
	src := filepath.Join(tempDir, "archive.zip")
	dest := filepath.Join(tempDir, "out")
	createZipArchive(t, src)

	if err := extractZip(src, dest); err != nil {
		t.Fatalf("extractZip returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dest, "ship", "file.txt"))
	if err != nil {
		t.Fatalf("expected extracted file: %v", err)
	}
	if string(data) != "hello from zip" {
		t.Fatalf("unexpected extracted content: %q", string(data))
	}

	for _, skipped := range []string{
		filepath.Join(dest, "__MACOSX", "ignored.txt"),
		filepath.Join(dest, ".DS_Store"),
		filepath.Join(dest, "ship", "conn.sock"),
	} {
		if _, err := os.Stat(skipped); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be skipped, stat err=%v", skipped, err)
		}
	}

	events := collectImportTransitions(200 * time.Millisecond)
	if !hasExtracted100(events) {
		t.Fatalf("expected extracted=100 transition, got %+v", events)
	}
}

func TestExtractZipRejectsInvalidArchive(t *testing.T) {
	src := filepath.Join(t.TempDir(), "bad.zip")
	if err := os.WriteFile(src, []byte("not a zip"), 0o644); err != nil {
		t.Fatalf("failed to write invalid zip input: %v", err)
	}

	if err := extractZip(src, t.TempDir()); err == nil {
		t.Fatal("expected extractZip to fail for invalid archive")
	}
}

func TestExtractTarExtractsFilesAndSkipsArtifacts(t *testing.T) {
	drainImportTransitions()

	tempDir := t.TempDir()
	src := filepath.Join(tempDir, "archive.tar")
	dest := filepath.Join(tempDir, "out")
	createTarArchive(t, src)

	if err := extractTar(src, dest); err != nil {
		t.Fatalf("extractTar returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dest, "ship", "file.txt"))
	if err != nil {
		t.Fatalf("expected extracted file: %v", err)
	}
	if string(data) != "hello from tar" {
		t.Fatalf("unexpected extracted content: %q", string(data))
	}

	for _, skipped := range []string{
		filepath.Join(dest, "__MACOSX", "ignored.txt"),
		filepath.Join(dest, ".DS_Store"),
		filepath.Join(dest, "ship", "conn.sock"),
	} {
		if _, err := os.Stat(skipped); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be skipped, stat err=%v", skipped, err)
		}
	}

	events := collectImportTransitions(200 * time.Millisecond)
	if !hasExtracted100(events) {
		t.Fatalf("expected extracted=100 transition, got %+v", events)
	}
}

func TestExtractTarRejectsInvalidArchive(t *testing.T) {
	src := filepath.Join(t.TempDir(), "bad.tar")
	if err := os.WriteFile(src, []byte("not a tar"), 0o644); err != nil {
		t.Fatalf("failed to write invalid tar input: %v", err)
	}

	if err := extractTar(src, t.TempDir()); err == nil {
		t.Fatal("expected extractTar to fail for invalid archive")
	}
}

func TestExtractTarRejectsMissingSource(t *testing.T) {
	src := filepath.Join(t.TempDir(), "missing.tar")
	if err := extractTar(src, t.TempDir()); err == nil {
		t.Fatal("expected extractTar to fail for missing archive source")
	}
}

func TestExtractTarGzExtractsFilesAndSkipsArtifacts(t *testing.T) {
	drainImportTransitions()

	tempDir := t.TempDir()
	src := filepath.Join(tempDir, "archive.tar.gz")
	dest := filepath.Join(tempDir, "out")
	createTarGzArchive(t, src)

	if err := extractTarGz(src, dest); err != nil {
		t.Fatalf("extractTarGz returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dest, "ship", "file.txt"))
	if err != nil {
		t.Fatalf("expected extracted file: %v", err)
	}
	if string(data) != "hello from targz" {
		t.Fatalf("unexpected extracted content: %q", string(data))
	}

	for _, skipped := range []string{
		filepath.Join(dest, "__MACOSX", "ignored.txt"),
		filepath.Join(dest, ".DS_Store"),
		filepath.Join(dest, "ship", "conn.sock"),
	} {
		if _, err := os.Stat(skipped); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be skipped, stat err=%v", skipped, err)
		}
	}

	events := collectImportTransitions(200 * time.Millisecond)
	if !hasExtracted100(events) {
		t.Fatalf("expected extracted=100 transition, got %+v", events)
	}
}

func TestExtractTarGzRejectsInvalidArchive(t *testing.T) {
	src := filepath.Join(t.TempDir(), "bad.tar.gz")
	if err := os.WriteFile(src, []byte("not a gz stream"), 0o644); err != nil {
		t.Fatalf("failed to write invalid tar.gz input: %v", err)
	}

	if err := extractTarGz(src, t.TempDir()); err == nil {
		t.Fatal("expected extractTarGz to fail for invalid archive")
	}
}
