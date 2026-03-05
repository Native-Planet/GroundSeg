package artifactwriter

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type VolumeOps struct {
	VolumeExistsFn func(string) (bool, error)
	CreateVolumeFn func(string) error
}

type VolumeInitializationPlan struct {
	ops         VolumeOps
	volumeNames []string
}

func NewVolumeInitializationPlan(ops VolumeOps, volumeNames ...string) VolumeInitializationPlan {
	return VolumeInitializationPlan{ops: ops, volumeNames: volumeNames}
}

func (plan VolumeInitializationPlan) EnsureVolumes() error {
	if len(plan.volumeNames) == 0 {
		return nil
	}
	if plan.ops.VolumeExistsFn == nil || plan.ops.CreateVolumeFn == nil {
		return fmt.Errorf("volume initialization requires volume existence and create callbacks")
	}
	return EnsureContainerVolumes(plan.ops, plan.volumeNames...)
}

type WriteConfig struct {
	FilePath            string
	Content             string
	FileMode            os.FileMode
	DirectoryMode       os.FileMode
	WriteFileFn         func(string, []byte, os.FileMode) error
	MkdirAllFn          func(string, os.FileMode) error
	EnsureVolumesFn     func() error
	CopyToVolumeFn      func(string, string, string, string, func() (string, error)) error
	TargetPath          string
	VolumeName          string
	WriterContainerName string
	SelectImageFn       func() (string, error)
	CopyErrorPrefix     string
	NormalizeTarget     func(string) string
}

func Write(opts WriteConfig) error {
	writeFileFn := opts.WriteFileFn
	if writeFileFn == nil {
		writeFileFn = os.WriteFile
	}

	mkdirAllFn := opts.MkdirAllFn
	if mkdirAllFn == nil {
		mkdirAllFn = os.MkdirAll
	}

	filePath := opts.FilePath
	writeErr := writeFileFn(filePath, []byte(opts.Content), opts.FileMode)
	if writeErr == nil {
		return nil
	}

	if opts.DirectoryMode == 0 {
		opts.DirectoryMode = 0o755
	}
	dir := filepath.Dir(filePath)
	mkdirErr := mkdirAllFn(dir, opts.DirectoryMode)
	if mkdirErr != nil {
		return errors.Join(writeErr, mkdirErr)
	}

	writeErr = writeFileFn(filePath, []byte(opts.Content), opts.FileMode)
	if writeErr == nil {
		return nil
	}

	if opts.EnsureVolumesFn != nil {
		if opts.SelectImageFn == nil && (opts.CopyToVolumeFn != nil) {
			return fmt.Errorf("artifact write requires image selector callback when copy fallback is configured")
		}
		if err := opts.EnsureVolumesFn(); err != nil {
			return errors.Join(writeErr, fmt.Errorf("failed to initialize artifact volumes: %w", err))
		}
	}

	if opts.CopyToVolumeFn == nil {
		return writeErr
	}

	normalize := opts.NormalizeTarget
	if normalize == nil {
		normalize = NormalizeVolumeTargetPath
	}
	targetPath := normalize(opts.TargetPath)
	if err := opts.CopyToVolumeFn(filePath, targetPath, opts.VolumeName, opts.WriterContainerName, opts.SelectImageFn); err != nil {
		if opts.CopyErrorPrefix != "" {
			err = fmt.Errorf("%s: %w", opts.CopyErrorPrefix, err)
		}
		return errors.Join(writeErr, err)
	}

	return nil
}

func EnsureContainerVolumes(ops VolumeOps, volumeNames ...string) error {
	if ops.VolumeExistsFn == nil || ops.CreateVolumeFn == nil {
		return fmt.Errorf("container volume ops not configured")
	}
	for _, name := range volumeNames {
		exists, err := ops.VolumeExistsFn(name)
		if err != nil {
			return fmt.Errorf("check docker volume %q: %w", name, err)
		}
		if exists {
			continue
		}
		if err := ops.CreateVolumeFn(name); err != nil {
			return fmt.Errorf("create docker volume %q: %w", name, err)
		}
	}
	return nil
}

func NormalizeVolumeTargetPath(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasSuffix(path, "/") {
		return path + "/"
	}
	return path
}

func TarArchiveForSingleFile(filePath string) (io.Reader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open artifact file %q: %w", filePath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat artifact file %q: %w", filePath, err)
	}

	var buffer bytes.Buffer
	tw := tar.NewWriter(&buffer)
	header := &tar.Header{
		Name: filepath.Base(filePath),
		Mode: int64(info.Mode().Perm()),
		Size: info.Size(),
	}
	if err := tw.WriteHeader(header); err != nil {
		if closeErr := tw.Close(); closeErr != nil {
			return nil, fmt.Errorf("close tar archive after header write failure: %w", closeErr)
		}
		return nil, fmt.Errorf("write tar header for %q: %w", filePath, err)
	}
	if _, err := io.Copy(tw, file); err != nil {
		if closeErr := tw.Close(); closeErr != nil {
			return nil, fmt.Errorf("close tar archive after copy failure: %w", closeErr)
		}
		return nil, fmt.Errorf("copy artifact file %q into tar stream: %w", filePath, err)
	}
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("close tar writer for %q: %w", filePath, err)
	}
	return &buffer, nil
}
