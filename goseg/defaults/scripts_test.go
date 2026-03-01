package defaults

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedScriptsHaveValidBashSyntax(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available in test environment")
	}

	scripts := map[string]string{
		"prep":      PrepScript,
		"start":     StartScript,
		"roll":      RollScript,
		"pack":      PackScript,
		"chop":      ChopScript,
		"meld":      MeldScript,
		"run-llama": RunLlama,
	}
	for name, script := range scripts {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name+".sh")
			if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
				t.Fatalf("write script file: %v", err)
			}
			output, err := exec.Command("bash", "-n", path).CombinedOutput()
			if err != nil {
				t.Fatalf("bash syntax validation failed: %v\n%s", err, string(output))
			}
		})
	}
}

func TestEmbeddedScriptsContainExpectedCommands(t *testing.T) {
	testCases := []struct {
		name    string
		script  string
		command string
	}{
		{name: "prep", script: PrepScript, command: "urbit prep"},
		{name: "start", script: StartScript, command: "trap_urbit"},
		{name: "roll", script: RollScript, command: "urbit roll"},
		{name: "pack", script: PackScript, command: "urbit pack"},
		{name: "chop", script: ChopScript, command: "urbit chop"},
		{name: "meld", script: MeldScript, command: "urbit meld"},
		{name: "run-llama", script: RunLlama, command: "llama_cpp.server"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.HasPrefix(tc.script, "#!/bin/bash") {
				t.Fatalf("%s script missing bash shebang", tc.name)
			}
			if !strings.Contains(tc.script, tc.command) {
				t.Fatalf("%s script missing command %q", tc.name, tc.command)
			}
		})
	}
}

func TestGetBasePathUsesEnvironmentOverride(t *testing.T) {
	t.Setenv("GS_BASE_PATH", "/tmp/custom-groundseg")
	if got := getBasePath(); got != "/tmp/custom-groundseg" {
		t.Fatalf("expected env override base path, got %q", got)
	}
}

func TestGetBasePathUsesDefaultWhenUnset(t *testing.T) {
	t.Setenv("GS_BASE_PATH", "")
	if got := getBasePath(); got != "/opt/nativeplanet/groundseg" {
		t.Fatalf("expected default base path, got %q", got)
	}
}
