package orchestration

import (
	"errors"
	"groundseg/docker/registry"
	"groundseg/structs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testNetdataRuntime() dockerRuntime {
	rt := newDockerRuntime()
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
	}
	rt.imageOps = RuntimeImageOps{
		GetLatestContainerInfoFn: func(string) (registry.ImageDescriptor, error) {
			return registry.ImageDescriptor{Repo: "repo/netdata", Tag: "stable", Hash: "abcd"}, nil
		},
		GetLatestContainerImageFn: func(string) (string, error) { return "netdata:latest", nil },
	}
	rt.commandOps = RuntimeCommandOps{
		CopyFileToVolumeFn: func(string, string, string, string, volumeWriterImageSelector) error { return nil },
	}
	rt.volumeOps = RuntimeVolumeOps{
		VolumeExistsFn: func(string) (bool, error) { return false, nil },
		CreateVolumeFn: func(string) error { return nil },
	}
	return rt
}

func TestNetdataContainerConfBuildsExpectedConfig(t *testing.T) {
	rt := testNetdataRuntime()
	rt.imageOps = RuntimeImageOps{
		GetLatestContainerInfoFn: func(string) (registry.ImageDescriptor, error) {
			return registry.ImageDescriptor{Repo: "repo/netdata", Tag: "stable", Hash: "abcd"}, nil
		},
	}

	containerCfg, hostCfg, err := netdataContainerConfWithRuntime(netdataRuntimeFromDocker(rt))
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
	rt.contextOps = RuntimeContextOps{DockerDirFn: func() string { return tmpDockerDir }}
	rt.fileOps = RuntimeFileOps{
		ReadFileFn:  func(string) ([]byte, error) { return nil, os.ErrNotExist },
		WriteFileFn: os.WriteFile,
		MkdirAllFn:  os.MkdirAll,
	}

	if err := writeNDConfWithRuntime(netdataRuntimeFromDocker(rt)); err != nil {
		t.Fatalf("WriteNDConf failed: %v", err)
	}
	confPath := filepath.Join(rt.contextOps.DockerDirFn(), "netdataconfig", "_data", "netdata.conf")
	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("expected netdata.conf to be created: %v", err)
	}
	expected := "[plugins]\n     apps = no\n"
	if string(data) != expected {
		t.Fatalf("unexpected netdata.conf content: %q", string(data))
	}
	if err := writeNDConfWithRuntime(netdataRuntimeFromDocker(rt)); err != nil {
		t.Fatalf("WriteNDConf should be idempotent, got: %v", err)
	}
}

func TestWriteNDConfToFileFallsBackToVolumeCopy(t *testing.T) {
	rt := testNetdataRuntime()
	writeAttempts := 0
	rt.fileOps = RuntimeFileOps{
		WriteFileFn: func(string, []byte, os.FileMode) error {
			writeAttempts++
			return errors.New("write failed")
		},
		MkdirAllFn: func(string, os.FileMode) error { return nil },
	}
	rt.containerOps = RuntimeContainerOps{}
	rt.commandOps = RuntimeCommandOps{
		CopyFileToVolumeFn: func(filePath string, targetPath string, volumeName string, _ string, _ volumeWriterImageSelector) error {
			if !strings.Contains(targetPath, "/etc/netdata/") || volumeName != "netdata" {
				t.Fatalf("unexpected copy args: %s %s %s", filePath, targetPath, volumeName)
			}
			return nil
		},
	}

	if err := writeNDConfToFileWithRuntime(netdataRuntimeFromDocker(rt), "/tmp/netdata.conf", "data"); err != nil {
		t.Fatalf("expected fallback copy success, got: %v", err)
	}
	if writeAttempts != 2 {
		t.Fatalf("expected two write attempts, got attempts=%d", writeAttempts)
	}
}

func TestLoadNetdataFlowAndStartError(t *testing.T) {
	rt := testNetdataRuntime()
	rt.fileOps = RuntimeFileOps{
		OpenFn: func(string) (*os.File, error) { return nil, os.ErrNotExist },
	}
	defaultCalled := false
	writeCalled := false
	rt.netdataOps = RuntimeNetdataOps{
		CreateDefaultNetdataConfFn: func() error { defaultCalled = true; return nil },
		WriteNDConfFn: func() error {
			writeCalled = true
			return nil
		},
	}
	updated := false
	rt.containerOps = RuntimeContainerOps{
		StartContainerFn: func(string, string) (structs.ContainerState, error) {
			return structs.ContainerState{ActualStatus: "running"}, nil
		},
		UpdateContainerStateFn: func(name string, _ structs.ContainerState) {
			if name == "netdata" {
				updated = true
			}
		},
	}

	if err := loadNetdataWithRuntime(netdataRuntimeFromDocker(rt)); err != nil {
		t.Fatalf("LoadNetdata failed: %v", err)
	}
	if !defaultCalled || !writeCalled || !updated {
		t.Fatalf("expected default/write/update flow, got default=%v write=%v updated=%v", defaultCalled, writeCalled, updated)
	}

	rt.containerOps = RuntimeContainerOps{
		StartContainerFn: func(string, string) (structs.ContainerState, error) {
			return structs.ContainerState{}, errors.New("start failed")
		},
	}
	if err := loadNetdataWithRuntime(netdataRuntimeFromDocker(rt)); err == nil {
		t.Fatalf("expected start error")
	}
}

func TestCopyNDFileToVolumeDelegatesToVolumeWriter(t *testing.T) {
	rt := testNetdataRuntime()
	var gotPath, gotTarget, gotVolume, gotWriter, gotImage string
	rt.commandOps = RuntimeCommandOps{
		CopyFileToVolumeFn: func(filePath string, targetPath string, volumeName string, writer string, selectImage volumeWriterImageSelector) error {
			gotPath = filePath
			gotTarget = targetPath
			gotVolume = volumeName
			gotWriter = writer
			img, err := selectImage()
			if err != nil {
				return err
			}
			gotImage = img
			return nil
		},
	}
	rt.imageOps = RuntimeImageOps{
		GetLatestContainerImageFn: func(string) (string, error) { return "netdata:latest", nil },
	}

	if err := copyNDFileToVolumeWithRuntime(netdataRuntimeFromDocker(rt), "/tmp/netdata.conf", "/etc/netdata/", "netdata"); err != nil {
		t.Fatalf("copyNDFileToVolume failed: %v", err)
	}
	if gotPath != "/tmp/netdata.conf" || gotTarget != "/etc/netdata/" || gotVolume != "netdata" || gotWriter != "nd_writer" || gotImage != "netdata:latest" {
		t.Fatalf("unexpected delegated args: %s %s %s %s %s", gotPath, gotTarget, gotVolume, gotWriter, gotImage)
	}
}

func TestCopyNDFileToVolumeReturnsErrorWhenCopyRuntimeMissing(t *testing.T) {
	rt := testNetdataRuntime()
	rt.commandOps = RuntimeCommandOps{}
	rt.imageOps.GetLatestContainerImageFn = nil

	err := copyNDFileToVolumeWithRuntime(netdataRuntimeFromDocker(rt), "/tmp/netdata.conf", "/etc/netdata/", "netdata")
	if err == nil {
		t.Fatalf("expected copy runtime error")
	}
	if !strings.Contains(err.Error(), "missing copy-to-volume runtime") {
		t.Fatalf("expected missing copy runtime error, got %v", err)
	}
}
