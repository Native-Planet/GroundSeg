package orchestration

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"groundseg/dockerclient"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"go.uber.org/zap"
)

type volumeWriterImageSelector func() (string, error)

func copyFileToVolumeWithTempContainer(
	filePath string,
	targetPath string,
	volumeName string,
	writerContainerName string,
	selectImage volumeWriterImageSelector,
) error {
	ctx := context.Background()
	cli, err := dockerclient.New()
	if err != nil {
		return err
	}
	defer cli.Close()

	image, err := selectImage()
	if err != nil {
		return err
	}

	// Best-effort cleanup in case a previous run left this writer container behind.
	_ = cli.ContainerRemove(ctx, writerContainerName, container.RemoveOptions{Force: true})

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: image,
	}, &container.HostConfig{
		Binds: []string{volumeName + ":" + targetPath},
	}, nil, nil, writerContainerName)
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true}); removeErr != nil {
			zap.L().Error(fmt.Sprintf("Failed to remove temporary container %s: %v", writerContainerName, removeErr))
		}
	}()

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}

	tarReader, err := tarArchiveForSingleFile(filePath)
	if err != nil {
		return err
	}
	if err := cli.CopyToContainer(ctx, resp.ID, targetPath, tarReader, container.CopyToContainerOptions{}); err != nil {
		return err
	}
	return nil
}

func latestContainerImage(containerType string) (string, error) {
	containerInfo, err := GetLatestContainerInfo(containerType)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"]), nil
}

func tarArchiveForSingleFile(filePath string) (io.Reader, error) {
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
		_ = tw.Close()
		return nil, err
	}
	if _, err := io.Copy(tw, file); err != nil {
		_ = tw.Close()
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return &buffer, nil
}
