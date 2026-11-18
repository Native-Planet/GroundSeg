package config

// functions related to managing urbit config jsons & corresponding structs

import (
	"context"
	"encoding/json"
	"fmt"
	"groundseg/defaults"
	"groundseg/dockerclient"
	"groundseg/structs"

	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

var (
	UrbitsConfig = make(map[string]structs.UrbitDocker)
	urbitMutex   sync.RWMutex
)

// retrieve struct corresponding with urbit json file
func UrbitConf(pier string) structs.UrbitDocker {
	urbitMutex.Lock()
	defer urbitMutex.Unlock()
	return UrbitsConfig[pier]
}

// this should eventually be a click/conn.c command
func GetMinIOLinkedStatus(patp string) bool {
	urbConf := UrbitConf(patp)
	return urbConf.MinIOLinked
}

// retrieve map of urbit config structs
func UrbitConfAll() map[string]structs.UrbitDocker {
	urbitMutex.Lock()
	defer urbitMutex.Unlock()
	return UrbitsConfig
}

// load urbit conf json into memory
func LoadUrbitConfig(pier string) error {
	urbitMutex.Lock()
	defer urbitMutex.Unlock()
	// pull docker info from json
	confPath := filepath.Join(BasePath, "settings", "pier", pier+".json")
	file, err := os.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("Unable to load %s config: %w", pier, err)
		// todo: write a new conf
	}
	// Unmarshal JSON
	var targetStruct structs.UrbitDocker
	if err := json.Unmarshal(file, &targetStruct); err != nil {
		return fmt.Errorf("Error decoding %s JSON: %w", pier, err)
	}
	applyUrbitDefaults(&targetStruct)
	structs.SyncCustomS3Domains(&targetStruct)
	// Store in var
	UrbitsConfig[pier] = targetStruct
	return nil
}

// Delete urbit config entry
func RemoveUrbitConfig(pier string) error {
	urbitMutex.Lock()
	defer urbitMutex.Unlock()
	path := filepath.Join(BasePath, "settings", "pier", pier+".json")
	if err := os.Remove(path); err != nil {
		return err
	}
	if err := syncDirectory(filepath.Dir(path)); err != nil {
		return fmt.Errorf("sync pier config directory: %w", err)
	}
	delete(UrbitsConfig, pier)
	return nil
}

// update the in-memory struct and save it to json
func UpdateUrbitConfig(inputConfig map[string]structs.UrbitDocker) error {
	urbitMutex.Lock()
	defer urbitMutex.Unlock()
	// update UrbitsConfig with the values from inputConfig
	for pier, config := range inputConfig {
		applyUrbitDefaults(&config)
		structs.SyncCustomS3Domains(&config)
		ver, err := getImageTagByContainerName(pier)
		if err == nil {
			config.UrbitVersion = ver
		}
		// also update the corresponding json files
		path := filepath.Join(BasePath, "settings", "pier", pier+".json")
		encoded, err := json.MarshalIndent(&config, "", "    ")
		if err != nil {
			return fmt.Errorf("error encoding config: %v", err)
		}
		if len(encoded) == 0 {
			return fmt.Errorf("refusing to persist empty configuration for pier %s", pier)
		}
		encoded = append(encoded, '\n')
		if err := writeFileDurably(path, encoded, 0o644); err != nil {
			return fmt.Errorf("error persisting configuration for pier %s: %v", pier, err)
		}
		UrbitsConfig[pier] = config
	}
	return nil
}

func ReplaceUrbitConfigJSON(pier string, raw []byte) ([]byte, error) {
	var configMap map[string]any
	if err := json.Unmarshal(raw, &configMap); err != nil {
		return nil, fmt.Errorf("invalid %s config JSON: %v", pier, err)
	}
	if len(configMap) == 0 {
		return nil, fmt.Errorf("refusing to persist empty configuration for pier %s", pier)
	}
	formatted, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("error encoding %s config: %v", pier, err)
	}
	var targetStruct structs.UrbitDocker
	if err := unmarshalUrbitDockerSafe(formatted, &targetStruct); err != nil {
		return nil, err
	}
	if targetStruct.PierName != "" && targetStruct.PierName != pier {
		return nil, fmt.Errorf("pier_name %q does not match %q", targetStruct.PierName, pier)
	}
	applyUrbitDefaults(&targetStruct)
	structs.SyncCustomS3Domains(&targetStruct)

	urbitMutex.Lock()
	defer urbitMutex.Unlock()
	path := filepath.Join(BasePath, "settings", "pier", pier+".json")
	if len(formatted) == 0 {
		return nil, fmt.Errorf("refusing to persist empty configuration for pier %s", pier)
	}
	if err := writeFileDurably(path, append(formatted, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("error persisting configuration for pier %s: %v", pier, err)
	}
	UrbitsConfig[pier] = targetStruct
	return formatted, nil
}

func unmarshalUrbitDockerSafe(data []byte, target *structs.UrbitDocker) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("invalid urbit config: %v", r)
		}
	}()
	return json.Unmarshal(data, target)
}

func applyUrbitDefaults(target *structs.UrbitDocker) {
	if target.StartramReminder == nil {
		target.StartramReminder = defaults.UrbitConfig.StartramReminder
	}
	if target.SnapTime == 0 {
		target.SnapTime = defaults.UrbitConfig.SnapTime
	}
	if !structs.IsValidVereBits(target.VereBits) {
		target.VereBits = defaults.UrbitConfig.VereBits
	}
}

func UpdateUrbitConfigForPier(pier string, mutate func(*structs.UrbitDocker)) error {
	if err := LoadUrbitConfig(pier); err != nil {
		return err
	}
	urbConf := UrbitConf(pier)
	mutate(&urbConf)
	return UpdateUrbitConfig(map[string]structs.UrbitDocker{pier: urbConf})
}

func getImageTagByContainerName(containerName string) (string, error) {
	ctx := context.Background()

	// Create a new Docker client
	cli, err := dockerclient.New()
	if err != nil {
		return "", fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	// Set up a filter to search for the container by name using a filter
	filterArgs := filters.NewArgs()
	filterArgs.Add("name", containerName)

	// List containers using the filter
	containers, err := cli.ContainerList(ctx, container.ListOptions{Filters: filterArgs, All: true})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	// Check if any container matches the exact given name
	for _, container := range containers {
		for _, name := range container.Names {
			// Docker names are prefixed with "/", so we need to trim it
			if strings.TrimPrefix(name, "/") == containerName {
				// Extract the image tag from the container's image name
				imageParts := strings.Split(container.Image, ":")
				if len(imageParts) > 1 {
					return strings.Split(imageParts[1], "@")[0], nil
				}
				return "latest", nil // Default tag if no specific tag is found
			}
		}
	}

	return "", fmt.Errorf("no exact match found for container with name %s", containerName)
}
