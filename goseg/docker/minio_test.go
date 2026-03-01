package docker

import (
	"errors"
	"groundseg/structs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func testMinIORuntime() minioRuntime {
	rt := newMinIORuntime()
	rt.conf = func() structs.SysConfig { return structs.SysConfig{} }
	rt.osOpen = func(string) (*os.File, error) { return nil, nil }
	rt.createDefaultMcConf = func() error { return nil }
	rt.startContainer = func(string, string) (structs.ContainerState, error) {
		return structs.ContainerState{ActualStatus: "running"}, nil
	}
	rt.updateContainerState = func(string, structs.ContainerState) {}
	rt.loadUrbitConfig = func(string) error { return nil }
	rt.urbitConf = func(string) structs.UrbitDocker { return structs.UrbitDocker{} }
	rt.getLatestContainerInfo = func(string) (map[string]string, error) {
		return map[string]string{"repo": "repo/image", "tag": "latest", "hash": "hash"}, nil
	}
	rt.randRead = func(dst []byte) (int, error) {
		for i := range dst {
			dst[i] = byte(i)
		}
		return len(dst), nil
	}
	rt.setMinIOPassword = func(string, string) error { return nil }
	rt.getContainerRunningStatus = func(string) (string, error) { return "Up", nil }
	rt.sleep = func(time.Duration) {}
	rt.execDockerCommand = func(string, []string) (string, error) { return "ok", nil }
	rt.getMinIOPassword = func(string) (string, error) { return "secret", nil }
	rt.writeFile = func(string, []byte, os.FileMode) error { return nil }
	rt.pollInterval = time.Millisecond
	return rt
}

func TestGetPatpFromMinIOName(t *testing.T) {
	patp, err := getPatpFromMinIOName("minio_~zod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if patp != "~zod" {
		t.Fatalf("unexpected patp: %s", patp)
	}
	if _, err := getPatpFromMinIOName("invalid"); err == nil {
		t.Fatalf("expected invalid minio name error")
	}
}

