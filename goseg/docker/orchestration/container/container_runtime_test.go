package container

import (
	"os"
	"strings"
	"testing"

	"groundseg/structs"
)

func TestLoadMCWithRuntimeRequiresConfFn(t *testing.T) {
	err := LoadMCWithRuntime(MinioRuntime{})
	if err == nil {
		t.Fatalf("expected missing runtime error")
	}
	if !strings.Contains(err.Error(), "minio runtime requires ConfFn") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadMinIOsWithRuntimeRequiresConfFn(t *testing.T) {
	err := LoadMinIOsWithRuntime(MinioRuntime{})
	if err == nil {
		t.Fatalf("expected missing runtime error")
	}
	if !strings.Contains(err.Error(), "minio runtime requires ConfFn") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadLlamaWithRuntimeRequiresConfFn(t *testing.T) {
	err := LoadLlamaWithRuntime(LlamaRuntime{})
	if err == nil {
		t.Fatalf("expected missing runtime error")
	}
	if !strings.Contains(err.Error(), "llama runtime requires ConfFn") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLlamaContainerConfWithRuntimeRequiresConfFn(t *testing.T) {
	_, _, err := LlamaContainerConfWithRuntime(LlamaRuntime{})
	if err == nil {
		t.Fatalf("expected missing runtime error")
	}
	if !strings.Contains(err.Error(), "llama runtime requires ConfFn") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMinioContainerConfWithRuntimeRejectsNilGetLatestContainerInfo(t *testing.T) {
	rt := MinioRuntime{
		ConfFn: func() structs.SysConfig { return structs.SysConfig{} },
	}
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
		ConfFn: func() structs.SysConfig {
			return structs.SysConfig{
				PenpaiActive: "phi.gguf",
				PenpaiModels: []structs.Penpai{
					{ModelName: "phi.gguf", ModelUrl: "https://example.invalid/model.gguf"},
				},
			}
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
