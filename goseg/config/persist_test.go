package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileDurablyReplacesContentsAndPreservesMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings", "system.json")
	if err := writeFileDurably(path, []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeFileDurably(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	afterNoOp, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, afterNoOp) {
		t.Fatal("identical contents unexpectedly replaced the config file")
	}
	if err := writeFileDurably(path, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("contents = %q, want %q", data, "new")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("mode = %o, want 640", got)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
		t.Fatalf("config directory entries = %v, want only %q", entries, filepath.Base(path))
	}
}

func TestWriteJSONDurablyCreatesValidIndentedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings", "pier", "zod.json")
	value := map[string]any{"pier_name": "zod", "enabled": true}
	if err := writeJSONDurably(path, value, 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n    \"enabled\": true,\n    \"pier_name\": \"zod\"\n}\n"
	if string(data) != want {
		t.Fatalf("JSON = %q, want %q", data, want)
	}
}
