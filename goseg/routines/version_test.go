package routines

import (
	"errors"
	"groundseg/structs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testVersionRuntime() versionRuntime {
	rt := newVersionRuntime()
	rt.syncVersionInfo = func() (structs.Channel, bool) { return structs.Channel{}, true }
	rt.getVersionChannel = func() structs.Channel { return structs.Channel{} }
	rt.setVersionChannel = func(structs.Channel) {}
	rt.updateDocker = func(string, structs.Channel, structs.Channel) {}
	rt.updateBinary = func(versionRuntime, string, structs.Channel) {}
	rt.getSha256 = func(string) (string, error) { return "", nil }
	rt.architecture = func() string { return "amd64" }
	rt.basePath = func() string { return "" }
	rt.debugMode = func() bool { return false }
	rt.getConf = func() structs.SysConfig { return structs.SysConfig{} }
	rt.getShipStatus = func([]string) (map[string]string, error) { return map[string]string{}, nil }
	rt.startContainer = func(string, string) (structs.ContainerState, error) { return structs.ContainerState{}, nil }
	rt.stopContainer = func(string) error { return nil }
	rt.loadUrbitConfig = func(string) error { return nil }
	rt.urbitConf = func(string) structs.UrbitDocker { return structs.UrbitDocker{} }
	rt.waitComplete = func(string) error { return nil }
	rt.chopPier = func(string, structs.UrbitDocker) error { return nil }
	rt.updateUrbit = func(string, func(*structs.UrbitDocker) error) error { return nil }
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
	rt.syncVersionInfo = func() (structs.Channel, bool) {
		return structs.Channel{}, false
	}
	updateDockerCalled := false
	rt.updateDocker = func(string, structs.Channel, structs.Channel) {
		updateDockerCalled = true
	}
	updateBinaryCalled := false
	rt.updateBinary = func(_ versionRuntime, _ string, _ structs.Channel) {
		updateBinaryCalled = true
	}

	callUpdaterWithRuntime(rt, "latest")

	if updateDockerCalled {
		t.Fatal("docker update should be skipped when version sync fails")
	}
	if updateBinaryCalled {
		t.Fatal("binary update should be skipped when version sync fails")
	}
}

func TestCallUpdaterTriggersDockerAndChannelUpdate(t *testing.T) {
	rt := testVersionRuntime()
	rt.architecture = func() string { return "amd64" }

	current := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v1")
	latest := versionChannelWithHashes("g2", "n2", "w1", "m1", "s1", "v1")

	rt.syncVersionInfo = func() (structs.Channel, bool) {
		return latest, true
	}
	rt.getVersionChannel = func() structs.Channel {
		return current
	}
	dockerCalls := 0
	rt.updateDocker = func(release string, gotCurrent, gotLatest structs.Channel) {
		dockerCalls++
		if release != "latest" {
			t.Fatalf("unexpected release: %s", release)
		}
		if gotCurrent != current || gotLatest != latest {
			t.Fatalf("unexpected docker update args")
		}
	}
	setChannelCalls := 0
	rt.setVersionChannel = func(ch structs.Channel) {
		setChannelCalls++
		if ch != latest {
			t.Fatalf("unexpected channel set value")
		}
	}
	rt.getSha256 = func(string) (string, error) {
		return latest.Groundseg.Amd64Sha256, nil
	}
	rt.updateBinary = func(_ versionRuntime, _ string, _ structs.Channel) {
		t.Fatal("binary update should not run when hashes match")
	}

	callUpdaterWithRuntime(rt, "latest")

	if dockerCalls != 1 {
		t.Fatalf("expected one docker update call, got %d", dockerCalls)
	}
	if setChannelCalls != 1 {
		t.Fatalf("expected one setVersionChannel call, got %d", setChannelCalls)
	}
}

