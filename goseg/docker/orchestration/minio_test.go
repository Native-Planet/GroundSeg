package orchestration

import (
	"errors"
	"groundseg/config"
	"groundseg/docker/registry"
	"groundseg/structs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func testMinIORuntime() dockerRuntime {
	rt := newDockerRuntime()
	rt.contextOps = RuntimeContextOps{
		BasePathFn:  func() string { return "/tmp" },
		DockerDirFn: func() string { return "/tmp/docker" },
	}
	rt.fileOps = RuntimeFileOps{
		OpenFn:      func(string) (*os.File, error) { return nil, nil },
		ReadFileFn:  func(string) ([]byte, error) { return nil, os.ErrNotExist },
		WriteFileFn: func(string, []byte, os.FileMode) error { return nil },
		MkdirAllFn:  func(string, os.FileMode) error { return nil },
	}
	rt.containerOps = RuntimeContainerOps{
		StartContainerFn: func(string, string) (structs.ContainerState, error) {
			return structs.ContainerState{ActualStatus: "running"}, nil
		},
		UpdateContainerStateFn:      func(string, structs.ContainerState) {},
		GetContainerRunningStatusFn: func(string) (string, error) { return "Up", nil },
	}
	rt.imageOps = RuntimeImageOps{
		GetLatestContainerInfoFn: func(string) (registry.ImageDescriptor, error) {
			return registry.ImageDescriptor{Repo: "repo/image", Tag: "latest", Hash: "hash"}, nil
		},
	}
	rt.configOps = RuntimeSnapshotOps{
		StartramSettingsSnapshotFn: func() config.StartramSettings { return config.StartramSettings{} },
		ShipSettingsSnapshotFn:     func() config.ShipSettings { return config.ShipSettings{} },
		PenpaiSettingsSnapshotFn:   func() config.PenpaiSettings { return config.PenpaiSettings{} },
	}
	rt.urbitOps = RuntimeUrbitOps{
		LoadUrbitConfigFn: func(string) error { return nil },
		UrbitConfFn:       func(string) structs.UrbitDocker { return structs.UrbitDocker{} },
		UpdateUrbitFn:     func(string, func(*structs.UrbitDocker) error) error { return nil },
	}
	rt.minioOps = RuntimeMinioOps{
		CreateDefaultMcConfFn: func() error { return nil },
		SetMinIOPasswordFn:    func(string, string) error { return nil },
		GetMinIOPasswordFn:    func(string) (string, error) { return "secret", nil },
	}
	rt.commandOps = RuntimeCommandOps{
		RandReadFn: func(dst []byte) (int, error) {
			for i := range dst {
				dst[i] = byte(i)
			}
			return len(dst), nil
		},
		ExecDockerCommandFn:     func(string, []string) (string, error) { return "ok", nil },
		ExecDockerCommandExitFn: func(string, []string) (string, int, error) { return "ok", 0, nil },
		CopyFileToVolumeFn:      func(string, string, string, string, volumeWriterImageSelector) error { return nil },
	}
	rt.volumeOps = RuntimeVolumeOps{
		VolumeExistsFn: func(string) (bool, error) { return false, nil },
		CreateVolumeFn: func(string) error { return nil },
	}
	rt.timerOps = RuntimeTimerOps{
		SleepFn:        func(time.Duration) {},
		PollIntervalFn: func() time.Duration { return 500 * time.Millisecond },
	}

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
	rt.configOps = RuntimeSnapshotOps{
		StartramSettingsSnapshotFn: func() config.StartramSettings {
			return config.StartramSettings{WgRegistered: true}
		},
		ShipSettingsSnapshotFn: func() config.ShipSettings { return config.ShipSettings{} },
	}
	rt.fileOps = RuntimeFileOps{
		OpenFn:      func(string) (*os.File, error) { return nil, os.ErrNotExist },
		ReadFileFn:  func(string) ([]byte, error) { return nil, os.ErrNotExist },
		WriteFileFn: func(string, []byte, os.FileMode) error { return nil },
		MkdirAllFn:  func(string, os.FileMode) error { return nil },
	}
	defaultCalled := false
	rt.minioOps = RuntimeMinioOps{
		CreateDefaultMcConfFn: func() error {
			defaultCalled = true
			return nil
		},
		SetMinIOPasswordFn: func(string, string) error { return nil },
		GetMinIOPasswordFn: func(string) (string, error) { return "", nil },
	}
	var startedName, startedType string
	rt.containerOps = RuntimeContainerOps{
		StartContainerFn: func(name, ctype string) (structs.ContainerState, error) {
			startedName, startedType = name, ctype
			return structs.ContainerState{ActualStatus: "running"}, nil
		},
		UpdateContainerStateFn: func(name string, state structs.ContainerState) {
			if name == "mc" && state.ActualStatus == "running" {
				// no-op
			}
		},
	}

	if err := loadMCWithRuntime(minioRuntimeFromDocker(rt)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !defaultCalled {
		t.Fatalf("expected default mc config creation when file missing")
	}
	if startedName != "mc" || startedType != "miniomc" {
		t.Fatalf("unexpected start/update flow: %s %s", startedName, startedType)
	}
}

func TestLoadMinIOsStartsPerPier(t *testing.T) {
	rt := testMinIORuntime()
	rt.configOps = RuntimeSnapshotOps{
		StartramSettingsSnapshotFn: func() config.StartramSettings {
			return config.StartramSettings{WgRegistered: true}
		},
		ShipSettingsSnapshotFn: func() config.ShipSettings {
			return config.ShipSettings{Piers: []string{"~zod", "~bus"}}
		},
	}
	var started []string
	rt.containerOps = RuntimeContainerOps{
		StartContainerFn: func(name, ctype string) (structs.ContainerState, error) {
			started = append(started, name+":"+ctype)
			if strings.Contains(name, "~bus") {
				return structs.ContainerState{}, errors.New("boom")
			}
			return structs.ContainerState{ActualStatus: "running"}, nil
		},
		UpdateContainerStateFn: func(string, structs.ContainerState) {
			// no-op
		},
	}

	if err := loadMinIOsWithRuntime(minioRuntimeFromDocker(rt)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(started, []string{"minio_~zod:minio", "minio_~bus:minio"}) {
		t.Fatalf("unexpected started containers: %v", started)
	}
}

func TestMinioRuntimeFromDockerReturnsErrorWhenCopyRuntimeMissing(t *testing.T) {
	rt := testMinIORuntime()
	rt.commandOps = RuntimeCommandOps{}
	minioRt := minioRuntimeFromDocker(rt)

	err := minioRt.CopyFileToVolumeFn("/tmp/policy.json", "/data/", "minio_~zod", "mc_writer", func() (string, error) {
		return "", errors.New("image selector called unexpectedly")
	})
	if err == nil {
		t.Fatalf("expected copy runtime error")
	}
	if !strings.Contains(err.Error(), "missing copy-to-volume runtime") {
		t.Fatalf("expected missing copy runtime error, got %v", err)
	}
}

func TestMinioContainerConfBuildsExpectedConfig(t *testing.T) {
	rt := testMinIORuntime()
	rt.urbitOps = RuntimeUrbitOps{LoadUrbitConfigFn: func(string) error { return nil },
		UrbitConfFn: func(string) structs.UrbitDocker {
			return structs.UrbitDocker{
				UrbitNetworkConfig: structs.UrbitNetworkConfig{
					WgURL:         "ship.example",
					WgS3Port:      9000,
					WgConsolePort: 9001,
				},
			}
		},
		UpdateUrbitFn: func(string, func(*structs.UrbitDocker) error) error {
			return nil
		},
	}
	rt.imageOps = RuntimeImageOps{
		GetLatestContainerInfoFn: func(string) (registry.ImageDescriptor, error) {
			return registry.ImageDescriptor{Repo: "repo/minio", Tag: "latest", Hash: "abcd"}, nil
		},
	}
	var storedName, storedPwd string
	rt.minioOps = RuntimeMinioOps{
		SetMinIOPasswordFn: func(name, pwd string) error {
			storedName = name
			storedPwd = pwd
			return nil
		},
	}

	containerConf, hostCfg, err := minioContainerConfWithRuntime(minioRuntimeFromDocker(rt), "minio_~zod")
	if err != nil {
		t.Fatalf("minioContainerConf failed: %v", err)
	}
	if containerConf.Image != "repo/minio:latest@sha256:abcd" {
		t.Fatalf("unexpected image: %s", containerConf.Image)
	}
	if !strings.Contains(strings.Join(containerConf.Env, " "), "MINIO_ROOT_USER=~zod") || !strings.Contains(strings.Join(containerConf.Env, " "), "MINIO_DOMAIN=s3.ship.example") {
		t.Fatalf("unexpected env: %v", containerConf.Env)
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
	_, _, err := minioContainerConfWithRuntime(minioRuntimeFromDocker(rt), "broken-name")
	if err == nil {
		t.Fatalf("expected invalid container name error")
	}
}

func TestMCContainerConfPollsUntilWireguardUp(t *testing.T) {
	rt := testMinIORuntime()
	rt.imageOps = RuntimeImageOps{
		GetLatestContainerInfoFn: func(string) (registry.ImageDescriptor, error) {
			return registry.ImageDescriptor{Repo: "repo/mc", Tag: "latest", Hash: "hash"}, nil
		},
	}
	calls := 0
	rt.containerOps = RuntimeContainerOps{
		GetContainerRunningStatusFn: func(string) (string, error) {
			calls++
			if calls < 2 {
				return "Exited", nil
			}
			return "Up 2 seconds", nil
		},
	}
	sleepCalls := 0
	rt.timerOps = RuntimeTimerOps{
		SleepFn:        func(time.Duration) { sleepCalls++ },
		PollIntervalFn: func() time.Duration { return 500 * time.Millisecond },
	}

	containerCfg, hostCfg, err := mcContainerConfWithRuntime(minioRuntimeFromDocker(rt))
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
	rt.contextOps = RuntimeContextOps{
		DockerDirFn: func() string { return tmpDockerDir },
	}
	rt.urbitOps = RuntimeUrbitOps{
		UrbitConfFn: func(string) structs.UrbitDocker {
			return structs.UrbitDocker{
				UrbitNetworkConfig: structs.UrbitNetworkConfig{
					WgS3Port: 9000,
				},
			}
		},
	}
	rt.containerOps = RuntimeContainerOps{
		GetContainerRunningStatusFn: func(string) (string, error) { return "Up", nil },
	}
	rt.minioOps = RuntimeMinioOps{
		GetMinIOPasswordFn: func(string) (string, error) { return "secret", nil },
	}
	var commands [][]string
	rt.commandOps = RuntimeCommandOps{
		ExecDockerCommandExitFn: func(_ string, cmd []string) (string, int, error) {
			commands = append(commands, append([]string(nil), cmd...))
			return "ok", 0, nil
		},
	}
	var scriptPath string
	rt.fileOps = RuntimeFileOps{
		WriteFileFn: func(path string, _ []byte, _ os.FileMode) error {
			scriptPath = path
			return nil
		},
	}

	if err := setMinIOAdminAccountWithRuntime(minioRuntimeFromDocker(rt), "minio_~zod"); err != nil {
		t.Fatalf("setMinIOAdminAccount failed: %v", err)
	}
	if len(commands) != 3 {
		t.Fatalf("expected 3 docker exec commands, got %d", len(commands))
	}
	if commands[0][0] != "mc" || commands[1][1] != "mb" || commands[2][2] != "set-json" {
		t.Fatalf("unexpected command sequence: %v", commands)
	}
	wantPath := filepath.Join(rt.contextOps.DockerDirFn(), "minio_~zod", "_data", "policy.json")
	if scriptPath != wantPath {
		t.Fatalf("unexpected policy path: got %s want %s", scriptPath, wantPath)
	}
}

func TestCreateMinIOServiceAccountFlow(t *testing.T) {
	rt := testMinIORuntime()
	calls := 0
	rt.commandOps = RuntimeCommandOps{
		RandReadFn: func(dst []byte) (int, error) {
			for i := range dst {
				dst[i] = 0xAA
			}
			return len(dst), nil
		},
		ExecDockerCommandFn: func(_ string, _ []string) (string, error) {
			calls++
			if calls == 1 {
				return "successfully removed", nil
			}
			return "service account created", nil
		},
		ExecDockerCommandExitFn: func(_ string, _ []string) (string, int, error) {
			return "service account created", 0, nil
		},
	}
	svc, err := createMinIOServiceAccountWithRuntime(minioRuntimeFromDocker(rt), "~zod")
	if err != nil {
		t.Fatalf("CreateMinIOServiceAccount failed: %v", err)
	}
	if svc.AccessKey != "urbit_minio" || svc.Alias != "patp_~zod" || svc.User != "~zod" || len(svc.SecretKey) != 40 {
		t.Fatalf("unexpected service account: %+v", svc)
	}
}

func TestCreateMinIOServiceAccountErrors(t *testing.T) {
	rt := testMinIORuntime()
	rt.commandOps = RuntimeCommandOps{
		RandReadFn:          func(dst []byte) (int, error) { return len(dst), nil },
		ExecDockerCommandFn: func(string, []string) (string, error) { return "", errors.New("docker failed") },
		ExecDockerCommandExitFn: func(string, []string) (string, int, error) {
			return "", -1, errors.New("docker failed")
		},
	}
	if _, err := createMinIOServiceAccountWithRuntime(minioRuntimeFromDocker(rt), "~zod"); err == nil {
		t.Fatalf("expected docker failure")
	}

	calls := 0
	rt.commandOps = RuntimeCommandOps{
		RandReadFn: func(dst []byte) (int, error) { return len(dst), nil },
		ExecDockerCommandExitFn: func(string, []string) (string, int, error) {
			calls++
			if calls == 1 {
				return "no such access key", 1, errors.New("no such access key")
			}
			return "service account created", 0, nil
		},
	}
	_, err := createMinIOServiceAccountWithRuntime(minioRuntimeFromDocker(rt), "~zod")
	if err != nil {
		t.Fatalf("expected remove step warning only and add step success, got %v", err)
	}
}
