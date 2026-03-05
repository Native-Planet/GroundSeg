package wireguardstore

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"groundseg/defaults"
)

func TestFileStoreRoundTripsWireguardConfig(t *testing.T) {
	store := FileStore{}
	dir := t.TempDir()
	path := filepath.Join(dir, "settings", "wireguard.json")

	if err := store.EnsureDir(filepath.Dir(path)); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}
	if err := store.Save(path, defaults.DefaultWgConfig()); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := store.Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !reflect.DeepEqual(got, defaults.DefaultWgConfig()) {
		t.Fatalf("unexpected config: got %+v want %+v", got, defaults.DefaultWgConfig())
	}
}

func TestFileStoreLoadReturnsHelpfulErrorForMissingFile(t *testing.T) {
	store := FileStore{}
	_, err := store.Load(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("expected load error for missing file")
	}
	if _, ok := err.(*os.PathError); !ok {
		t.Fatalf("expected *os.PathError, got %T", err)
	}
}
