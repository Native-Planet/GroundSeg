package orchestration

import (
	"context"
	"fmt"
	"groundseg/docker/orchestration/internal/artifactwriter"
	"groundseg/dockerclient"

	"github.com/docker/docker/api/types/container"
	"go.uber.org/zap"
)

type volumeWriterImageSelector = func() (string, error)

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
		return fmt.Errorf("initialize docker client for %s: %w", writerContainerName, err)
	}
	defer cli.Close()

	image, err := selectImage()
	if err != nil {
		return fmt.Errorf("resolve writer image for %s: %w", writerContainerName, err)
	}

	// Best-effort cleanup in case a previous run left this writer container behind.
	_ = cli.ContainerRemove(ctx, writerContainerName, container.RemoveOptions{Force: true})

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: image,
	}, &container.HostConfig{
		Binds: []string{volumeName + ":" + targetPath},
	}, nil, nil, writerContainerName)
	if err != nil {
		return fmt.Errorf("create temporary writer container %s: %w", writerContainerName, err)
	}
	defer func() {
		if removeErr := cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true}); removeErr != nil {
			zap.L().Error(fmt.Sprintf("Failed to remove temporary container %s: %v", writerContainerName, removeErr))
		}
	}()

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("start temporary writer container %s: %w", writerContainerName, err)
	}

	tarReader, err := artifactwriter.TarArchiveForSingleFile(filePath)
	if err != nil {
		return fmt.Errorf("prepare archive for file %s: %w", filePath, err)
	}
	if err := cli.CopyToContainer(ctx, resp.ID, targetPath, tarReader, container.CopyToContainerOptions{}); err != nil {
		return fmt.Errorf("copy file %s to container %s path %s: %w", filePath, writerContainerName, targetPath, err)
	}
	return nil
}
