package metrics

import (
	"errors"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"groundseg/internal/testseams"
)

func withMetricsDependencies(
	t *testing.T,
	fn func(),
) {
	t.Helper()
	testseams.WithRestore(t, &cpuPercentFn, cpu.Percent)
	testseams.WithRestore(t, &openFn, os.Open)
	testseams.WithRestore(t, &readDirFn, ioutil.ReadDir)
	testseams.WithRestore(t, &readlinkFn, os.Readlink)
	testseams.WithRestore(t, &readFileFn, ioutil.ReadFile)
	testseams.WithRestore(t, &globFn, filepath.Glob)
	testseams.WithRestore(t, &statfsFn, syscall.Statfs)
	testseams.WithRestore(t, &queryUnescapeFn, url.QueryUnescape)
	testseams.WithRestore(t, &mountsPath, "/proc/mounts")
	testseams.WithRestore(t, &diskLabelBasePath, "/dev/disk/by-label/")
	testseams.WithRestore(t, &hwmonBasePath, "/sys/class/hwmon/")

	fn()
}

func withTempDir(t *testing.T, fn func(string)) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "metrics")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})
	fn(tempDir)
}

func TestGetCPUUsesInjectedSampler(t *testing.T) {
	withMetricsDependencies(t, func() {
		cpuPercentFn = func(_ time.Duration, _ bool) ([]float64, error) {
			return []float64{42.7}, nil
		}
		got, err := GetCPU()
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if got != 42 {
			t.Fatalf("expected sample to be floored to 42, got %v", got)
		}
	})
}

func TestGetCPUReturnsEmptySampleError(t *testing.T) {
	withMetricsDependencies(t, func() {
		cpuPercentFn = func(_ time.Duration, _ bool) ([]float64, error) {
			return []float64{}, nil
		}
		if _, err := GetCPU(); err == nil {
			t.Fatal("expected empty-sample error")
		}
	})
}

func TestGetDiskUsesMountAndLabelDependencies(t *testing.T) {
	withTempDir(t, func(tempDir string) {
		withMetricsDependencies(t, func() {
			labelBase := filepath.Join(tempDir, "labels")
			mountsFile := filepath.Join(tempDir, "mounts")
			hwmonUnused := filepath.Join(tempDir, "unused")

			if err := os.MkdirAll(filepath.Join(labelBase, "unused"), 0o755); err != nil {
				t.Fatalf("mkdir label base: %v", err)
			}
			if err := os.WriteFile(mountsFile, []byte(
				"/dev/sda1 /mnt/data ext4 rw 0 0\n"+
					"/dev/loop1 /loop ext4 rw 0 0\n"+
					"/dev/sdb1 /media/storage ext4 rw 0 0\n",
			), 0o644); err != nil {
				t.Fatalf("write mounts file: %v", err)
			}
			if err := os.MkdirAll(labelBase, 0o755); err != nil {
				t.Fatalf("mkdir label base: %v", err)
			}
			if err := os.MkdirAll(hwmonUnused, 0o755); err != nil {
				t.Fatalf("mkdir hwmon: %v", err)
			}
			hwmonBasePath = hwmonUnused

			readlinkFn = func(name string) (string, error) {
				if filepath.Base(name) == "system%2fdrive" {
					return "/dev/sda1", nil
				}
				return "", errors.New("label not found")
			}
			readDirFn = func(path string) ([]os.FileInfo, error) {
				if path == labelBase {
					return []os.FileInfo{
						mockFileInfo{path: "system%2fdrive"},
					}, nil
				}
				if path == hwmonUnused {
					return []os.FileInfo{}, nil
				}
				return ioutil.ReadDir(path)
			}
			queryUnescapeFn = url.QueryUnescape
			diskLabelBasePath = labelBase
			mountsPath = mountsFile
			statfsFn = func(path string, stat *syscall.Statfs_t) error {
				switch path {
				case "/mnt/data":
					stat.Blocks = 100
					stat.Bsize = 4096
					stat.Bfree = 20
				case "/media/storage":
					stat.Blocks = 50
					stat.Bsize = 2048
					stat.Bfree = 10
				default:
					return errors.New("unexpected mountpoint")
				}
				return nil
			}

			got, err := GetDisk()
			if err != nil {
				t.Fatalf("expected success, got %v", err)
			}
			if got["system/drive"] != [2]uint64{320 * 1024, 400 * 1024} {
				t.Fatalf("unexpected labeled usage for system drive: %#v", got["system/drive"])
			}
			if got["/dev/sdb1"] != [2]uint64{80 * 1024, 100 * 1024} {
				t.Fatalf("unexpected unlabeled usage for sdb1: %#v", got["/dev/sdb1"])
			}
		})
	})
}

