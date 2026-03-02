package artifactwriter

import (
	"archive/tar"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteWritesFileWhenWritable(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "ok", "config.txt")
	opts := WriteConfig{
		FilePath:      output,
		Content:       "content",
		FileMode:      0644,
		DirectoryMode: 0755,
	}
	if err := Write(opts); err != nil {
		t.Fatalf("Write returned unexpected error: %v", err)
	}

	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("expected output file to be written: %v", err)
	}
	if string(got) != "content" {
		t.Fatalf("unexpected file content: %q", string(got))
	}
}

func TestWriteFallsBackToVolumeCopyOnWriteFailure(t *testing.T) {
	t.Parallel()

	var ensureVolumesCalled bool
	var copyCalled bool
	var gotPath string
	var gotTarget string
	var gotVolume string
	var gotWriter string
	var gotImage string

	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "netdata.conf")
	opts := WriteConfig{
		FilePath:      output,
		Content:       "content",
		FileMode:      0644,
		DirectoryMode: 0755,
		WriteFileFn: func(string, []byte, os.FileMode) error {
			return errors.New("readonly filesystem")
		},
		MkdirAllFn: func(string, os.FileMode) error { return nil },
		EnsureVolumesFn: func() error {
			ensureVolumesCalled = true
			return nil
		},
		CopyToVolumeFn: func(path string, target string, volume string, writer string, selectImage func() (string, error)) error {
			copyCalled = true
			gotPath = path
			gotTarget = target
			gotVolume = volume
			gotWriter = writer
			image, err := selectImage()
			if err != nil {
				return err
			}
			gotImage = image
			return nil
		},
		TargetPath:          "/etc/netdata",
		VolumeName:          "netdata",
		WriterContainerName: "netdata_writer",
		SelectImageFn: func() (string, error) {
			return "netdata:latest", nil
		},
		CopyErrorPrefix: "copy failed",
	}
	if err := Write(opts); err != nil {
		t.Fatalf("expected fallback copy to succeed, got %v", err)
	}

	if !ensureVolumesCalled {
		t.Fatal("expected volume initialization to run on fallback")
	}
	if !copyCalled {
		t.Fatal("expected copy-to-volume writer fallback")
	}
	if gotPath != output {
		t.Fatalf("unexpected copied path: %s", gotPath)
	}
	if gotTarget != "/etc/netdata/" {
		t.Fatalf("expected normalized target path, got %q", gotTarget)
	}
	if gotVolume != "netdata" || gotWriter != "netdata_writer" {
		t.Fatalf("unexpected volume/writer: %s %s", gotVolume, gotWriter)
	}
	if gotImage != "netdata:latest" {
		t.Fatalf("unexpected selected image: %s", gotImage)
	}
}

func TestWriteReturnsWriteErrorWithoutCopyWriter(t *testing.T) {
	t.Parallel()

	errWrite := errors.New("write failed")
	err := Write(WriteConfig{
		FilePath:       filepath.Join(t.TempDir(), "netdata.conf"),
		Content:        "content",
		FileMode:       0644,
		DirectoryMode:  0755,
		WriteFileFn:    func(string, []byte, os.FileMode) error { return errWrite },
		MkdirAllFn:     func(string, os.FileMode) error { return nil },
		CopyToVolumeFn: nil,
	})
	if err != errWrite {
		t.Fatalf("expected original write error, got %v", err)
	}
}

func TestEnsureContainerVolumesCreatesMissingVolumes(t *testing.T) {
	t.Parallel()

	ops := VolumeOps{
		VolumeExistsFn: func(name string) (bool, error) {
			return name == "existing", nil
		},
		CreateVolumeFn: func(name string) error {
			if name == "missing" {
				return nil
			}
			return nil
		},
	}
	if err := EnsureContainerVolumes(ops, "existing", "missing"); err != nil {
		t.Fatalf("EnsureContainerVolumes returned unexpected error: %v", err)
	}
}

func TestEnsureContainerVolumesSkipsWithMissingOps(t *testing.T) {
	t.Parallel()

	if err := EnsureContainerVolumes(VolumeOps{}, "netdata"); err == nil {
		t.Fatal("expected missing volume operations to return an error")
	}
}

func TestNormalizeVolumeTargetPath(t *testing.T) {
	t.Parallel()

	if got := NormalizeVolumeTargetPath(""); got != "/" {
		t.Fatalf("empty path should normalize to root, got %q", got)
	}
	if got := NormalizeVolumeTargetPath("/etc"); got != "/etc/" {
		t.Fatalf("expected trailing slash to be appended, got %q", got)
	}
	if got := NormalizeVolumeTargetPath("/var/lib/"); got != "/var/lib/" {
		t.Fatalf("expected existing slash to be preserved, got %q", got)
	}
}

func TestTarArchiveForSingleFileIncludesContent(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	source := filepath.Join(tmp, "source.txt")
	if err := os.WriteFile(source, []byte("hello tar"), 0644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	reader, err := TarArchiveForSingleFile(source)
	if err != nil {
		t.Fatalf("TarArchiveForSingleFile failed: %v", err)
	}

	tarReader := tar.NewReader(reader)
	header, err := tarReader.Next()
	if err != nil {
		t.Fatalf("reading archive header: %v", err)
	}
	if header.Name != filepath.Base(source) {
		t.Fatalf("unexpected archive name: %q", header.Name)
	}
	got, err := io.ReadAll(tarReader)
	if err != nil {
		t.Fatalf("reading archive content: %v", err)
	}
	if string(got) != "hello tar" {
		t.Fatalf("unexpected archive content: %q", string(got))
	}
}
