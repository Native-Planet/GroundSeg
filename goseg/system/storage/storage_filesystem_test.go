package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNextGroundSegPath(t *testing.T) {
	t.Parallel()

	attempts := 0
	path, err := nextGroundSegPath(DiskSeams{
		StatFn: func(path string) (os.FileInfo, error) {
			attempts++
			if attempts <= 2 {
				return stubFileInfo{mode: os.ModeDir}, nil
			}
			return nil, os.ErrNotExist
		},
	})
	if err != nil {
		t.Fatalf("nextGroundSegPath returned error: %v", err)
	}
	if path != "/groundseg-3" {
		t.Fatalf("expected /groundseg-3, got %q", path)
	}
}

func TestNextGroundSegPathPropagatesStatError(t *testing.T) {
	t.Parallel()

	statErr := errors.New("disk full")
	if _, err := nextGroundSegPath(DiskSeams{
		StatFn: func(_ string) (os.FileInfo, error) { return nil, statErr },
	}); err == nil || !errors.Is(err, statErr) {
		t.Fatalf("expected stat error, got %v", err)
	}
}

func TestFilePathExists(t *testing.T) {
	t.Parallel()

	if exists, err := filePathExists("/missing", func(_ string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}); err != nil {
		t.Fatalf("filePathExists returned error: %v", err)
	} else if exists {
		t.Fatalf("expected missing path to return exists=false")
	}

	exists, err := filePathExists("/tmp", func(_ string) (os.FileInfo, error) {
		return stubFileInfo{mode: 0}, nil
	})
	if err != nil {
		t.Fatalf("filePathExists returned error: %v", err)
	}
	if !exists {
		t.Fatal("expected path to exist")
	}
}

func TestFormatGroundSegFilesystem(t *testing.T) {
	t.Parallel()

	var capturedUUID string
	seams := DiskSeams{
		MkfsExt4CommandFn: func(uuid, devPath string) error {
			if devPath != "/dev/sda" {
				t.Fatalf("unexpected device path %q", devPath)
			}
			if uuid == "" {
				t.Fatal("expected generated UUID")
			}
			capturedUUID = uuid
			return nil
		},
	}
	uuid, err := formatGroundSegFilesystem("sda", seams)
	if err != nil {
		t.Fatalf("formatGroundSegFilesystem error: %v", err)
	}
	if uuid != capturedUUID {
		t.Fatalf("expected returned uuid %q, got %q", capturedUUID, uuid)
	}
}

func TestReconcileGroundSegFstabWritesAndMounts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	in := filepath.Join(dir, "fstab.in")
	out := filepath.Join(dir, "fstab.out")
	if err := os.WriteFile(in, []byte("tmpfs /tmp tmpfs defaults 0 0\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	seams := DiskSeams{
		RunDiskCommandFn: func(string, ...string) (string, error) {
			return `{"blockdevices":[{"name":"sda","mountpoints":["/"]}]}`, nil
		},
		OpenFn: func(path string) (*os.File, error) {
			if path != "/etc/fstab" {
				t.Fatalf("expected /etc/fstab, got %q", path)
			}
			return os.Open(in)
		},
		OpenFileFn: func(path string, _ int, _ os.FileMode) (*os.File, error) {
			if path != "/etc/fstab" {
				t.Fatalf("expected /etc/fstab, got %q", path)
			}
			return os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		},
		MountAllCommandFn: func() error { return nil },
	}

	if err := reconcileGroundSegFstab("sda", "/groundseg-1", "uuid-1234", seams); err != nil {
		t.Fatalf("reconcileGroundSegFstab error: %v", err)
	}

	content, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(content), "/groundseg-1 ext4 defaults,nofail 0 2") {
		t.Fatalf("expected updated fstab line, got %q", content)
	}

	if !strings.Contains(string(content), "tmpfs /tmp tmpfs defaults 0 0") {
		t.Fatalf("expected preserved unrelated line, got %q", content)
	}
}

func TestReconcileGroundSegFstabSkipsMissingDevice(t *testing.T) {
	t.Parallel()

	var mounted bool
	seams := DiskSeams{
		RunDiskCommandFn: func(string, ...string) (string, error) {
			return `{"blockdevices":[{"name":"nvme0n1","mountpoints":["/"]}]}`, nil
		},
		MountAllCommandFn: func() error {
			mounted = true
			return nil
		},
	}
	if err := reconcileGroundSegFstab("sda", "/groundseg-1", "uuid-1234", seams); err != nil {
		t.Fatalf("reconcileGroundSegFstab error: %v", err)
	}
	if mounted {
		t.Fatal("did not expect mount to run when selected device is absent")
	}
}

func TestCreateGroundSegFilesystem(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	in := filepath.Join(dir, "fstab.in")
	out := filepath.Join(dir, "fstab.out")
	if err := os.WriteFile(in, []byte("tmpfs /tmp tmpfs defaults 0 0\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var mkdirCalled, mountCalled bool
	seams := DiskSeams{
		StatFn: func(_ string) (os.FileInfo, error) { return nil, os.ErrNotExist },
		MkdirFn: func(path string, perm os.FileMode) error {
			if path != "/groundseg-1" {
				t.Fatalf("expected /groundseg-1, got %q", path)
			}
			mkdirCalled = true
			return nil
		},
		MkfsExt4CommandFn: func(uuid, devPath string) error {
			if devPath != "/dev/sda" {
				t.Fatalf("unexpected device path %q", devPath)
			}
			return nil
		},
		OpenFn: func(path string) (*os.File, error) {
			if path != "/etc/fstab" {
				t.Fatalf("expected /etc/fstab, got %q", path)
			}
			return os.Open(in)
		},
		OpenFileFn: func(path string, _ int, _ os.FileMode) (*os.File, error) {
			if path != "/etc/fstab" {
				t.Fatalf("expected /etc/fstab, got %q", path)
			}
			return os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		},
		RunDiskCommandFn: func(string, ...string) (string, error) {
			return `{"blockdevices":[{"name":"sda","mountpoints":["/"]}]}`, nil
		},
		MountAllCommandFn: func() error {
			mountCalled = true
			return nil
		},
	}

	path, err := CreateGroundSegFilesystem("sda", seams)
	if err != nil {
		t.Fatalf("CreateGroundSegFilesystem error: %v", err)
	}
	if path != "/groundseg-1" {
		t.Fatalf("expected /groundseg-1, got %q", path)
	}
	if !mkdirCalled || !mountCalled {
		t.Fatalf("expected directory creation and mount, got mkdir=%v mount=%v", mkdirCalled, mountCalled)
	}

	content, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(content), "/groundseg-1 ext4 defaults,nofail 0 2") {
		t.Fatalf("expected reconciled fstab entry, got %q", content)
	}
}

func TestCreateGroundSegFilesystemRejectsPathLookupFailure(t *testing.T) {
	t.Parallel()

	lookupErr := errors.New("cannot stat")
	_, err := CreateGroundSegFilesystem("sda", DiskSeams{
		StatFn: func(_ string) (os.FileInfo, error) {
			return nil, lookupErr
		},
	})
	if !errors.Is(err, lookupErr) {
		t.Fatalf("expected lookup error, got %v", err)
	}
}
