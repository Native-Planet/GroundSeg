package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"groundseg/structs"
)

func resetUrbitTestState() {
	urbitMutex.Lock()
	UrbitsConfig = make(map[string]structs.UrbitDocker)
	urbitMutex.Unlock()
}

func TestUrbitConfAccessorsAndCopy(t *testing.T) {
	t.Cleanup(resetUrbitTestState)
	urbitMutex.Lock()
	UrbitsConfig["~zod"] = structs.UrbitDocker{PierName: "~zod", MinIOLinked: true}
	urbitMutex.Unlock()

	if !GetMinIOLinkedStatus("~zod") {
		t.Fatalf("expected minio linked status true")
	}
	conf := UrbitConf("~zod")
	if conf.PierName != "~zod" {
		t.Fatalf("unexpected conf: %+v", conf)
	}

	all := UrbitConfAll()
	all["~zod"] = structs.UrbitDocker{PierName: "mutated"}
	if UrbitConf("~zod").PierName != "~zod" {
		t.Fatalf("UrbitConfAll should return a copy, not original map")
	}
}

func TestLoadAndRemoveUrbitConfig(t *testing.T) {
	t.Cleanup(resetUrbitTestState)

	oldBasePath := BasePath()
	SetBasePath(t.TempDir())
	t.Cleanup(func() { SetBasePath(oldBasePath) })

	pier := "~bus"
	path := filepath.Join(BasePath(), "settings", "pier")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	initial := map[string]interface{}{
		"pier_name":    pier,
		"minio_linked": true,
	}
	data, err := json.Marshal(initial)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(path, pier+".json"), data, 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if err := LoadUrbitConfig(pier); err != nil {
		t.Fatalf("LoadUrbitConfig failed: %v", err)
	}
	loaded := UrbitConf(pier)
	if !loaded.MinIOLinked || loaded.PierName != pier {
		t.Fatalf("unexpected loaded config: %+v", loaded)
	}
	if loaded.SnapTime != 60 {
		t.Fatalf("expected snap_time default of 60, got %d", loaded.SnapTime)
	}
	if !loaded.StartramReminder {
		t.Fatalf("expected startram_reminder default to be populated as true")
	}

	if err := RemoveUrbitConfig(pier); err != nil {
		t.Fatalf("RemoveUrbitConfig failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(path, pier+".json")); !os.IsNotExist(err) {
		t.Fatalf("expected config file removal, got err=%v", err)
	}
	if _, ok := UrbitConfAll()[pier]; ok {
		t.Fatalf("expected in-memory config to be removed")
	}
}

func TestUpdateUrbitPersistsMutations(t *testing.T) {
	t.Cleanup(resetUrbitTestState)

	oldBasePath := BasePath()
	SetBasePath(t.TempDir())
	t.Cleanup(func() { SetBasePath(oldBasePath) })

	pier := "~nec"
	urbitMutex.Lock()
	UrbitsConfig[pier] = structs.UrbitDocker{PierName: pier, SnapTime: 30}
	urbitMutex.Unlock()

	err := UpdateUrbit(pier, func(conf *structs.UrbitDocker) error {
		conf.MinIOLinked = true
		conf.UrbitVersion = "file-version"
		return nil
	})
	if err != nil {
		t.Fatalf("UpdateUrbit failed: %v", err)
	}
	updated := UrbitConf(pier)
	if !updated.MinIOLinked || updated.UrbitVersion != "file-version" {
		t.Fatalf("unexpected updated config: %+v", updated)
	}

	file := filepath.Join(BasePath(), "settings", "pier", pier+".json")
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read persisted file failed: %v", err)
	}
	var persisted structs.UrbitDocker
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatalf("unmarshal persisted file failed: %v", err)
	}
	if !persisted.MinIOLinked || persisted.UrbitVersion != "file-version" {
		t.Fatalf("unexpected persisted config: %+v", persisted)
	}
}

func TestUpdateUrbitValidationAndLoadFailure(t *testing.T) {
	t.Cleanup(resetUrbitTestState)

	if err := UpdateUrbit("~zod", nil); err == nil {
		t.Fatalf("expected nil mutate function error")
	}

	oldBasePath := BasePath()
	SetBasePath(t.TempDir())
	t.Cleanup(func() { SetBasePath(oldBasePath) })
	err := UpdateUrbit("~missing", func(*structs.UrbitDocker) error { return nil })
	if err == nil {
		t.Fatalf("expected load error for missing on-disk config")
	}
}
