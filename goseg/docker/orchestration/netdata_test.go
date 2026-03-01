package orchestration

import (
	"errors"
	"groundseg/structs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testNetdataRuntime() netdataRuntime {
	rt := newNetdataRuntime()
	rt.osOpen = func(string) (*os.File, error) { return nil, nil }
	rt.createDefaultNetdataConf = func() error { return nil }
	rt.writeNDConf = func(netdataRuntime) error { return nil }
	rt.startContainer = func(string, string) (structs.ContainerState, error) {
		return structs.ContainerState{ActualStatus: "running"}, nil
	}
	rt.updateContainerState = func(string, structs.ContainerState) {}
	rt.getLatestContainerInfo = func(string) (map[string]string, error) {
		return map[string]string{"repo": "repo/netdata", "tag": "stable", "hash": "abcd"}, nil
	}
	rt.readFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }
	rt.writeFile = func(string, []byte, os.FileMode) error { return nil }
	rt.mkdirAll = func(string, os.FileMode) error { return nil }
	rt.copyNDFileToVolume = func(netdataRuntime, string, string, string) error { return nil }
	rt.copyFileToVolumeWithTempContainer = func(string, string, string, string, volumeWriterImageSelector) error { return nil }
	rt.latestContainerImage = func(string) (string, error) { return "netdata:latest", nil }
	return rt
}

func TestNetdataContainerConfBuildsExpectedConfig(t *testing.T) {
	rt := testNetdataRuntime()
	rt.getLatestContainerInfo = func(string) (map[string]string, error) {
		return map[string]string{"repo": "repo/netdata", "tag": "stable", "hash": "abcd"}, nil
	}

	containerCfg, hostCfg, err := netdataContainerConfWithRuntime(rt)
	if err != nil {
		t.Fatalf("netdataContainerConf failed: %v", err)
	}
	if containerCfg.Image != "repo/netdata:stable@sha256:abcd" {
		t.Fatalf("unexpected image: %s", containerCfg.Image)
	}
	if _, ok := containerCfg.ExposedPorts["19999/tcp"]; !ok {
		t.Fatalf("expected port 19999 to be exposed")
	}
	if hostCfg.RestartPolicy.Name != "unless-stopped" {
		t.Fatalf("unexpected restart policy: %+v", hostCfg.RestartPolicy)
	}
	if len(hostCfg.Binds) == 0 || hostCfg.Binds[0] != "netdataconfig:/etc/netdata" {
		t.Fatalf("unexpected binds: %v", hostCfg.Binds)
	}
}

func TestWriteNDConfCreatesAndPreservesContent(t *testing.T) {
	rt := testNetdataRuntime()
	tmpDockerDir := t.TempDir()
	rt.DockerDir = func() string { return tmpDockerDir }
	rt.readFile = os.ReadFile
	rt.writeFile = os.WriteFile
	rt.mkdirAll = os.MkdirAll

	if err := writeNDConfWithRuntime(rt); err != nil {
		t.Fatalf("WriteNDConf failed: %v", err)
	}
	confPath := filepath.Join(rt.DockerDir(), "netdataconfig", "_data", "netdata.conf")
	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("expected netdata.conf to be created: %v", err)
	}
	expected := "[plugins]\n     apps = no\n"
	if string(data) != expected {
		t.Fatalf("unexpected netdata.conf content: %q", string(data))
	}
	if err := writeNDConfWithRuntime(rt); err != nil {
		t.Fatalf("WriteNDConf should be idempotent, got: %v", err)
	}
}

func TestWriteNDConfToFileFallsBackToVolumeCopy(t *testing.T) {
	rt := testNetdataRuntime()
	writeAttempts := 0
	rt.writeFile = func(string, []byte, os.FileMode) error {
		writeAttempts++
		return errors.New("write failed")
	}
	rt.mkdirAll = func(string, os.FileMode) error { return nil }
	copyCalled := false
	rt.copyNDFileToVolume = func(_ netdataRuntime, filePath, targetPath, volumeName string) error {
		copyCalled = true
		if !strings.Contains(targetPath, "/etc/netdata/") || volumeName != "netdata" {
			t.Fatalf("unexpected copy args: %s %s %s", filePath, targetPath, volumeName)
		}
		return nil
	}

	if err := writeNDConfToFileWithRuntime(rt, "/tmp/netdata.conf", "data"); err != nil {
		t.Fatalf("expected fallback copy success, got: %v", err)
	}
	if writeAttempts != 2 || !copyCalled {
		t.Fatalf("expected two write attempts and copy fallback, got attempts=%d copy=%v", writeAttempts, copyCalled)
	}
}

func TestLoadNetdataFlowAndStartError(t *testing.T) {
	rt := testNetdataRuntime()
	rt.osOpen = func(string) (*os.File, error) { return nil, os.ErrNotExist }
	defaultCalled := false
	rt.createDefaultNetdataConf = func() error { defaultCalled = true; return nil }
	writeCalled := false
	rt.writeNDConf = func(netdataRuntime) error { writeCalled = true; return nil }

	rt.startContainer = func(string, string) (structs.ContainerState, error) {
		return structs.ContainerState{ActualStatus: "running"}, nil
	}
	updated := false
	rt.updateContainerState = func(name string, _ structs.ContainerState) {
		if name == "netdata" {
			updated = true
		}
	}
	if err := loadNetdata(rt); err != nil {
		t.Fatalf("LoadNetdata failed: %v", err)
	}
	if !defaultCalled || !writeCalled || !updated {
		t.Fatalf("expected default/write/update flow, got default=%v write=%v updated=%v", defaultCalled, writeCalled, updated)
	}

	rt.startContainer = func(string, string) (structs.ContainerState, error) {
		return structs.ContainerState{}, errors.New("start failed")
	}
	if err := loadNetdata(rt); err == nil {
		t.Fatalf("expected start error")
	}
}

func TestCopyNDFileToVolumeDelegatesToVolumeWriter(t *testing.T) {
	rt := testNetdataRuntime()
	var gotPath, gotTarget, gotVolume, gotWriter, gotImage string
	rt.copyFileToVolumeWithTempContainer = func(
		filePath string,
		targetPath string,
		volumeName string,
		writerContainerName string,
		selectImage volumeWriterImageSelector,
	) error {
		gotPath = filePath
		gotTarget = targetPath
		gotVolume = volumeName
		gotWriter = writerContainerName
		img, err := selectImage()
		if err != nil {
			return err
		}
		gotImage = img
		return nil
	}
	rt.latestContainerImage = func(string) (string, error) { return "netdata:latest", nil }

	if err := copyNDFileToVolumeWithRuntime(rt, "/tmp/netdata.conf", "/etc/netdata/", "netdata"); err != nil {
		t.Fatalf("copyNDFileToVolume failed: %v", err)
	}
	if gotPath != "/tmp/netdata.conf" || gotTarget != "/etc/netdata/" || gotVolume != "netdata" || gotWriter != "nd_writer" || gotImage != "netdata:latest" {
		t.Fatalf("unexpected delegated args: %s %s %s %s %s", gotPath, gotTarget, gotVolume, gotWriter, gotImage)
	}
}
