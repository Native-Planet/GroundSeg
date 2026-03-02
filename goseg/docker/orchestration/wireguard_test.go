package orchestration

import (
	"encoding/base64"
	"errors"
	"groundseg/structs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testWireguardRuntime() WireguardRuntime {
	return WireguardRuntime{
		BasePathFn: func() string { return "/tmp" },
		DockerDirFn: func() string {
			return "/tmp/docker"
		},
		OpenFn: func(string) (*os.File, error) {
			return nil, nil
		},
		ReadFileFn: func(string) ([]byte, error) {
			return nil, os.ErrNotExist
		},
		WriteFileFn: func(string, []byte, os.FileMode) error {
			return nil
		},
		MkdirAllFn: func(string, os.FileMode) error {
			return nil
		},
		StartContainerFn: func(string, string) (structs.ContainerState, error) {
			return structs.ContainerState{}, nil
		},
		UpdateContainerFn: func(string, structs.ContainerState) {},
		GetLatestContainerInfoFn: func(string) (map[string]string, error) {
			return map[string]string{"repo": "repo/wg", "tag": "latest", "hash": "hash"}, nil
		},
		GetLatestContainerImageFn: func(string) (string, error) {
			return "wg:latest", nil
		},
		CreateDefaultWGConfFn: func() error {
			return nil
		},
		GetWgConfFn: func() (structs.WgConfig, error) {
			return structs.WgConfig{}, nil
		},
		GetWgConfBlobFn: func() (string, error) {
			return base64.StdEncoding.EncodeToString([]byte("blob")), nil
		},
		GetWgPrivkeyFn: func() string {
			return "k1"
		},
		WriteWgConfFn: func(WireguardRuntime) error {
			return nil
		},
		CopyFileToVolumeFn: func(string, string, string, string, volumeWriterImageSelector) error {
			return nil
		},
		VolumeExistsFn: func(string) (bool, error) {
			return false, nil
		},
		CreateVolumeFn: func(string) error {
			return nil
		},
	}
}

func TestLoadWireguardFlowAndStartError(t *testing.T) {
	rt := testWireguardRuntime()

	rt.OpenFn = func(string) (*os.File, error) { return nil, os.ErrNotExist }
	defaultCalled := false
	rt.CreateDefaultWGConfFn = func() error {
		defaultCalled = true
		return nil
	}
	writeCalled := false
	rt.WriteWgConfFn = func(WireguardRuntime) error {
		writeCalled = true
		return nil
	}
	rt.StartContainerFn = func(string, string) (structs.ContainerState, error) {
		return structs.ContainerState{ActualStatus: "running"}, nil
	}
	updated := false
	rt.UpdateContainerFn = func(name string, _ structs.ContainerState) {
		if name == "wireguard" {
			updated = true
		}
	}

	if err := loadWireguard(rt); err != nil {
		t.Fatalf("loadWireguard failed: %v", err)
	}
	if !defaultCalled || !writeCalled || !updated {
		t.Fatalf("expected default/write/update flow, got default=%v write=%v updated=%v", defaultCalled, writeCalled, updated)
	}

	rt.StartContainerFn = func(string, string) (structs.ContainerState, error) {
		return structs.ContainerState{}, errors.New("start failed")
	}
	if err := loadWireguard(rt); err == nil {
		t.Fatalf("expected start failure")
	}
}

func TestWgContainerConfBuildsExpectedConfig(t *testing.T) {
	rt := testWireguardRuntime()
	rt.GetLatestContainerInfoFn = func(string) (map[string]string, error) {
		return map[string]string{"repo": "repo/wg", "tag": "latest", "hash": "hash"}, nil
	}
	rt.GetWgConfFn = func() (structs.WgConfig, error) {
		cfg := structs.WgConfig{CapAdd: []string{"NET_ADMIN", "SYS_MODULE"}}
		cfg.Sysctls.NetIpv4ConfAllSrcValidMark = 1
		return cfg, nil
	}

	containerCfg, hostCfg, err := wgContainerConfWithRuntime(rt)
	if err != nil {
		t.Fatalf("wgContainerConf failed: %v", err)
	}
	if containerCfg.Image != "repo/wg:latest@sha256:hash" || containerCfg.Hostname != "wireguard" {
		t.Fatalf("unexpected container config: %+v", containerCfg)
	}
	if len(hostCfg.Mounts) != 1 || hostCfg.Mounts[0].Source != "wireguard" {
		t.Fatalf("unexpected mounts: %+v", hostCfg.Mounts)
	}
	if hostCfg.Sysctls["net.ipv4.conf.all.src_valid_mark"] != "1" {
		t.Fatalf("unexpected sysctl: %+v", hostCfg.Sysctls)
	}
}

func TestBuildWgConfDecodeAndReplace(t *testing.T) {
	rt := testWireguardRuntime()
	template := "PrivateKey = privkey\nAddress = 10.0.0.2/24\n"
	rt.GetWgConfBlobFn = func() (string, error) {
		return base64.StdEncoding.EncodeToString([]byte(template)), nil
	}
	rt.GetWgPrivkeyFn = func() string { return "actual-private-key" }

	rendered, err := buildWgConfWithRuntime(rt)
	if err != nil {
		t.Fatalf("buildWgConf failed: %v", err)
	}
	if !strings.Contains(rendered, "actual-private-key") || strings.Contains(rendered, "privkey") {
		t.Fatalf("expected private key replacement, got %q", rendered)
	}

	rt.GetWgConfBlobFn = func() (string, error) { return "***invalid-base64***", nil }
	if _, err := buildWgConfWithRuntime(rt); err == nil {
		t.Fatalf("expected base64 decode error")
	}
}

