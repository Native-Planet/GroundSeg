package routines

import (
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/structs"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	autoLinkInitialDelay  = 15 * time.Second
	autoLinkRetryInterval = 20 * time.Second
	autoLinkMaxAttempts   = 18
)

// AutoConfigureObjectStoreLinks links S3 storage for ships that are unlinked at startup.
// This is bounded and only targets ships unlinked at daemon boot, so user-initiated unlinks are respected.
func AutoConfigureObjectStoreLinks() {
	time.Sleep(autoLinkInitialDelay)

	conf := config.Conf()
	if !conf.WgRegistered || len(conf.Piers) == 0 {
		return
	}

	pending := make(map[string]struct{})
	for _, pier := range conf.Piers {
		if err := config.LoadUrbitConfig(pier); err != nil {
			zap.L().Warn(fmt.Sprintf("Skipping auto S3 link for %s: failed to load config: %v", pier, err))
			continue
		}
		shipConf := config.UrbitConf(pier)
		if !shipConf.MinIOLinked {
			pending[pier] = struct{}{}
		}
	}
	if len(pending) == 0 {
		return
	}

	zap.L().Info(fmt.Sprintf("Auto-linking S3 storage for %d ship(s) missing link configuration", len(pending)))
	for attempt := 1; attempt <= autoLinkMaxAttempts && len(pending) > 0; attempt++ {
		for pier := range pending {
			if err := ensureObjectStoreLinked(pier); err != nil {
				zap.L().Debug(fmt.Sprintf("Auto-link S3 skipped for %s (attempt %d/%d): %v", pier, attempt, autoLinkMaxAttempts, err))
				continue
			}
			delete(pending, pier)
			zap.L().Info(fmt.Sprintf("Auto-linked S3 storage for %s", pier))
		}
		if len(pending) > 0 {
			time.Sleep(autoLinkRetryInterval)
		}
	}

	if len(pending) > 0 {
		remaining := make([]string, 0, len(pending))
		for pier := range pending {
			remaining = append(remaining, pier)
		}
		sort.Strings(remaining)
		zap.L().Info(fmt.Sprintf("Auto-link S3 timed out for: %s", strings.Join(remaining, ", ")))
	}
}

func ensureObjectStoreLinked(patp string) error {
	if err := config.LoadUrbitConfig(patp); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	shipConf := config.UrbitConf(patp)
	if shipConf.MinIOLinked {
		return nil
	}

	objectStoreName := docker.GetObjectStoreContainerName(patp)
	objectStoreStatus, err := docker.GetContainerRunningStatus(objectStoreName)
	if err != nil {
		return fmt.Errorf("RustFS is unavailable: %w", err)
	}
	if !strings.Contains(objectStoreStatus, "Up") {
		return fmt.Errorf("RustFS is not running (%s)", objectStoreStatus)
	}

	shipStatus, err := docker.GetContainerRunningStatus(patp)
	if err != nil {
		return fmt.Errorf("ship is unavailable: %w", err)
	}
	if !strings.Contains(shipStatus, "Up") {
		return fmt.Errorf("ship is not running (%s)", shipStatus)
	}

	if _, err := click.GetLusCode(patp); err != nil {
		return fmt.Errorf("ship not booted yet: %w", err)
	}

	svcAccount, err := docker.CreateObjectStoreCredentials(patp)
	if err != nil {
		return fmt.Errorf("failed to create RustFS credentials: %w", err)
	}

	endpoint := strings.TrimSpace(shipConf.CustomS3Web)
	if endpoint == "" && strings.TrimSpace(shipConf.WgURL) != "" {
		endpoint = fmt.Sprintf("s3.%s", shipConf.WgURL)
	}
	if endpoint == "" {
		return fmt.Errorf("no S3 endpoint configured")
	}

	if err := click.LinkStorage(patp, endpoint, svcAccount); err != nil {
		return fmt.Errorf("failed to link storage: %w", err)
	}

	shipConf.MinIOLinked = true
	update := map[string]structs.UrbitDocker{patp: shipConf}
	if err := config.UpdateUrbitConfig(update); err != nil {
		return fmt.Errorf("failed to persist link status: %w", err)
	}
	return nil
}
