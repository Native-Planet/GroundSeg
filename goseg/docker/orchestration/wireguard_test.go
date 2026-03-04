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

type wireguardRuntimeOption func(*WireguardRuntime)

func testWireguardRuntime(overrides ...wireguardRuntimeOption) WireguardRuntime {
	runtime := newWireguardRuntimeForTests()
	for _, apply := range overrides {
		apply(&runtime)
	}
	return runtime
}

func withWireguardLatestContainerInfo(info map[string]string) wireguardRuntimeOption {
	return func(runtime *WireguardRuntime) {
		runtime.GetLatestContainerInfoFn = func(string) (map[string]string, error) {
			return info, nil
		}
	}
}

func withWireguardWgConfBlob(blob string) wireguardRuntimeOption {
	return func(runtime *WireguardRuntime) {
		runtime.GetWgConfBlobFn = func() (string, error) {
			return base64.StdEncoding.EncodeToString([]byte(blob)), nil
		}
	}
}

func withWireguardWgPrivateKey(privateKey string) wireguardRuntimeOption {
	return func(runtime *WireguardRuntime) {
		runtime.GetWgPrivkeyFn = func() string {
			return privateKey
		}
	}
}

func withWireguardWgConfig(cfg structs.WgConfig) wireguardRuntimeOption {
	return func(runtime *WireguardRuntime) {
		runtime.GetWgConfFn = func() (structs.WgConfig, error) {
			return cfg, nil
		}
	}
}

func withWireguardLatestImage(image string) wireguardRuntimeOption {
	return func(runtime *WireguardRuntime) {
		runtime.GetLatestContainerImageFn = func(string) (string, error) {
			return image, nil
		}
	}
}

func withWireguardRuntimeFileOps(
	readFn func(string) ([]byte, error),
	writeFn func(string, []byte, os.FileMode) error,
	mkdirFn func(string, os.FileMode) error,
) wireguardRuntimeOption {
	return func(runtime *WireguardRuntime) {
		runtime.ReadFileFn = readFn
		runtime.WriteFileFn = writeFn
		runtime.MkdirAllFn = mkdirFn
	}
}

