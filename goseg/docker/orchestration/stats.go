package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"groundseg/dockerclient"
	"groundseg/structs"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"go.uber.org/zap"
)

var (
	containers             = make(map[string]structs.ContainerStats)
	getMemoryUsageForStats = getMemoryUsage
	getDiskUsageForStats   = getDiskUsage
	nowForStats            = time.Now
)

func GetContainerStats(name string) structs.ContainerStats {
	// Attempt to get the container stats from the map
	if stats, exists := containers[name]; exists {
		// Check if the LastContact was at least 1 minute ago
		if time.Since(stats.LastContact) >= time.Minute {
			// More than 1 minute has passed, update stats
			stats.MemoryUsage = getMemoryUsageForStats(name)
			stats.DiskUsage = getDiskUsageForStats(name)
			stats.LastContact = nowForStats()
			containers[name] = stats // update the map with the new stats
		}
		// Return the stats (either updated or as they were)
		return stats
	} else {
		return ForceUpdateContainerStats(name)
	}
}

func ForceUpdateContainerStats(name string) structs.ContainerStats {
	// Container not found in map, get new stats
	memUsage := getMemoryUsageForStats(name)
	diskUsage := getDiskUsageForStats(name) // assuming getDiskUsage(name) returns an int64
	// Create new ContainerStats struct
	newStats := structs.ContainerStats{
		LastContact: nowForStats(),
		MemoryUsage: memUsage,
		DiskUsage:   diskUsage,
	}
	containers[name] = newStats // add the new stats to the map
	return newStats
}

// getMemoryUsage retrieves the memory usage of a container.
func getMemoryUsage(containerID string) uint64 {
	cli, err := dockerclient.New()
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to create Docker client: %v", err))
		return 0
	}
	defer func() {
		if closeErr := cli.Close(); closeErr != nil {
			zap.L().Warn(fmt.Sprintf("Failed to close Docker client after memory usage lookup: %v", closeErr))
		}
	}()

	resp, err := cli.ContainerStats(context.Background(), containerID, false)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to get container stats: %v", err))
		return 0
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			zap.L().Warn(fmt.Sprintf("Failed to close container stats response: %v", closeErr))
		}
	}()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to read container stats: %v", err))
		return 0
	}

	var stats container.StatsResponse
	err = json.Unmarshal(data, &stats)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to unmarshal container stats: %v", err))
		return 0
	}

	return stats.MemoryStats.Usage
}

// getDiskUsage calculates the disk usage by totaling the space used by volumes of the container.
func getDiskUsage(containerID string) int64 {
	cli, err := dockerclient.New()
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to create Docker client: %v", err))
		return 0
	}
	defer func() {
		if closeErr := cli.Close(); closeErr != nil {
			zap.L().Warn(fmt.Sprintf("Failed to close Docker client after disk usage lookup: %v", closeErr))
		}
	}()

	inspect, err := cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to inspect container: %v", err))
		return 0
	}

	var totalSize int64
	for _, mount := range inspect.Mounts {
		if mount.Type == "volume" || mount.Type == "bind" {
			size, err := getDirSize(mount.Source)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Failed to get size for directory %s: %v", mount.Source, err))
				continue // Continue calculating the size of other volumes even if one fails
			}
			totalSize += size
		}
	}

	return totalSize
}

func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}
