package docker

import (
	"context"
	"encoding/json"
	"groundseg/logger"
	"groundseg/structs"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var (
	containers = make(map[string]structs.ContainerStats)
)

func GetContainerStats(name string) structs.ContainerStats {
	// Attempt to get the container stats from the map
	if stats, exists := containers[name]; exists {
		// Check if the LastContact was at least 1 minute ago
		if time.Since(stats.LastContact) >= 10*time.Second {
			// More than 1 minute has passed, update stats
			stats.MemoryUsage = getMemoryUsage(name)
			stats.DiskUsage = getDiskUsage(name)
			stats.LastContact = time.Now()
			containers[name] = stats // update the map with the new stats
		}
		// Return the stats (either updated or as they were)
		return stats
	} else {
		// Container not found in map, get new stats
		memUsage := getMemoryUsage(name)
		diskUsage := getDiskUsage(name) // assuming getDiskUsage(name) returns an int64
		// Create new ContainerStats struct
		newStats := structs.ContainerStats{
			LastContact: time.Now(),
			MemoryUsage: memUsage,
			DiskUsage:   diskUsage,
		}
		containers[name] = newStats // add the new stats to the map
		return newStats
	}
}

// getMemoryUsage retrieves the memory usage of a container.
func getMemoryUsage(containerID string) uint64 {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Logger.Error("Failed to create Docker client: ", err)
		return 0
	}
	defer cli.Close()

	resp, err := cli.ContainerStats(context.Background(), containerID, false)
	if err != nil {
		logger.Logger.Error("Failed to get container stats: ", err)
		return 0
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Logger.Error("Failed to read container stats: ", err)
		return 0
	}

	var stats types.StatsJSON
	err = json.Unmarshal(data, &stats)
	if err != nil {
		logger.Logger.Error("Failed to unmarshal container stats: ", err)
		return 0
	}

	return stats.MemoryStats.Usage
}

// getDiskUsage calculates the disk usage by totaling the space used by volumes of the container.
func getDiskUsage(containerID string) int64 {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Logger.Error("Failed to create Docker client: ", err)
		return 0
	}
	defer cli.Close()

	inspect, err := cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		logger.Logger.Error("Failed to inspect container: ", err)
		return 0
	}

	var totalSize int64
	for _, mount := range inspect.Mounts {
		if mount.Type == "volume" || mount.Type == "bind" {
			size, err := getDirSize(mount.Source)
			if err != nil {
				logger.Logger.Error("Failed to get size for directory: ", mount.Source, " Error: ", err)
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