func withWireguardDockerDir(dockerDir string) wireguardRuntimeOption {
	return func(runtime *WireguardRuntime) {
		runtime.DockerDirFn = func() string { return dockerDir }
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
	rt.WriteWgConfFn = func() error {
		writeCalled = true
		return nil
	}
	rt.StartContainerFn = func(string, string) (structs.ContainerState, error) {
		return structs.ContainerState{ActualStatus: "running"}, nil
	}
	updated := false
	rt.UpdateContainerStateFn = func(name string, _ structs.ContainerState) {
		if name == "wireguard" {
			updated = true
		}
	}

	if err := rt.LoadWireguard(); err != nil {
		t.Fatalf("loadWireguard failed: %v", err)
	}
	if !defaultCalled || !writeCalled || !updated {
		t.Fatalf("expected default/write/update flow, got default=%v write=%v updated=%v", defaultCalled, writeCalled, updated)
	}

	rt.StartContainerFn = func(string, string) (structs.ContainerState, error) {
		return structs.ContainerState{}, errors.New("start failed")
	}
	if err := rt.LoadWireguard(); err == nil {
		t.Fatalf("expected start failure")
	}
}

func TestWgContainerConfBuildsExpectedConfig(t *testing.T) {
	cfg := structs.WgConfig{CapAdd: []string{"NET_ADMIN", "SYS_MODULE"}}
	cfg.Sysctls.NetIpv4ConfAllSrcValidMark = 1
	rt := testWireguardRuntime(
		withWireguardLatestContainerInfo(map[string]string{"repo": "repo/wg", "tag": "latest", "hash": "hash"}),
		withWireguardWgConfig(cfg),
	)

	containerCfg, hostCfg, err := rt.wgContainerConf()
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
	rt := testWireguardRuntime(
		withWireguardWgConfBlob("PrivateKey = privkey\nAddress = 10.0.0.2/24\n"),
		withWireguardWgPrivateKey("actual-private-key"),
	)

	rendered, err := rt.buildWgConf()
	if err != nil {
		t.Fatalf("buildWgConf failed: %v", err)
	}
	if !strings.Contains(rendered, "actual-private-key") || strings.Contains(rendered, "privkey") {
		t.Fatalf("expected private key replacement, got %q", rendered)
	}

	rt = testWireguardRuntime(
		withWireguardWgPrivateKey("actual-private-key"),
	)
	rt.GetWgConfBlobFn = func() (string, error) {
		return "***invalid-base64***", nil
	}
	if _, err := rt.buildWgConf(); err == nil {
		t.Fatalf("expected base64 decode error")
	}
}

func TestWriteWgConfCreateAndUpdate(t *testing.T) {
	tmpDockerDir := t.TempDir()
	rt := testWireguardRuntime(
		withWireguardDockerDir(tmpDockerDir),
		withWireguardWgConfBlob("privkey"),
		withWireguardWgConfig(structs.WgConfig{}),
		withWireguardRuntimeFileOps(os.ReadFile, os.WriteFile, os.MkdirAll),
	)

	if err := rt.WriteWgConf(); err != nil {
		t.Fatalf("writeWgConf failed: %v", err)
	}
	confPath := filepath.Join(rt.DockerDirFn(), "wireguard", "_data", "wg0.conf")
	first, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read wg0.conf failed: %v", err)
	}
	if string(first) != "k1" {
		t.Fatalf("unexpected first wg0.conf content: %q", string(first))
	}

	rt = testWireguardRuntime(
		withWireguardDockerDir(tmpDockerDir),
		withWireguardWgConfBlob("privkey"),
		withWireguardWgConfig(structs.WgConfig{}),
		withWireguardRuntimeFileOps(os.ReadFile, os.WriteFile, os.MkdirAll),
		withWireguardWgPrivateKey("k2"),
	)
	if err := rt.WriteWgConf(); err != nil {
		t.Fatalf("writeWgConf update failed: %v", err)
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
	tmpDockerDir := t.TempDir()
	readErr := errors.New("permission denied")
	rt := testWireguardRuntime(
		withWireguardDockerDir(tmpDockerDir),
		withWireguardWgConfBlob("privkey"),
		withWireguardRuntimeFileOps(
			func(string) ([]byte, error) {
				return nil, readErr
			},
			os.WriteFile,
			os.MkdirAll,
		),
	)

	if err := rt.WriteWgConf(); err == nil {
		t.Fatalf("expected writeWgConf to fail on read error")
	}
}

func TestWriteWgConfToFileFallbackCopy(t *testing.T) {
	writeAttempts := 0
	rt := testWireguardRuntime(
		withWireguardRuntimeFileOps(
			func(string) ([]byte, error) {
				return nil, nil
			},
			func(string, []byte, os.FileMode) error {
				writeAttempts++
				return errors.New("write fail")
			},
			os.MkdirAll,
		),
		withWireguardLatestImage("wg:latest"),
		withWireguardWgConfBlob("content"),
		withWireguardDockerDir("/tmp/docker"),
		withWireguardWgConfig(structs.WgConfig{}),
		withWireguardWgPrivateKey("k1"),
		withWireguardLatestContainerInfo(map[string]string{"repo": "repo/wg", "tag": "latest", "hash": "hash"}),
	)
	copyCalled := false
	rt.CopyFileToVolumeFn = func(filePath string, targetPath string, volumeName string, _ string, _ volumeWriterImageSelector) error {
		copyCalled = true
		if targetPath != "/etc/wireguard/" || volumeName != "wireguard" {
			t.Fatalf("unexpected copy args: %s %s %s", filePath, targetPath, volumeName)
		}
		return nil
	}

	if err := rt.writeWgConfToFile("/tmp/wg0.conf", "content"); err != nil {
		t.Fatalf("expected fallback success, got %v", err)
	}
	if writeAttempts != 2 || !copyCalled {
		t.Fatalf("expected two writes and copy fallback, got attempts=%d copy=%v", writeAttempts, copyCalled)
	}
}

func TestCopyWGFileToVolumeDelegates(t *testing.T) {
	var gotPath, gotTarget, gotVolume, gotWriter, gotImage string
	rt := testWireguardRuntime(withWireguardLatestImage("wg:latest"))
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

	if err := rt.copyWGFileToVolume("/tmp/wg0.conf", "/etc/wireguard/", "wireguard"); err != nil {
		t.Fatalf("copyWGFileToVolume failed: %v", err)
	}
	if gotPath != "/tmp/wg0.conf" || gotTarget != "/etc/wireguard/" || gotVolume != "wireguard" || gotWriter != "wg_writer" || gotImage != "wg:latest" {
		t.Fatalf("unexpected delegated args: %s %s %s %s %s", gotPath, gotTarget, gotVolume, gotWriter, gotImage)
	}
}

func TestCopyWGFileToVolumeReturnsErrorWhenCopyRuntimeMissing(t *testing.T) {
	rt := testWireguardRuntime(withWireguardLatestImage("wg:latest"))
	rt.CopyFileToVolumeFn = nil

	err := rt.copyWGFileToVolume("/tmp/wg0.conf", "/etc/wireguard/", "wireguard")
	if err == nil {
		t.Fatalf("expected copy runtime error")
	}
	if !strings.Contains(err.Error(), "missing copy-to-volume runtime") {
		t.Fatalf("expected missing copy runtime error, got %v", err)
	}
}