func TestGetDiskPropagatesStatFailure(t *testing.T) {
	withTempDir(t, func(tempDir string) {
		withMetricsDependencies(t, func() {
			labelBase := filepath.Join(tempDir, "labels")
			mountsFile := filepath.Join(tempDir, "mounts")
			if err := os.WriteFile(mountsFile, []byte("/dev/sda1 /mnt/data ext4 rw 0 0\n"), 0o644); err != nil {
				t.Fatalf("write mounts file: %v", err)
			}
			if err := os.MkdirAll(labelBase, 0o755); err != nil {
				t.Fatalf("mkdir label base: %v", err)
			}
			diskLabelBasePath = labelBase
			mountsPath = mountsFile
			statErr := errors.New("stat failed")
			statfsFn = func(_ string, _ *syscall.Statfs_t) error {
				return statErr
			}
			readDirFn = func(path string) ([]os.FileInfo, error) {
				if path == labelBase {
					return []os.FileInfo{}, nil
				}
				return ioutil.ReadDir(path)
			}

			_, err := GetDisk()
			if err == nil {
				t.Fatal("expected disk stat failure")
			}
			if !errors.Is(err, statErr) {
				t.Fatalf("expected wrapped stat error, got: %v", err)
			}
		})
	})
}

func TestGetTempAveragesReadingsAndErrorsWhenAbsent(t *testing.T) {
	withTempDir(t, func(tempDir string) {
		withMetricsDependencies(t, func() {
			hwmonBase := filepath.Join(tempDir, "hwmon")
			readPath := filepath.Join(hwmonBase, "0")
			if err := os.MkdirAll(readPath, 0o755); err != nil {
				t.Fatalf("mkdir hwmon path: %v", err)
			}
			if err := os.WriteFile(filepath.Join(readPath, "name"), []byte("coretemp"), 0o644); err != nil {
				t.Fatalf("write hwmon name: %v", err)
			}
			if err := os.WriteFile(filepath.Join(readPath, "temp1_input"), []byte("42000"), 0o644); err != nil {
				t.Fatalf("write temp input: %v", err)
			}
			if err := os.WriteFile(filepath.Join(readPath, "temp2_input"), []byte("44000"), 0o644); err != nil {
				t.Fatalf("write temp input: %v", err)
			}
			hwmonBasePath = hwmonBase

			got, err := GetTemp()
			if err != nil {
				t.Fatalf("expected temperature average, got %v", err)
			}
			if got != 43.0 {
				t.Fatalf("expected 43.0C, got %v", got)
			}

			hwmonBasePath = filepath.Join(tempDir, "no-hwmon")
			if err := os.MkdirAll(hwmonBasePath, 0o755); err != nil {
				t.Fatalf("mkdir empty hwmon: %v", err)
			}
			readDirFn = func(path string) ([]os.FileInfo, error) {
				if path == filepath.Join(tempDir, "no-hwmon") {
					return []os.FileInfo{}, nil
				}
				return ioutil.ReadDir(path)
			}

			if _, err := GetTemp(); err == nil {
				t.Fatal("expected no temperature reading error")
			}
		})
	})
}

type mockFileInfo struct {
	path string
}

func (m mockFileInfo) Name() string       { return m.path }
func (m mockFileInfo) Size() int64        { return 0 }
func (m mockFileInfo) Mode() os.FileMode  { return 0o644 }
func (m mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m mockFileInfo) IsDir() bool        { return false }
func (m mockFileInfo) Sys() interface{}   { return nil }

var _ os.FileInfo = mockFileInfo{}
