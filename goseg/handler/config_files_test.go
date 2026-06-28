package handler

import (
	"encoding/json"
	"groundseg/config"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigFileTarget(t *testing.T) {
	base := t.TempDir()
	t.Setenv("GS_BASE_PATH", base)
	oldBasePath := config.BasePath
	t.Cleanup(func() {
		config.BasePath = oldBasePath
	})
	config.BasePath = base
	if err := os.MkdirAll(filepath.Join(base, "settings"), 0o755); err != nil {
		t.Fatalf("failed to create settings dir: %v", err)
	}
	raw, err := json.Marshal(map[string]any{
		"piers": []string{"sampel-palnet"},
		"sessions": map[string]any{
			"authorized":   map[string]any{},
			"unauthorized": map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	if _, err := config.ReplaceConfJSON(raw); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}

	tests := []struct {
		name    string
		file    string
		wantErr bool
		kind    string
		pier    string
	}{
		{name: "system file", file: "system.json", kind: "system"},
		{name: "settings alias", file: "settings.json", kind: "system"},
		{name: "configured pier", file: "pier/sampel-palnet.json", kind: "pier", pier: "sampel-palnet"},
		{name: "hermes yaml", file: "hermes/config.yaml", kind: "hermes-yaml"},
		{name: "hermes env", file: "hermes/.env", kind: "hermes-env"},
		{name: "traversal", file: "../system.json", wantErr: true},
		{name: "nested traversal", file: "pier/../system.json", wantErr: true},
		{name: "nested pier path", file: "pier/sampel-palnet/extra.json", wantErr: true},
		{name: "unconfigured pier", file: "pier/zod.json", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveConfigFileTarget(tt.file)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveConfigFileTarget(%q) returned nil error", tt.file)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveConfigFileTarget(%q) returned error: %v", tt.file, err)
			}
			if got.kind != tt.kind {
				t.Fatalf("kind = %q, want %q", got.kind, tt.kind)
			}
			if got.pier != tt.pier {
				t.Fatalf("pier = %q, want %q", got.pier, tt.pier)
			}
		})
	}
}

func TestValidateHermesEnv(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "basic", raw: "OPENAI_BASE_URL=http://localhost:1234/v1\nOPENAI_API_KEY=\n"},
		{name: "export", raw: "export HERMES_MODEL=local-model\n"},
		{name: "comments", raw: "# local endpoint\nOPENAI_API_KEY=sk-test\n"},
		{name: "empty", raw: "  \n", wantErr: true},
		{name: "invalid key", raw: "1BAD=value\n", wantErr: true},
		{name: "missing separator", raw: "OPENAI_API_KEY\n", wantErr: true},
		{name: "null byte", raw: "OPENAI_API_KEY=x\x00\n", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateHermesEnv([]byte(tt.raw))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateHermesEnv(%q) returned nil error", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateHermesEnv(%q) returned error: %v", tt.raw, err)
			}
			if len(got) == 0 || got[len(got)-1] != '\n' {
				t.Fatalf("validated env should end with newline: %q", string(got))
			}
		})
	}
}

func TestValidateHermesConfigYAML(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "mapping", raw: "model:\n  provider: openrouter\n"},
		{name: "empty", raw: "  \n", wantErr: true},
		{name: "invalid", raw: "model: [", wantErr: true},
		{name: "scalar", raw: "hello", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateHermesConfigYAML([]byte(tt.raw))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateHermesConfigYAML(%q) returned nil error", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateHermesConfigYAML(%q) returned error: %v", tt.raw, err)
			}
			if len(got) == 0 || got[len(got)-1] != '\n' {
				t.Fatalf("validated YAML should end with newline: %q", string(got))
			}
		})
	}
}