func TestCallUpdaterTriggersBinaryUpdateOnHashMismatch(t *testing.T) {
	rt := testVersionRuntime()
	rt.architecture = func() string { return "amd64" }

	current := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v1")
	latest := versionChannelWithHashes("g2", "n1", "w1", "m1", "s1", "v1")

	rt.syncVersionInfo = func() (structs.Channel, bool) {
		return latest, true
	}
	rt.getVersionChannel = func() structs.Channel {
		return current
	}
	rt.updateDocker = func(string, structs.Channel, structs.Channel) {}
	rt.setVersionChannel = func(structs.Channel) {}
	rt.getSha256 = func(string) (string, error) {
		return "different-local-hash", nil
	}
	binaryCalls := 0
	rt.updateBinary = func(_ versionRuntime, branch string, info structs.Channel) {
		binaryCalls++
		if branch != "latest" || info != latest {
			t.Fatalf("unexpected binary update args")
		}
	}

	callUpdaterWithRuntime(rt, "latest")

	if binaryCalls != 1 {
		t.Fatalf("expected one binary update call, got %d", binaryCalls)
	}
}

func TestUpdateDockerStartsSharedServicesWhenHashesChange(t *testing.T) {
	rt := testVersionRuntime()
	rt.architecture = func() string { return "amd64" }

	rt.getConf = func() structs.SysConfig {
		return structs.SysConfig{}
	}
	rt.getShipStatus = func([]string) (map[string]string, error) {
		return map[string]string{}, nil
	}
	var started []string
	rt.startContainer = func(name, typ string) (structs.ContainerState, error) {
		started = append(started, name+":"+typ)
		return structs.ContainerState{}, nil
	}

	current := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v1")
	latest := versionChannelWithHashes("g1", "n2", "w2", "m2", "s1", "v1")
	updateDockerWithRuntime(rt, "latest", current, latest)

	expected := map[string]bool{
		"netdata:netdata":     true,
		"wireguard:wireguard": true,
		"miniomc:miniomc":     true,
	}
	for _, key := range started {
		delete(expected, key)
	}
	if len(expected) != 0 {
		t.Fatalf("missing expected service start calls, remaining=%v all=%v", expected, started)
	}
}

func TestUpdateDockerMinioOnlyStartsRunningPiers(t *testing.T) {
	rt := testVersionRuntime()
	rt.architecture = func() string { return "amd64" }

	rt.getConf = func() structs.SysConfig {
		return structs.SysConfig{Piers: []string{"zod", "nec"}}
	}
	rt.getShipStatus = func([]string) (map[string]string, error) {
		return map[string]string{
			"zod": "Up 1 second",
			"nec": "Exited",
		}, nil
	}
	var started []string
	rt.startContainer = func(name, typ string) (structs.ContainerState, error) {
		started = append(started, name+":"+typ)
		return structs.ContainerState{}, nil
	}

	current := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v1")
	latest := versionChannelWithHashes("g1", "n1", "w1", "m1", "s2", "v1")
	updateDockerWithRuntime(rt, "latest", current, latest)

	if len(started) != 1 || started[0] != "minio_zod:minio" {
		t.Fatalf("unexpected minio starts: %v", started)
	}
}

func TestUpdateDockerVereStoppedShipPrepFlow(t *testing.T) {
	rt := testVersionRuntime()
	rt.architecture = func() string { return "amd64" }

	rt.getConf = func() structs.SysConfig {
		return structs.SysConfig{Piers: []string{"zod"}}
	}
	rt.getShipStatus = func([]string) (map[string]string, error) {
		return map[string]string{"zod": "Exited"}, nil
	}
	rt.loadUrbitConfig = func(string) error { return nil }
	rt.urbitConf = func(string) structs.UrbitDocker { return structs.UrbitDocker{} }
	rt.stopContainer = func(string) error {
		t.Fatal("stopContainer should not run for stopped ship")
		return nil
	}
	var bootStatuses []string
	rt.updateUrbit = func(_ string, update func(*structs.UrbitDocker) error) error {
		cfg := &structs.UrbitDocker{}
		if err := update(cfg); err != nil {
			return err
		}
		bootStatuses = append(bootStatuses, cfg.BootStatus)
		return nil
	}
	startCalls := 0
	rt.startContainer = func(name, typ string) (structs.ContainerState, error) {
		startCalls++
		if name != "zod" || typ != "vere" {
			t.Fatalf("unexpected start call: %s %s", name, typ)
		}
		return structs.ContainerState{}, nil
	}
	rt.waitComplete = func(string) error { return nil }

	current := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v1")
	latest := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v2")
	updateDockerWithRuntime(rt, "latest", current, latest)

	if startCalls != 1 {
		t.Fatalf("expected one vere start for prep, got %d", startCalls)
	}
	if len(bootStatuses) != 2 || bootStatuses[0] != "prep" || bootStatuses[1] != "noboot" {
		t.Fatalf("unexpected boot status updates: %v", bootStatuses)
	}
}

