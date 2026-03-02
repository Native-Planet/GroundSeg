package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/disk"
)

func checkIsEMMCMachine() bool {
	partitions, err := disk.Partitions(true)
	if err != nil {
		fmt.Println("Failed to get partitions, defaulting to non-eMMC assumption")
		return false
	}

	// Check if root partition is on eMMC
	for _, p := range partitions {
		if p.Mountpoint == "/" {
			return strings.Contains(p.Device, "mmc")
		}
	}

	// If we can't find the root partition, check if /media/data exists as a fallback
	if _, err := os.Stat("/media/data"); err == nil {
		return true
	}

	return false
}

func GetStoragePath(operation string) (string, error) {
	basePath := os.Getenv("GS_BASE_PATH")
	if basePath == "" {
		basePath = "/opt/nativeplanet/groundseg"
	}
	if !strings.HasPrefix(basePath, "/") {
		fmt.Println("Base path is not absolute! Using default")
		basePath = "/opt/nativeplanet/groundseg"
	}
	var operationPaths = map[string]string{
		"uploads":     "uploads",
		"temp":        "temp",
		"exports":     "exports",
		"logs":        "logs",
		"bug-reports": "bug-reports",
	}
	opPath, exists := operationPaths[operation]
	if !exists {
		return "", fmt.Errorf("invalid storage operation: %s", operation)
	}
	var storagePath string
	if isEMMCMachine {
		storagePath = filepath.Join("/media/data", opPath)
		if _, err := os.Stat("/media/data"); os.IsNotExist(err) {
			fmt.Printf("/media/data not found, falling back to %s\n", basePath)
			storagePath = filepath.Join(basePath, opPath)
		}
	} else {
		storagePath = filepath.Join(basePath, opPath)
	}
	if err := mkdirAllFn(storagePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create storage directory: %w", err)
	}

	return storagePath, nil
}
