package routines

import (
	"context"
	"errors"
	"groundseg/structs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type versionRuntimeChannelStub = versionChannelOps

type versionRuntimeConfigStub = versionConfigOps

type versionRuntimeUpdateStub = versionUpdateOps

func testVersionRuntime() versionRuntime {
	rt := newVersionRuntime()
	rt.channelOps = versionRuntimeChannelStub{
		syncVersionInfoFn:   func() (structs.Channel, bool) { return structs.Channel{}, true },
		getVersionChannelFn: func() structs.Channel { return structs.Channel{} },
		setVersionChannelFn: func(structs.Channel) {},
	}
	rt.configOps = versionRuntimeConfigStub{
		getSha256Fn:       func(string) (string, error) { return "", nil },
		architectureFn:    func() string { return "amd64" },
		basePathFn:        func() string { return "" },
		debugModeFn:       func() bool { return false },
		getConfFn:         func() structs.SysConfig { return structs.SysConfig{} },
		getShipStatusFn:   func([]string) (map[string]string, error) { return map[string]string{}, nil },
		startContainerFn:  func(string, string) (structs.ContainerState, error) { return structs.ContainerState{}, nil },
		stopContainerFn:   func(string) error { return nil },
		loadUrbitConfigFn: func(string) error { return nil },
		urbitConfFn:       func(string) structs.UrbitDocker { return structs.UrbitDocker{} },
		waitCompleteFn:    func(string) error { return nil },
		chopPierFn:        func(string) error { return nil },
		updateUrbitFn:     func(string, func(*structs.UrbitDocker) error) error { return nil },
	}
	rt.updateOps = versionRuntimeUpdateStub{
		updateDockerFn: func(_ versionConfigOps, _ string, _ structs.Channel, _ structs.Channel) error { return nil },
		updateBinaryFn: func(context.Context, versionUpdateOps, versionConfigOps, string, structs.Channel) error { return nil },
	}
	return rt
}

func versionChannelWithHashes(groundseg, netdata, wireguard, miniomc, minio, vere string) structs.Channel {
	return structs.Channel{
		Groundseg: structs.VersionDetails{Amd64Sha256: groundseg, Arm64Sha256: groundseg},
		Netdata:   structs.VersionDetails{Amd64Sha256: netdata, Arm64Sha256: netdata},
		Wireguard: structs.VersionDetails{Amd64Sha256: wireguard, Arm64Sha256: wireguard},
		Miniomc:   structs.VersionDetails{Amd64Sha256: miniomc, Arm64Sha256: miniomc},
		Minio:     structs.VersionDetails{Amd64Sha256: minio, Arm64Sha256: minio},
		Vere:      structs.VersionDetails{Amd64Sha256: vere, Arm64Sha256: vere},
	}
}

func TestContains(t *testing.T) {
	if !contains([]string{"netdata", "wireguard"}, "wireguard") {
		t.Fatal("expected contains to find existing item")
	}
	if contains([]string{"netdata", "wireguard"}, "minio") {
		t.Fatal("expected contains to return false for missing item")
	}
}

func TestGetSha256(t *testing.T) {
	file := filepath.Join(t.TempDir(), "bin")
	if err := os.WriteFile(file, []byte("groundseg"), 0o644); err != nil {
		t.Fatalf("write test binary: %v", err)
	}
	sum, err := getSha256(file)
	if err != nil {
		t.Fatalf("getSha256 returned error: %v", err)
	}
	if sum == "" {
		t.Fatal("expected non-empty hash")
	}
	if _, err := getSha256(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected getSha256 to fail for missing file")
	}
}

func TestCallUpdaterSkipsWhenSyncFails(t *testing.T) {
	rt := testVersionRuntime()
	rt.channelOps = versionRuntimeChannelStub{
		syncVersionInfoFn: func() (structs.Channel, bool) {
			return structs.Channel{}, false
		},
	}
	called := false
	rt.updateOps = versionRuntimeUpdateStub{
		updateDockerFn: func(_ versionConfigOps, _ string, _ structs.Channel, _ structs.Channel) error {
			called = true
			return nil
		},
		updateBinaryFn: func(context.Context, versionUpdateOps, versionConfigOps, string, structs.Channel) error {
			called = true
			return nil
		},
	}

	if err := callUpdater(context.Background(), rt, "latest"); err == nil {
		t.Fatal("expected version metadata sync failure to be reported")
	}

	if called {
		t.Fatal("docker or binary update should be skipped when version sync fails")
	}
}

func TestCallUpdaterTriggersDockerAndChannelUpdate(t *testing.T) {
	rt := testVersionRuntime()
	rt.configOps = versionRuntimeConfigStub{
		getConfFn:        func() structs.SysConfig { return structs.SysConfig{} },
		architectureFn:   func() string { return "amd64" },
		getSha256Fn:      func(string) (string, error) { return "g2", nil },
		basePathFn:       func() string { return "" },
		getShipStatusFn:  func([]string) (map[string]string, error) { return map[string]string{}, nil },
		startContainerFn: func(string, string) (structs.ContainerState, error) { return structs.ContainerState{}, nil },
	}
	current := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v1")
	latest := versionChannelWithHashes("g2", "n2", "w1", "m1", "s1", "v1")

	rt.channelOps = versionRuntimeChannelStub{
		syncVersionInfoFn:   func() (structs.Channel, bool) { return latest, true },
		getVersionChannelFn: func() structs.Channel { return current },
		setVersionChannelFn: func(ch structs.Channel) {
			if ch != latest {
				t.Fatalf("unexpected channel set value")
			}
		},
	}
	dockerCalls := 0
	rt.updateOps = versionRuntimeUpdateStub{
		updateDockerFn: func(_ versionConfigOps, release string, gotCurrent, gotLatest structs.Channel) error {
			dockerCalls++
			if release != "latest" {
				t.Fatalf("unexpected release: %s", release)
			}
			if gotCurrent != current || gotLatest != latest {
				t.Fatalf("unexpected docker update args")
			}
			return nil
		},
		updateBinaryFn: func(context.Context, versionUpdateOps, versionConfigOps, string, structs.Channel) error {
			t.Fatal("binary update should not run when hashes match")
			return nil
		},
	}

	if err := callUpdater(context.Background(), rt, "latest"); err != nil {
		t.Fatalf("expected docker update path to succeed, got: %v", err)
	}
	if dockerCalls != 1 {
		t.Fatalf("expected one docker update call, got %d", dockerCalls)
	}
}

func TestCallUpdaterTriggersBinaryUpdateOnHashMismatch(t *testing.T) {
	rt := testVersionRuntime()
	rt.configOps = versionRuntimeConfigStub{
		architectureFn: func() string { return "amd64" },
		getSha256Fn:    func(string) (string, error) { return "different-local-hash", nil },
	}

	current := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v1")
	latest := versionChannelWithHashes("g2", "n1", "w1", "m1", "s1", "v1")
	rt.channelOps = versionRuntimeChannelStub{
		syncVersionInfoFn:   func() (structs.Channel, bool) { return latest, true },
		getVersionChannelFn: func() structs.Channel { return current },
		setVersionChannelFn: func(structs.Channel) {},
	}
	binaryCalls := 0
	rt.updateOps = versionRuntimeUpdateStub{
		updateDockerFn: func(_ versionConfigOps, _ string, _ structs.Channel, _ structs.Channel) error { return nil },
		updateBinaryFn: func(_ context.Context, _ versionUpdateOps, _ versionConfigOps, branch string, info structs.Channel) error {
			binaryCalls++
			if branch != "latest" || info != latest {
				t.Fatalf("unexpected binary update args")
			}
			return nil
		},
	}

	if err := callUpdater(context.Background(), rt, "latest"); err != nil {
		t.Fatalf("expected binary update path to succeed, got: %v", err)
	}
	if binaryCalls != 1 {
		t.Fatalf("expected one binary update call, got %d", binaryCalls)
	}
}

func TestUpdateDockerStartsSharedServicesWhenHashesChange(t *testing.T) {
	rt := testVersionRuntime()
	rt.configOps = versionRuntimeConfigStub{
		architectureFn: func() string { return "amd64" },
		getConfFn:      func() structs.SysConfig { return structs.SysConfig{} },
		getShipStatusFn: func([]string) (map[string]string, error) {
			return nil, errors.New("status lookup failed")
		},
		startContainerFn: func(name, typ string) (structs.ContainerState, error) {
			return structs.ContainerState{}, nil
		},
	}

	current := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v1")
	latest := versionChannelWithHashes("g1", "n2", "w2", "m2", "s1", "v1")
	if err := updateDockerForRuntime(rt.configOps, "latest", current, latest); err == nil {
		t.Fatal("expected docker update to return an error when ship status lookup fails")
	}

	rt.updateOps = versionRuntimeUpdateStub{}
	called := map[string]bool{}
	rt.configOps = versionRuntimeConfigStub{
		architectureFn: func() string { return "amd64" },
		getConfFn:      func() structs.SysConfig { return structs.SysConfig{} },
		startContainerFn: func(name, typ string) (structs.ContainerState, error) {
			called[name+":"+typ] = true
			return structs.ContainerState{}, nil
		},
		getShipStatusFn: func([]string) (map[string]string, error) { return map[string]string{}, nil },
	}
	if err := updateDockerForRuntime(rt.configOps, "latest", current, latest); err != nil {
		t.Fatalf("expected docker update to succeed, got %v", err)
	}

	if _, ok := called["netdata:netdata"]; !ok {
		t.Fatal("expected netdata container start")
	}
	if _, ok := called["wireguard:wireguard"]; !ok {
		t.Fatal("expected wireguard container start")
	}
	if _, ok := called["miniomc:miniomc"]; !ok {
		t.Fatal("expected miniomc container start")
	}
}

func TestUpdateDockerMinioOnlyStartsRunningPiers(t *testing.T) {
	rt := testVersionRuntime()
	rt.configOps = versionRuntimeConfigStub{
		architectureFn: func() string { return "amd64" },
		getConfFn: func() structs.SysConfig {
			return structs.SysConfig{Connectivity: structs.ConnectivityConfig{Piers: []string{"zod", "nec"}}}
		},
		getShipStatusFn: func([]string) (map[string]string, error) {
			return map[string]string{
				"zod": "Up 1 second",
				"nec": "Exited",
			}, nil
		},
		startContainerFn: func(name, typ string) (structs.ContainerState, error) {
			if name == "minio_nec" {
				return structs.ContainerState{}, errors.New("boom")
			}
			return structs.ContainerState{}, nil
		},
	}

	current := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v1")
	latest := versionChannelWithHashes("g1", "n1", "w1", "m1", "s2", "v1")

	started := 0
	rt.configOps = versionRuntimeConfigStub{
		architectureFn: func() string { return "amd64" },
		getConfFn: func() structs.SysConfig {
			return structs.SysConfig{Connectivity: structs.ConnectivityConfig{Piers: []string{"zod", "nec"}}}
		},
		getShipStatusFn: func([]string) (map[string]string, error) {
			return map[string]string{
				"zod": "Up 1 second",
				"nec": "Exited",
			}, nil
		},
		startContainerFn: func(name, typ string) (structs.ContainerState, error) {
			started++
			if name == "minio_zod" {
				return structs.ContainerState{}, nil
			}
			if name == "minio_nec" {
				return structs.ContainerState{}, errors.New("boom")
			}
			return structs.ContainerState{}, nil
		},
	}
	if err := updateDockerForRuntime(rt.configOps, "latest", current, latest); err != nil {
		t.Fatalf("expected docker update to succeed, got %v", err)
	}
	if started != 1 {
		t.Fatalf("expected one start call for running piers, got %d", started)
	}
}

func TestUpdateDockerVereStoppedShipPrepFlow(t *testing.T) {
	rt := testVersionRuntime()
	seenStatus := ""
	rt.configOps = versionRuntimeConfigStub{
		architectureFn: func() string { return "amd64" },
		getConfFn: func() structs.SysConfig {
			return structs.SysConfig{Connectivity: structs.ConnectivityConfig{Piers: []string{"zod"}}}
		},
		getShipStatusFn:   func([]string) (map[string]string, error) { return map[string]string{"zod": "Exited"}, nil },
		loadUrbitConfigFn: func(string) error { return nil },
		urbitConfFn:       func(string) structs.UrbitDocker { return structs.UrbitDocker{} },
		stopContainerFn:   func(string) error { t.Fatal("stopContainer should not run for stopped ship"); return nil },
		updateUrbitFn: func(_ string, update func(*structs.UrbitDocker) error) error {
			cfg := structs.UrbitDocker{}
			if err := update(&cfg); err != nil {
				return err
			}
			seenStatus = cfg.BootStatus
			return nil
		},
		startContainerFn: func(name, typ string) (structs.ContainerState, error) {
			if name != "zod" || typ != "vere" {
				t.Fatalf("unexpected start call: %s %s", name, typ)
			}
			return structs.ContainerState{}, nil
		},
		waitCompleteFn: func(string) error { return nil },
	}

	current := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v1")
	latest := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v2")
	if err := updateDockerForRuntime(rt.configOps, "latest", current, latest); err != nil {
		t.Fatalf("expected docker update to succeed, got %v", err)
	}
	if seenStatus != "noboot" {
		t.Fatalf("expected final boot status noboot, got %s", seenStatus)
	}
}

func TestStartVersionSubsystemWithContextReturnsImmediately(t *testing.T) {
	original := runVersionSubsystemWithContextFn
	t.Cleanup(func() {
		runVersionSubsystemWithContextFn = original
	})

	blocked := make(chan struct{}, 1)
	release := make(chan struct{})
	runVersionSubsystemWithContextFn = func(context.Context) error {
		blocked <- struct{}{}
		<-release
		return nil
	}

	start := time.Now()
	if err := StartVersionSubsystemWithContext(context.Background()); err != nil {
		t.Fatalf("StartVersionSubsystemWithContext returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("expected StartVersionSubsystemWithContext to be non-blocking, took %v", elapsed)
	}
	select {
	case <-blocked:
	case <-time.After(time.Second):
		t.Fatal("expected version subsystem worker to start")
	}
	close(release)
}

func TestStartVersionSubsystemWithContextHandleCapturesTerminalError(t *testing.T) {
	original := runVersionSubsystemWithContextFn
	t.Cleanup(func() {
		runVersionSubsystemWithContextFn = original
	})

	expectedErr := errors.New("version worker failed")
	runVersionSubsystemWithContextFn = func(context.Context) error {
		return expectedErr
	}

	handle, err := StartVersionSubsystemWithContextHandle(context.Background())
	if err != nil {
		t.Fatalf("StartVersionSubsystemWithContextHandle returned error: %v", err)
	}
	if handle == nil {
		t.Fatal("expected async handle")
	}

	select {
	case <-handle.Done():
	case <-time.After(time.Second):
		t.Fatal("expected version handle to complete")
	}
	if !errors.Is(handle.Err(), expectedErr) {
		t.Fatalf("expected terminal error %v, got %v", expectedErr, handle.Err())
	}
}

func TestUpdateDockerVereRunningShipRestartAndChop(t *testing.T) {
	rt := testVersionRuntime()
	rt.configOps = versionRuntimeConfigStub{
		architectureFn: func() string { return "amd64" },
		getConfFn: func() structs.SysConfig {
			return structs.SysConfig{Connectivity: structs.ConnectivityConfig{Piers: []string{"zod"}}}
		},
		getShipStatusFn:   func([]string) (map[string]string, error) { return map[string]string{"zod": "Up 1 second"}, nil },
		loadUrbitConfigFn: func(string) error { return nil },
		urbitConfFn: func(string) structs.UrbitDocker {
			return structs.UrbitDocker{
				UrbitFeatureConfig: structs.UrbitFeatureConfig{
					ChopOnUpgrade: true,
				},
			}
		},
		stopContainerFn: func(name string) error {
			if name != "zod" {
				t.Fatalf("unexpected stop target %s", name)
			}
			return nil
		},
		updateUrbitFn: func(_ string, update func(*structs.UrbitDocker) error) error {
			cfg := structs.UrbitDocker{}
			if err := update(&cfg); err != nil {
				return err
			}
			if cfg.BootStatus != "noboot" && cfg.BootStatus != "prep" && cfg.BootStatus != "boot" {
				t.Fatalf("unexpected boot status %s", cfg.BootStatus)
			}
			return nil
		},
		startContainerFn: func(name, typ string) (structs.ContainerState, error) {
			if name != "zod" || typ != "vere" {
				t.Fatalf("unexpected start call: %s %s", name, typ)
			}
			return structs.ContainerState{}, nil
		},
		waitCompleteFn: func(string) error { return nil },
		chopPierFn: func(patp string) error {
			if patp != "zod" {
				t.Fatalf("unexpected chop target %s", patp)
			}
			return nil
		},
	}

	current := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v1")
	latest := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v2")
	if err := updateDockerForRuntime(rt.configOps, "latest", current, latest); err != nil {
		t.Fatalf("expected docker update to succeed, got %v", err)
	}
}

func TestUpdateDockerReturnsOnStatusError(t *testing.T) {
	rt := testVersionRuntime()
	rt.configOps = versionRuntimeConfigStub{
		architectureFn: func() string { return "amd64" },
		getConfFn: func() structs.SysConfig {
			return structs.SysConfig{Connectivity: structs.ConnectivityConfig{Piers: []string{"zod"}}}
		},
		getShipStatusFn: func([]string) (map[string]string, error) { return nil, errors.New("status error") },
		startContainerFn: func(string, string) (structs.ContainerState, error) {
			t.Fatal("should not start containers when status lookup fails")
			return structs.ContainerState{}, nil
		},
	}

	current := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v1")
	latest := versionChannelWithHashes("g1", "n2", "w1", "m1", "s1", "v1")
	if err := updateDockerForRuntime(rt.configOps, "latest", current, latest); err == nil {
		t.Fatal("expected docker update to fail when ship status lookup fails")
	}
}
