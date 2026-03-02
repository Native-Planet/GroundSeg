package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"groundseg/config"
	"groundseg/structs"
)

func TestJoinGap(t *testing.T) {
	if got := JoinGap([]string{"a", "b", "c"}); got != "a  b  c" {
		t.Fatalf("unexpected join output: %q", got)
	}
}

func TestCreateAndDeleteHoonRespectCustomPierLocation(t *testing.T) {
	originalDockerDir := config.DockerDir()
	originalUrbitsConfig := config.UrbitsConfig
	customLocation := t.TempDir()
	patp := "~zod"
	file := "test-hoon"
	config.SetDockerDir(t.TempDir() + "/")
	config.UrbitsConfig = make(map[string]structs.UrbitDocker)
	config.UrbitsConfig[patp] = structs.UrbitDocker{
		UrbitWebConfig: structs.UrbitWebConfig{
			CustomPierLocation: customLocation,
		},
	}
	t.Cleanup(func() {
		config.SetDockerDir(originalDockerDir)
		config.UrbitsConfig = originalUrbitsConfig
	})

	if err := CreateHoon(patp, file, "(foo)"); err != nil {
		t.Fatalf("CreateHoon failed: %v", err)
	}
	hoonPath := filepath.Join(customLocation, file+".hoon")
	if _, err := os.Stat(hoonPath); err != nil {
		t.Fatalf("expected hoon file at %q: %v", hoonPath, err)
	}

	DeleteHoon(patp, file)
	if _, err := os.Stat(hoonPath); err == nil {
		t.Fatalf("expected hoon file to be deleted")
	}
}

func TestClickExecBuildsExpectedCommand(t *testing.T) {
	originalExec := execDockerCommandFn
	t.Cleanup(func() {
		execDockerCommandFn = originalExec
	})

	var gotPatp string
	var gotCommand []string
	execDockerCommandFn = func(patp string, cmd []string) (string, error) {
		gotPatp = patp
		gotCommand = append([]string{}, cmd...)
		return "ok", nil
	}

	response, err := ClickExec("~zod", "test-file", "desk/source")
	if err != nil {
		t.Fatalf("ClickExec failed: %v", err)
	}
	if response != "ok" {
		t.Fatalf("unexpected response: %q", response)
	}
	if gotPatp != "~zod" {
		t.Fatalf("unexpected patp: %q", gotPatp)
	}
	want := []string{"click", "-b", "urbit", "-kp", "-i", "test-file.hoon", "~zod", "desk/source"}
	for i, part := range want {
		if gotCommand[i] != part {
			t.Fatalf("unexpected command at %d: got %q, want %q (full=%q)", i, gotCommand[i], part, gotCommand)
		}
	}
}

func TestExecuteCommandWithLusInvalidationClearsLusAndDeletesHoon(t *testing.T) {
	originalDockerDir := config.DockerDir()
	originalUrbitsConfig := config.UrbitsConfig
	customLocation := t.TempDir()
	patp := "~zod"
	file := "clear-hoon"
	wasClearCalled := false
	config.SetDockerDir(t.TempDir() + "/")
	config.UrbitsConfig = make(map[string]structs.UrbitDocker)
	config.UrbitsConfig[patp] = structs.UrbitDocker{
		UrbitWebConfig: structs.UrbitWebConfig{
			CustomPierLocation: customLocation,
		},
	}
	t.Cleanup(func() {
		config.SetDockerDir(originalDockerDir)
		config.UrbitsConfig = originalUrbitsConfig
	})

	originalExec := execDockerCommandFn
	t.Cleanup(func() {
		execDockerCommandFn = originalExec
	})
	execDockerCommandFn = func(_ string, _ []string) (string, error) {
		return "some event [0 %avow 0 %noun %success]", nil
	}

	response, err := ExecuteCommandWithLusInvalidation(patp, file, "(+ 2 2)", "", "success", "test op", func(got string) {
		wasClearCalled = true
		if got != patp {
			t.Fatalf("unexpected clearLusCode patp: %q", got)
		}
	})
	if err != nil {
		t.Fatalf("ExecuteCommandWithLusInvalidation failed: %v", err)
	}
	if response != "some event [0 %avow 0 %noun %success]" {
		t.Fatalf("unexpected response: %q", response)
	}
	if !wasClearCalled {
		t.Fatal("expected clearLusCode callback to be called")
	}

	hoonPath := filepath.Join(customLocation, file+".hoon")
	if _, err := os.Stat(hoonPath); err == nil {
		t.Fatalf("expected hoon file removed after execution")
	}
}

func TestExecuteCommandWithLusInvalidationReturnsFailureOnMissingSuccessToken(t *testing.T) {
	originalDockerDir := config.DockerDir()
	originalUrbitsConfig := config.UrbitsConfig
	config.SetDockerDir(t.TempDir() + "/")
	config.UrbitsConfig = make(map[string]structs.UrbitDocker)
	config.UrbitsConfig["~zod"] = structs.UrbitDocker{
		UrbitWebConfig: structs.UrbitWebConfig{
			CustomPierLocation: t.TempDir(),
		},
	}
	t.Cleanup(func() {
		config.SetDockerDir(originalDockerDir)
		config.UrbitsConfig = originalUrbitsConfig
	})

	originalExec := execDockerCommandFn
	t.Cleanup(func() {
		execDockerCommandFn = originalExec
	})
	execDockerCommandFn = func(_ string, _ []string) (string, error) {
		return "no-success-marker", nil
	}

	_, err := ExecuteCommand("~zod", "poke", "(+ 1 1)", "", "success", "poke op")
	if err == nil {
		t.Fatal("expected failed poke error")
	}
	if err == nil || !strings.Contains(err.Error(), "poke") {
		t.Fatalf("expected failed poke error, got: %v", err)
	}
}