func TestWriteWgConfCreateAndUpdate(t *testing.T) {
	rt := testWireguardRuntime()
	tmpDockerDir := t.TempDir()
	rt.DockerDirFn = func() string { return tmpDockerDir }
	rt.GetWgConfBlobFn = func() (string, error) {
		return base64.StdEncoding.EncodeToString([]byte("privkey")), nil
	}
	rt.GetWgConfFn = func() (structs.WgConfig, error) {
		return structs.WgConfig{}, nil
	}
	rt.ReadFileFn = os.ReadFile
	rt.WriteFileFn = os.WriteFile
	rt.MkdirAllFn = os.MkdirAll

	if err := writeWgConfWithRuntime(rt); err != nil {
		t.Fatalf("writeWgConfWithRuntime failed: %v", err)
	}
	confPath := filepath.Join(rt.DockerDirFn(), "wireguard", "_data", "wg0.conf")
	first, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read wg0.conf failed: %v", err)
	}
	if string(first) != "k1" {
		t.Fatalf("unexpected first wg0.conf content: %q", string(first))
	}

	rt.GetWgPrivkeyFn = func() string { return "k2" }
	if err := writeWgConfWithRuntime(rt); err != nil {
		t.Fatalf("writeWgConfWithRuntime update failed: %v", err)
	}
	second, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read updated wg0.conf failed: %v", err)
	}
	if string(second) != "k2" {
		t.Fatalf("expected updated wg0.conf content, got %q", string(second))
	}
}

func TestWriteWgConfReturnsErrorOnReadFailure(t *testing.T) {
	rt := testWireguardRuntime()
	tmpDockerDir := t.TempDir()
	rt.DockerDirFn = func() string { return tmpDockerDir }
	rt.GetWgConfBlobFn = func() (string, error) {
		return base64.StdEncoding.EncodeToString([]byte("privkey")), nil
	}
	readErr := errors.New("permission denied")
	rt.ReadFileFn = func(string) ([]byte, error) {
		return nil, readErr
	}
	rt.WriteFileFn = os.WriteFile
	rt.MkdirAllFn = os.MkdirAll

	if err := writeWgConfWithRuntime(rt); err == nil {
		t.Fatalf("expected writeWgConfWithRuntime to fail on read error")
	}
}

func TestWriteWgConfToFileFallbackCopy(t *testing.T) {
	rt := testWireguardRuntime()
	writeAttempts := 0
	rt.WriteFileFn = func(string, []byte, os.FileMode) error {
		writeAttempts++
		return errors.New("write fail")
	}
	rt.MkdirAllFn = os.MkdirAll
	copyCalled := false
	rt.CopyFileToVolumeFn = func(filePath string, targetPath string, volumeName string, _ string, _ volumeWriterImageSelector) error {
		copyCalled = true
		if targetPath != "/etc/wireguard/" || volumeName != "wireguard" {
			t.Fatalf("unexpected copy args: %s %s %s", filePath, targetPath, volumeName)
		}
		return nil
	}

	if err := writeWgConfToFileWithRuntime(rt, "/tmp/wg0.conf", "content"); err != nil {
		t.Fatalf("expected fallback success, got %v", err)
	}
	if writeAttempts != 2 || !copyCalled {
		t.Fatalf("expected two writes and copy fallback, got attempts=%d copy=%v", writeAttempts, copyCalled)
	}
}

func TestCopyWGFileToVolumeDelegates(t *testing.T) {
	rt := testWireguardRuntime()
	var gotPath, gotTarget, gotVolume, gotWriter, gotImage string
	rt.CopyFileToVolumeFn = func(
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
	rt.GetLatestContainerImageFn = func(string) (string, error) { return "wg:latest", nil }

	if err := copyWGFileToVolumeWithRuntime(rt, "/tmp/wg0.conf", "/etc/wireguard/", "wireguard"); err != nil {
		t.Fatalf("copyWGFileToVolume failed: %v", err)
	}
	if gotPath != "/tmp/wg0.conf" || gotTarget != "/etc/wireguard/" || gotVolume != "wireguard" || gotWriter != "wg_writer" || gotImage != "wg:latest" {
		t.Fatalf("unexpected delegated args: %s %s %s %s %s", gotPath, gotTarget, gotVolume, gotWriter, gotImage)
	}
}

func TestCopyWGFileToVolumeReturnsErrorWhenCopyRuntimeMissing(t *testing.T) {
	rt := testWireguardRuntime()
	rt.CopyFileToVolumeFn = nil

	err := copyWGFileToVolumeWithRuntime(rt, "/tmp/wg0.conf", "/etc/wireguard/", "wireguard")
	if err == nil {
		t.Fatalf("expected copy runtime error")
	}
	if !strings.Contains(err.Error(), "missing copy-to-volume runtime") {
		t.Fatalf("expected missing copy runtime error, got %v", err)
	}
}
