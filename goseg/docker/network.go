package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"go.uber.org/zap"
)

// removeStaleNetworkContainer removes a container whose shared network belongs
// to a different container than the desired target. Its volumes remain intact.
// The caller recreates it and decides whether it should start.
func removeStaleNetworkContainer(ctx context.Context, cli *client.Client, name string, desired container.NetworkMode) (bool, error) {
	if !desired.IsContainer() {
		return false, nil
	}
	existing, err := cli.ContainerInspect(ctx, name)
	if errdefs.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect network for %s: %w", name, err)
	}
	// Resolve the desired target before changing anything. A missing or
	// unavailable WireGuard container must not cause a ship to be removed.
	target, err := cli.ContainerInspect(ctx, desired.ConnectedContainer())
	if err != nil {
		return false, fmt.Errorf("inspect network target for %s: %w", name, err)
	}
	if existing.HostConfig == nil || existing.State == nil || target.ID == "" {
		return false, fmt.Errorf("incomplete network inspection for %s", name)
	}
	current := existing.HostConfig.NetworkMode
	if current.IsContainer() {
		attached, err := cli.ContainerInspect(ctx, current.ConnectedContainer())
		if err != nil && !errdefs.IsNotFound(err) {
			return false, fmt.Errorf("inspect attached network for %s: %w", name, err)
		}
		if err == nil && attached.ID == target.ID {
			return false, nil
		}
	}
	if existing.State.Running {
		timeout := 60
		if err := cli.ContainerStop(ctx, existing.ID, container.StopOptions{Timeout: &timeout}); err != nil {
			return false, fmt.Errorf("stop %s to repair network: %w", name, err)
		}
	}
	if err := cli.ContainerRemove(ctx, existing.ID, container.RemoveOptions{}); err != nil {
		return false, fmt.Errorf("remove %s to repair network: %w", name, err)
	}
	zap.L().Info("Recreating container to attach to current network target", zap.String("container", name), zap.String("target", target.ID))
	return true, nil
}
