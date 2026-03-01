package click

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"groundseg/config"
	"groundseg/docker"
	"groundseg/structs"
)

func resetHoonSeams() {
	execDockerCommandForClick = docker.ExecDockerCommand
}

func setUrbitConfigForTest(t *testing.T, patp string, conf structs.UrbitDocker) {
	t.Helper()
	if config.UrbitsConfig == nil {
		config.UrbitsConfig = make(map[string]structs.UrbitDocker)
	}
	old, had := config.UrbitsConfig[patp]
	config.UrbitsConfig[patp] = conf
	t.Cleanup(func() {
		if had {
			config.UrbitsConfig[patp] = old
		} else {
			delete(config.UrbitsConfig, patp)
		}
	})
}

func TestCreateAndDeleteHoonDefaultLocation(t *testing.T) {
	t.Cleanup(resetHoonSeams)

	tmpRoot := t.TempDir()
	oldDockerDir := config.DockerDir
	config.DockerDir = tmpRoot
	t.Cleanup(func() { config.DockerDir = oldDockerDir })

	patp := "~zod"
	setUrbitConfigForTest(t, patp, structs.UrbitDocker{})
	defaultPath := filepath.Join(tmpRoot, patp, "_data")
	if err := os.MkdirAll(defaultPath, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	codeMutex.Lock()
	lusCodes[patp] = structs.ClickLusCode{LusCode: strings.Repeat("x", 27)}
	codeMutex.Unlock()

	if err := createHoon(patp, "sample", "hoon-body"); err != nil {
		t.Fatalf("createHoon failed: %v", err)
	}

	target := filepath.Join(defaultPath, "sample.hoon")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("expected hoon file: %v", err)
	}
	if string(data) != "hoon-body" {
		t.Fatalf("unexpected content: %s", string(data))
	}

	codeMutex.Lock()
	_, exists := lusCodes[patp]
	codeMutex.Unlock()
	if exists {
		t.Fatalf("expected createHoon to clear +code cache")
	}

	deleteHoon(patp, "sample")
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected hoon file to be deleted, err=%v", err)
	}
}

func TestCreateHoonUsesCustomPierLocation(t *testing.T) {
	t.Cleanup(resetHoonSeams)

	tmpRoot := t.TempDir()
	oldDockerDir := config.DockerDir
	config.DockerDir = filepath.Join(tmpRoot, "docker")
	t.Cleanup(func() { config.DockerDir = oldDockerDir })

	customPath := filepath.Join(tmpRoot, "custom-pier")
	if err := os.MkdirAll(customPath, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	patp := "~bus"
	setUrbitConfigForTest(t, patp, structs.UrbitDocker{CustomPierLocation: customPath})

	if err := createHoon(patp, "custom", "value"); err != nil {
		t.Fatalf("createHoon failed: %v", err)
	}

	target := filepath.Join(customPath, "custom.hoon")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected file in custom location: %v", err)
	}
}

func TestClickExecBuildsCommand(t *testing.T) {
	t.Cleanup(resetHoonSeams)

	var gotPatp string
	var gotCmd []string
	execDockerCommandForClick = func(patp string, cmd []string) (string, error) {
		gotPatp = patp
		gotCmd = append([]string(nil), cmd...)
		return "ok", nil
	}

	res, err := clickExec("~nec", "myfile", "/sur/hood/hoon")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != "ok" {
		t.Fatalf("unexpected response: %s", res)
	}
	if gotPatp != "~nec" {
		t.Fatalf("unexpected patp: %s", gotPatp)
	}
	want := []string{"click", "-b", "urbit", "-kp", "-i", "myfile.hoon", "~nec", "/sur/hood/hoon"}
	if !reflect.DeepEqual(gotCmd, want) {
		t.Fatalf("unexpected command: got %v want %v", gotCmd, want)
	}
}

func TestClickExecReturnsError(t *testing.T) {
	t.Cleanup(resetHoonSeams)
	execDockerCommandForClick = func(string, []string) (string, error) {
		return "", errors.New("docker failed")
	}

	_, err := clickExec("~mar", "file", "")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestJoinGapAndStorageAction(t *testing.T) {
	if got := joinGap([]string{"a", "b", "c"}); got != "a  b  c" {
		t.Fatalf("unexpected joinGap output: %s", got)
	}

	action := storageAction("%unlink", "https://s3.example")
	if !strings.Contains(action, "%storage-action") || !strings.Contains(action, "https://s3.example") {
		t.Fatalf("unexpected storageAction output: %s", action)
	}
}
