package container

import (
	"groundseg/config"
	"groundseg/structs"
	"os"
	"strings"
	"testing"
)

func TestLoadMCWithRuntimeRequiresConfFn(t *testing.T) {
	err := LoadMCWithRuntime(MinioRuntime{})
	if err == nil {
		t.Fatalf("expected missing runtime error")
	}
	if !strings.Contains(err.Error(), "minio runtime requires settings snapshot callbacks") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadMinIOsWithRuntimeRequiresConfFn(t *testing.T) {
	err := LoadMinIOsWithRuntime(MinioRuntime{})
	if err == nil {
		t.Fatalf("expected missing runtime error")
	}
	if !strings.Contains(err.Error(), "minio runtime requires settings snapshot callbacks") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadLlamaWithRuntimeRequiresConfFn(t *testing.T) {
	err := LoadLlamaWithRuntime(LlamaRuntime{})
	if err == nil {
		t.Fatalf("expected missing runtime error")
	}
	if !strings.Contains(err.Error(), "llama runtime requires penpai settings snapshot callback") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLlamaContainerConfWithRuntimeRequiresConfFn(t *testing.T) {
	_, _, err := LlamaContainerConfWithRuntime(LlamaRuntime{})
	if err == nil {
		t.Fatalf("expected missing runtime error")
	}
	if !strings.Contains(err.Error(), "llama runtime requires settings snapshot callbacks") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMinioContainerConfWithRuntimeRejectsNilGetLatestContainerInfo(t *testing.T) {
	rt := MinioRuntime{}
	_, _, err := MinioContainerConfWithRuntime(rt, "minio")
	if err == nil {
		t.Fatalf("expected missing image runtime error")
	}
	if !strings.Contains(err.Error(), "minio runtime not fully configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLlamaContainerConfWithRuntimeRejectsMissingRuntimeConfig(t *testing.T) {
	_, _, err := LlamaContainerConfWithRuntime(LlamaRuntime{
		PenpaiSettingsSnapshotFn: func() config.PenpaiSettings {
			return config.PenpaiSettings{
				ActiveModel: "phi.gguf",
				Models: []structs.Penpai{
					{ModelName: "phi.gguf", ModelUrl: "https://example.invalid/model.gguf"},
				},
			}
		},
		ShipSettingsSnapshotFn: func() config.ShipSettings {
			return config.ShipSettings{}
		},
		VolumeDirFn:       func() string { return "/tmp/volumes" },
		DockerDirFn:       func() string { return "/tmp/volumes" },
		WriteFileFn:       func(string, []byte, os.FileMode) error { return nil },
		AddOrGetNetworkFn: func(string) (string, error) { return "llama", nil },
	})
	if err == nil {
		t.Fatalf("expected missing urbits config runtime error")
	}
	if !strings.Contains(err.Error(), "missing urbits config runtime") {
		t.Fatalf("unexpected error: %v", err)
	}
}
