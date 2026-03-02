package lifecycle

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (runtime *Runtime) DockerPoller() {
	ticker := time.NewTicker(runtime.dockerPollInterval)
	defer ticker.Stop()
	for {
		<-ticker.C
		zap.L().Info("polling docker")
		if err := runtime.dockerPollerTickFn(); err != nil {
			zap.L().Error(fmt.Sprintf("Docker poller tick failed: %v", err))
		}
	}
}

func (runtime *Runtime) runDockerPollerTick() error {
	if err := runtime.ensureMonitoredContainerHealthy("netdata", "netdata"); err != nil {
		return err
	}
	return nil
}

func (runtime *Runtime) ensureMonitoredContainerHealthy(containerName, containerType string) error {
	status, err := runtime.getContainerRunningStatusFn(containerName)
	if err == nil {
		if strings.HasPrefix(status, "Up") {
			return nil
		}
		zap.L().Warn(fmt.Sprintf("Container %s is not running (%q); attempting restart", containerName, status))
		if _, err := runtime.startContainerFn(containerName, containerType); err != nil {
			return fmt.Errorf("failed to restart container %s: %w", containerName, err)
		}
		return nil
	}
	if !isContainerLookupNotFound(err) {
		return fmt.Errorf("container status check failed for %s: %w", containerName, err)
	}
	zap.L().Info(fmt.Sprintf("Container %s is not found; attempting start", containerName))
	if _, err := runtime.startContainerFn(containerName, containerType); err != nil {
		return fmt.Errorf("failed to start container %s: %w", containerName, err)
	}
	return nil
}
