package lifecycle

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"groundseg/structs"
)

type containerPlan struct {
	Name         string
	Type         string
	Config       container.Config
	HostConfig   container.HostConfig
	ImageInfo    map[string]string
	DesiredImage string
}

func (runtime *Runtime) containerPlanFor(containerName string, containerType string) (containerPlan, error) {
	plan := containerPlan{Name: containerName, Type: containerType}
	containerConfig, hostConfig, err := runtime.containerConfigResolver(containerName, containerType)
	if err != nil {
		return plan, fmt.Errorf("resolve container config for %s/%s: %w", containerName, containerType, err)
	}
	plan.Config = containerConfig
	plan.HostConfig = hostConfig

	imageInfo, err := runtime.getLatestContainerInfoFn(containerType)
	if err != nil {
		return plan, fmt.Errorf("lookup latest image for container type %s: %w", containerType, err)
	}
	plan.ImageInfo = imageInfo
	plan.DesiredImage = fmt.Sprintf("%s:%s@sha256:%s", imageInfo["repo"], imageInfo["tag"], imageInfo["hash"])
	if _, err := runtime.pullImageIfNotExistFn(plan.DesiredImage, imageInfo); err != nil {
		return plan, fmt.Errorf("ensure image exists %s: %w", plan.DesiredImage, err)
	}
	return plan, nil
}

func createAndStartContainer(ctx context.Context, cli *client.Client, plan containerPlan) error {
	if _, err := cli.ContainerCreate(ctx, &plan.Config, &plan.HostConfig, nil, nil, plan.Name); err != nil {
		return fmt.Errorf("create container %s: %w", plan.Name, err)
	}
	if err := cli.ContainerStart(ctx, plan.Name, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container %s: %w", plan.Name, err)
	}
	return nil
}

func recreateContainerIfImageChanged(ctx context.Context, cli *client.Client, plan containerPlan, currentImage string) error {
	digestParts := strings.Split(currentImage, "@sha256:")
	currentDigest := ""
	if len(digestParts) > 1 {
		currentDigest = digestParts[1]
	}
	if currentDigest == plan.ImageInfo["hash"] {
		return nil
	}
	if plan.Type == "vere" {
		gracefulTimeout := 60
		stopOpts := container.StopOptions{Timeout: &gracefulTimeout}
		zap.L().Info(fmt.Sprintf("Gracefully stopping %s (60s timeout) before update", plan.Name))
		if err := cli.ContainerStop(ctx, plan.Name, stopOpts); err != nil {
			zap.L().Warn(fmt.Sprintf("Graceful stop failed for %s: %v, forcing removal", plan.Name, err))
		}
	}
	if err := cli.ContainerRemove(ctx, plan.Name, container.RemoveOptions{Force: true}); err != nil {
		zap.L().Warn(fmt.Sprintf("Couldn't remove container %v (may not exist yet): %v", plan.Name, err))
	}
	if err := createAndStartContainer(ctx, cli, plan); err != nil {
		return fmt.Errorf("recreate container %s: %w", plan.Name, err)
	}
	zap.L().Info(fmt.Sprintf("Restarted %s with image %s", plan.Name, plan.DesiredImage))
	return nil
}

func ensureRunningContainer(runtime *Runtime, ctx context.Context, cli *client.Client, plan containerPlan) error {
	existingContainer, err := runtime.FindContainer(plan.Name)
	if err != nil {
		if !isContainerLookupNotFound(err) {
			return fmt.Errorf("lookup container %s: %w", plan.Name, err)
		}
		existingContainer = nil
	}
	switch {
	case existingContainer == nil:
		if err := createAndStartContainer(ctx, cli, plan); err != nil {
			return fmt.Errorf("start container %s: %w", plan.Name, err)
		}
		zap.L().Info(fmt.Sprintf("%s started with image %s", plan.Name, plan.DesiredImage))
	case existingContainer.State == "exited":
		if err := cli.ContainerRemove(ctx, plan.Name, container.RemoveOptions{Force: true}); err != nil {
			return fmt.Errorf("remove exited container %s: %w", plan.Name, err)
		}
		if err := createAndStartContainer(ctx, cli, plan); err != nil {
			return fmt.Errorf("restart exited container %s: %w", plan.Name, err)
		}
		zap.L().Info(fmt.Sprintf("Started stopped container %s", plan.Name))
	case existingContainer.State == "created":
		if err := cli.ContainerStart(ctx, plan.Name, container.StartOptions{}); err != nil {
			return fmt.Errorf("start created container %s: %w", plan.Name, err)
		}
		zap.L().Info(fmt.Sprintf("Started created container %s", plan.Name))
	default:
		if err := recreateContainerIfImageChanged(ctx, cli, plan, existingContainer.Image); err != nil {
			return fmt.Errorf("reconcile existing container %s: %w", plan.Name, err)
		}
	}
	return nil
}

func ensureCreatedContainer(ctx context.Context, cli *client.Client, plan containerPlan) error {
	_, err := cli.ContainerCreate(ctx, &plan.Config, &plan.HostConfig, nil, nil, plan.Name)
	if err != nil {
		return fmt.Errorf("create container %s: %w", plan.Name, err)
	}
	return nil
}

func containerStateFromInspect(plan containerPlan, desiredStatus string, containerDetails container.InspectResponse) structs.ContainerState {
	return structs.ContainerState{
		ID:            containerDetails.ID,
		Name:          plan.Name,
		Image:         plan.DesiredImage,
		Type:          plan.Type,
		DesiredStatus: desiredStatus,
		ActualStatus:  containerDetails.State.Status,
		CreatedAt:     containerDetails.Created,
		Config:        plan.Config,
		Host:          plan.HostConfig,
	}
}