func TestUpdateDockerVereRunningShipRestartAndChop(t *testing.T) {
	rt := testVersionRuntime()
	rt.architecture = func() string { return "amd64" }

	rt.getConf = func() structs.SysConfig {
		return structs.SysConfig{Piers: []string{"zod"}}
	}
	rt.getShipStatus = func([]string) (map[string]string, error) {
		return map[string]string{"zod": "Up 1 second"}, nil
	}
	rt.loadUrbitConfig = func(string) error { return nil }
	rt.urbitConf = func(string) structs.UrbitDocker {
		return structs.UrbitDocker{ChopOnUpgrade: true}
	}
	stopCalls := 0
	rt.stopContainer = func(name string) error {
		stopCalls++
		if name != "zod" {
			t.Fatalf("unexpected stop target %s", name)
		}
		return nil
	}
	var bootStatuses []string
	rt.updateUrbit = func(_ string, update func(*structs.UrbitDocker) error) error {
		cfg := &structs.UrbitDocker{}
		if err := update(cfg); err != nil {
			return err
		}
		bootStatuses = append(bootStatuses, cfg.BootStatus)
		return nil
	}
	startCalls := 0
	rt.startContainer = func(name, typ string) (structs.ContainerState, error) {
		startCalls++
		if name != "zod" || typ != "vere" {
			t.Fatalf("unexpected start call: %s %s", name, typ)
		}
		return structs.ContainerState{}, nil
	}
	rt.waitComplete = func(string) error { return nil }
	chopCalled := make(chan struct{}, 1)
	rt.chopPier = func(string, structs.UrbitDocker) error {
		chopCalled <- struct{}{}
		return nil
	}

	current := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v1")
	latest := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v2")
	updateDockerWithRuntime(rt, "latest", current, latest)

	if stopCalls != 1 {
		t.Fatalf("expected one stop call, got %d", stopCalls)
	}
	if startCalls != 2 {
		t.Fatalf("expected two starts (prep and boot), got %d", startCalls)
	}
	if len(bootStatuses) != 2 || bootStatuses[0] != "prep" || bootStatuses[1] != "boot" {
		t.Fatalf("unexpected boot status updates: %v", bootStatuses)
	}
	select {
	case <-chopCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("expected chop routine to be triggered")
	}
}

func TestUpdateDockerReturnsOnStatusError(t *testing.T) {
	rt := testVersionRuntime()
	rt.getConf = func() structs.SysConfig {
		return structs.SysConfig{Piers: []string{"zod"}}
	}
	rt.getShipStatus = func([]string) (map[string]string, error) {
		return nil, errors.New("status error")
	}
	rt.startContainer = func(string, string) (structs.ContainerState, error) {
		t.Fatal("should not start containers when status lookup fails")
		return structs.ContainerState{}, nil
	}

	current := versionChannelWithHashes("g1", "n1", "w1", "m1", "s1", "v1")
	latest := versionChannelWithHashes("g1", "n2", "w1", "m1", "s1", "v1")
	updateDockerWithRuntime(rt, "latest", current, latest)
}
