package routines

import (
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/structs"
	"net/url"
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

// AutoConfigureObjectStoreLinks performs one-time S3 link/relink for ships missing the RustFS link marker.
// Once marked, startup auto-link stops touching that ship and user-managed unlinks are respected.
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
		if !startramObjectStoreEnabled(shipConf) {
			continue
		}
		linkConfigured, err := docker.IsObjectStoreLinkConfigured(pier)
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Skipping auto S3 link for %s: failed to read link marker: %v", pier, err))
			continue
		}
		// One-time migration behavior:
		// - marker missing: auto-link/relink once
		// - marker present: no auto action (respects user-managed link state)
		if !linkConfigured {
			pending[pier] = struct{}{}
		}
	}
	if len(pending) == 0 {
		return
	}

	zap.L().Info(fmt.Sprintf("Auto-linking S3 storage for %d ship(s) missing link configuration", len(pending)))
	for attempt := 1; attempt <= autoLinkMaxAttempts && len(pending) > 0; attempt++ {
		for pier := range pending {
			linked, err := ensureObjectStoreLinked(pier)
			if err != nil {
				zap.L().Debug(fmt.Sprintf("Auto-link S3 skipped for %s (attempt %d/%d): %v", pier, attempt, autoLinkMaxAttempts, err))
				continue
			}
			delete(pending, pier)
			if linked {
				zap.L().Info(fmt.Sprintf("Auto-linked S3 storage for %s", pier))
			} else {
				zap.L().Info(fmt.Sprintf("Leaving existing S3 configuration untouched for %s", pier))
			}
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

func ensureObjectStoreLinked(patp string) (bool, error) {
	if err := config.LoadUrbitConfig(patp); err != nil {
		return false, fmt.Errorf("failed to load config: %w", err)
	}
	shipConf := config.UrbitConf(patp)
	if !startramObjectStoreEnabled(shipConf) {
		return false, nil
	}
	linkConfigured, err := docker.IsObjectStoreLinkConfigured(patp)
	if err != nil {
		return false, fmt.Errorf("failed to read link marker: %w", err)
	}
	if shipConf.MinIOLinked && linkConfigured {
		return false, nil
	}

	objectStoreName := docker.GetObjectStoreContainerName(patp)
	objectStoreStatus, err := docker.GetContainerRunningStatus(objectStoreName)
	if err != nil {
		return false, fmt.Errorf("RustFS is unavailable: %w", err)
	}
	if !strings.Contains(objectStoreStatus, "Up") {
		return false, fmt.Errorf("RustFS is not running (%s)", objectStoreStatus)
	}

	shipStatus, err := docker.GetContainerRunningStatus(patp)
	if err != nil {
		return false, fmt.Errorf("ship is unavailable: %w", err)
	}
	if !strings.Contains(shipStatus, "Up") {
		return false, fmt.Errorf("ship is not running (%s)", shipStatus)
	}

	if _, err := click.GetLusCode(patp); err != nil {
		return false, fmt.Errorf("ship not booted yet: %w", err)
	}

	currentEndpoint, err := click.GetStorageEndpoint(patp)
	if err != nil {
		return false, fmt.Errorf("failed to inspect existing storage config: %w", err)
	}
	if currentEndpoint != "" && !storageEndpointMatchesShip(currentEndpoint, shipConf) {
		if err := docker.MarkObjectStoreLinkConfigured(patp); err != nil {
			return false, fmt.Errorf("failed to persist object-store link marker: %w", err)
		}
		return false, nil
	}

	svcAccount, err := docker.CreateObjectStoreCredentials(patp)
	if err != nil {
		return false, fmt.Errorf("failed to create RustFS credentials: %w", err)
	}

	endpoint := strings.TrimSpace(shipConf.CustomS3Web)
	if endpoint == "" && strings.TrimSpace(shipConf.WgURL) != "" {
		endpoint = fmt.Sprintf("s3.%s", shipConf.WgURL)
	}
	if endpoint == "" {
		return false, fmt.Errorf("no S3 endpoint configured")
	}

	if err := click.LinkStorage(patp, endpoint, svcAccount); err != nil {
		return false, fmt.Errorf("failed to link storage: %w", err)
	}

	shipConf.MinIOLinked = true
	update := map[string]structs.UrbitDocker{patp: shipConf}
	if err := config.UpdateUrbitConfig(update); err != nil {
		return false, fmt.Errorf("failed to persist link status: %w", err)
	}
	if err := docker.MarkObjectStoreLinkConfigured(patp); err != nil {
		return false, fmt.Errorf("failed to persist object-store link marker: %w", err)
	}
	return true, nil
}

func startramObjectStoreEnabled(shipConf structs.UrbitDocker) bool {
	return shipConf.Network == "wireguard" && strings.TrimSpace(shipConf.WgURL) != ""
}

func storageEndpointMatchesShip(endpoint string, shipConf structs.UrbitDocker) bool {
	currentHost := normalizeStorageEndpointHost(endpoint)
	if currentHost == "" {
		return true
	}
	for expectedHost := range expectedStorageEndpointHosts(shipConf) {
		if currentHost == expectedHost {
			return true
		}
	}
	return false
}

func expectedStorageEndpointHosts(shipConf structs.UrbitDocker) map[string]struct{} {
	endpoints := make(map[string]struct{})
	if host := normalizeStorageEndpointHost(fmt.Sprintf("s3.%s", strings.TrimSpace(shipConf.WgURL))); host != "" {
		endpoints[host] = struct{}{}
	}
	if host := normalizeStorageEndpointHost(shipConf.CustomS3Web); host != "" {
		endpoints[host] = struct{}{}
	}
	return endpoints
}

func normalizeStorageEndpointHost(endpoint string) string {
	trimmed := strings.TrimSpace(strings.Trim(endpoint, "'"))
	if trimmed == "" {
		return ""
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err == nil {
		if host := strings.TrimSpace(parsed.Hostname()); host != "" {
			return strings.TrimSuffix(strings.ToLower(host), ".")
		}
	}
	withoutScheme := strings.TrimPrefix(strings.TrimPrefix(trimmed, "https://"), "http://")
	host := strings.SplitN(withoutScheme, "/", 2)[0]
	host = strings.SplitN(host, ":", 2)[0]
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}