func TestLoadMCStartsContainerWhenRegistered(t *testing.T) {
	rt := testMinIORuntime()
	rt.conf = func() structs.SysConfig { return structs.SysConfig{WgRegistered: true} }
	rt.osOpen = func(string) (*os.File, error) { return nil, os.ErrNotExist }
	defaultCreated := false
	rt.createDefaultMcConf = func() error {
		defaultCreated = true
		return nil
	}
	var startedName, startedType string
	rt.startContainer = func(name, ctype string) (structs.ContainerState, error) {
		startedName, startedType = name, ctype
		return structs.ContainerState{ActualStatus: "running"}, nil
	}
	updated := false
	rt.updateContainerState = func(name string, state structs.ContainerState) {
		if name == "mc" && state.ActualStatus == "running" {
			updated = true
		}
	}

	if err := loadMC(rt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !defaultCreated {
		t.Fatalf("expected default mc config creation when file missing")
	}
	if startedName != "mc" || startedType != "miniomc" || !updated {
		t.Fatalf("unexpected start/update flow: %s %s updated=%v", startedName, startedType, updated)
	}
}

func TestLoadMinIOsStartsPerPier(t *testing.T) {
	rt := testMinIORuntime()
	rt.conf = func() structs.SysConfig {
		return structs.SysConfig{WgRegistered: true, Piers: []string{"~zod", "~bus"}}
	}
	var started []string
	rt.startContainer = func(name, ctype string) (structs.ContainerState, error) {
		started = append(started, name+":"+ctype)
		if strings.Contains(name, "~bus") {
			return structs.ContainerState{}, errors.New("boom")
		}
		return structs.ContainerState{ActualStatus: "running"}, nil
	}
	var updated []string
	rt.updateContainerState = func(name string, _ structs.ContainerState) {
		updated = append(updated, name)
	}

	if err := loadMinIOs(rt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(started, []string{"minio_~zod:minio", "minio_~bus:minio"}) {
		t.Fatalf("unexpected started containers: %v", started)
	}
	if !reflect.DeepEqual(updated, []string{"minio_~zod"}) {
		t.Fatalf("unexpected updated containers: %v", updated)
	}
}

func TestMinioContainerConfBuildsExpectedConfig(t *testing.T) {
	rt := testMinIORuntime()
	rt.loadUrbitConfig = func(string) error { return nil }
	rt.urbitConf = func(string) structs.UrbitDocker {
		return structs.UrbitDocker{WgURL: "ship.example", WgS3Port: 9000, WgConsolePort: 9001}
	}
	rt.getLatestContainerInfo = func(string) (map[string]string, error) {
		return map[string]string{"repo": "repo/minio", "tag": "latest", "hash": "abcd"}, nil
	}
	rt.randRead = func(dst []byte) (int, error) {
		for i := range dst {
			dst[i] = byte(i)
		}
		return len(dst), nil
	}
	var storedName, storedPwd string
	rt.setMinIOPassword = func(name, pwd string) error {
		storedName, storedPwd = name, pwd
		return nil
	}
	rt.getContainerRunningStatus = func(string) (string, error) { return "Up 3 seconds", nil }
	rt.sleep = func(time.Duration) {}

	containerCfg, hostCfg, err := minioContainerConfWithRuntime(rt, "minio_~zod")
	if err != nil {
		t.Fatalf("minioContainerConf failed: %v", err)
	}
	if containerCfg.Image != "repo/minio:latest@sha256:abcd" {
		t.Fatalf("unexpected image: %s", containerCfg.Image)
	}
	env := strings.Join(containerCfg.Env, " ")
	if !strings.Contains(env, "MINIO_ROOT_USER=~zod") || !strings.Contains(env, "MINIO_DOMAIN=s3.ship.example") {
		t.Fatalf("unexpected env: %v", containerCfg.Env)
	}
	if storedName != "minio_~zod" || len(storedPwd) != 32 {
		t.Fatalf("expected generated password stored, got name=%s pwd=%s", storedName, storedPwd)
	}
	if hostCfg.NetworkMode != "container:wireguard" || len(hostCfg.Mounts) != 1 || hostCfg.Mounts[0].Source != "minio_~zod" {
		t.Fatalf("unexpected host config: %+v", hostCfg)
	}
}

func TestMinioContainerConfInvalidName(t *testing.T) {
	rt := testMinIORuntime()
	_, _, err := minioContainerConfWithRuntime(rt, "broken-name")
	if err == nil {
		t.Fatalf("expected invalid container name error")
	}
}

func TestMCContainerConfPollsUntilWireguardUp(t *testing.T) {
	rt := testMinIORuntime()
	rt.getLatestContainerInfo = func(string) (map[string]string, error) {
		return map[string]string{"repo": "repo/mc", "tag": "latest", "hash": "hash"}, nil
	}
	calls := 0
	rt.getContainerRunningStatus = func(string) (string, error) {
		calls++
		if calls < 2 {
			return "Exited", nil
		}
		return "Up 2 seconds", nil
	}
	sleepCalls := 0
	rt.sleep = func(time.Duration) { sleepCalls++ }

	containerCfg, hostCfg, err := mcContainerConfWithRuntime(rt)
	if err != nil {
		t.Fatalf("mcContainerConf failed: %v", err)
	}
	if containerCfg.Image != "repo/mc:latest@sha256:hash" || hostCfg.NetworkMode != "container:wireguard" {
		t.Fatalf("unexpected mc container config: %+v %+v", containerCfg, hostCfg)
	}
	if sleepCalls != 1 {
		t.Fatalf("expected one poll sleep, got %d", sleepCalls)
	}
}

func TestSetMinIOAdminAccountSuccess(t *testing.T) {
	rt := testMinIORuntime()
	tmpDockerDir := t.TempDir()
	rt.dockerDir = func() string { return tmpDockerDir }

	rt.urbitConf = func(string) structs.UrbitDocker { return structs.UrbitDocker{WgS3Port: 9000} }
	rt.getContainerRunningStatus = func(string) (string, error) { return "Up", nil }
	rt.sleep = func(time.Duration) {}
	rt.getMinIOPassword = func(string) (string, error) { return "secret", nil }
	var commands [][]string
	rt.execDockerCommand = func(_ string, cmd []string) (string, error) {
		commands = append(commands, append([]string(nil), cmd...))
		return "ok", nil
	}
	var scriptPath string
	rt.writeFile = func(path string, _ []byte, _ os.FileMode) error {
		scriptPath = path
		return nil
	}

	if err := setMinIOAdminAccountWithRuntime(rt, "minio_~zod"); err != nil {
		t.Fatalf("setMinIOAdminAccount failed: %v", err)
	}
	if len(commands) != 3 {
		t.Fatalf("expected 3 docker exec commands, got %d", len(commands))
	}
	if commands[0][0] != "mc" || commands[1][1] != "mb" || commands[2][2] != "set-json" {
		t.Fatalf("unexpected command sequence: %v", commands)
	}
	wantPath := filepath.Join(rt.dockerDir(), "minio_~zod", "_data", "policy.json")
	if scriptPath != wantPath {
		t.Fatalf("unexpected policy path: got %s want %s", scriptPath, wantPath)
	}
}

func TestCreateMinIOServiceAccountFlow(t *testing.T) {
	rt := testMinIORuntime()
	rt.randRead = func(dst []byte) (int, error) {
		for i := range dst {
			dst[i] = 0xAA
		}
		return len(dst), nil
	}
	calls := 0
	rt.execDockerCommand = func(_ string, cmd []string) (string, error) {
		calls++
		if calls == 1 {
			return "successfully removed", nil
		}
		return "service account created", nil
	}

	svc, err := createMinIOServiceAccountWithRuntime(rt, "~zod")
	if err != nil {
		t.Fatalf("CreateMinIOServiceAccount failed: %v", err)
	}
	if svc.AccessKey != "urbit_minio" || svc.Alias != "patp_~zod" || svc.User != "~zod" || len(svc.SecretKey) != 40 {
		t.Fatalf("unexpected service account: %+v", svc)
	}
}

func TestCreateMinIOServiceAccountErrors(t *testing.T) {
	rt := testMinIORuntime()
	rt.randRead = func(dst []byte) (int, error) { return len(dst), nil }

	rt.execDockerCommand = func(string, []string) (string, error) {
		return "", errors.New("docker failed")
	}
	if _, err := createMinIOServiceAccountWithRuntime(rt, "~zod"); err == nil {
		t.Fatalf("expected docker failure")
	}

	rt.execDockerCommand = func(string, []string) (string, error) {
		return "nothing removed", nil
	}
	if _, err := createMinIOServiceAccountWithRuntime(rt, "~zod"); err == nil || !strings.Contains(err.Error(), "remove old service account") {
		t.Fatalf("expected failed removal response error, got %v", err)
	}
}
