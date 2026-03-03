package artifactwriter

import (
	"archive/tar"
	"bytes"
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
		return nil
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
	err := writeFileFn(filePath, []byte(opts.Content), opts.FileMode)
	if err == nil {
		return nil
	}

	if opts.DirectoryMode == 0 {
		opts.DirectoryMode = 0o755
	}
	dir := filepath.Dir(filePath)
	if mkdirErr := mkdirAllFn(dir, opts.DirectoryMode); mkdirErr != nil {
		return mkdirErr
	}
	err = writeFileFn(filePath, []byte(opts.Content), opts.FileMode)
	if err == nil {
		return nil
	}

	if opts.EnsureVolumesFn != nil {
		if err := opts.EnsureVolumesFn(); err != nil {
			return fmt.Errorf("failed to initialize artifact volumes: %w", err)
		}
	}

	if opts.CopyToVolumeFn == nil {
		return err
	}

	normalize := opts.NormalizeTarget
	if normalize == nil {
		normalize = NormalizeVolumeTargetPath
	}
	targetPath := normalize(opts.TargetPath)
	if err := opts.CopyToVolumeFn(filePath, targetPath, opts.VolumeName, opts.WriterContainerName, opts.SelectImageFn); err != nil {
		if opts.CopyErrorPrefix != "" {
			return fmt.Errorf("%s: %w", opts.CopyErrorPrefix, err)
		}
		return err
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
			return err
		}
		if exists {
			continue
		}
		if err := ops.CreateVolumeFn(name); err != nil {
			return err
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
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
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
		return nil, err
	}
	if _, err := io.Copy(tw, file); err != nil {
		if closeErr := tw.Close(); closeErr != nil {
			return nil, fmt.Errorf("close tar archive after copy failure: %w", closeErr)
		}
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return &buffer, nil
}
