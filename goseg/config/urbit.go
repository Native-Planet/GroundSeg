package config

// functions related to managing urbit config jsons & corresponding structs

import (
	"context"
	"encoding/json"
	"fmt"
	"groundseg/defaults"
	"groundseg/dockerclient"
	"groundseg/structs"
	"io/ioutil"
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
	file, err := ioutil.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("Unable to load %s config: %w", pier, err)
		// todo: write a new conf
	}
	// Unmarshal JSON
	var targetStruct structs.UrbitDocker
	if err := json.Unmarshal(file, &targetStruct); err != nil {
		return fmt.Errorf("Error decoding %s JSON: %w", pier, err)
	}
	// set startram reminder
	if targetStruct.StartramReminder == nil {
		targetStruct.StartramReminder = defaults.UrbitConfig.StartramReminder
	}
	if targetStruct.SnapTime == 0 {
		targetStruct.SnapTime = 60
	}
	// Store in var
	UrbitsConfig[pier] = targetStruct
	return nil
}

// Delete urbit config entry
func RemoveUrbitConfig(pier string) error {
	// remove from memory
	urbitMutex.Lock()
	defer urbitMutex.Unlock()
	delete(UrbitsConfig, pier)
	// remove from disk
	err := os.Remove(filepath.Join(BasePath, "settings", "pier", pier+".json"))
	return err
}

// update the in-memory struct and save it to json
func UpdateUrbitConfig(inputConfig map[string]structs.UrbitDocker) error {
	urbitMutex.Lock()
	defer urbitMutex.Unlock()
	// update UrbitsConfig with the values from inputConfig
	for pier, config := range inputConfig {
		ver, err := getImageTagByContainerName(pier)
		if err == nil {
			config.UrbitVersion = ver
		}
		UrbitsConfig[pier] = config
		// also update the corresponding json files
		path := filepath.Join(BasePath, "settings", "pier", pier+".json")
		if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
			return err
		}
		tmpFile, err := os.CreateTemp(filepath.Dir(path), pier+".json.*")
		if err != nil {
			return fmt.Errorf("error creating temp file: %v", err)
		}
		// write and validate temp file before overwriting
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)
		encoder := json.NewEncoder(tmpFile)
		encoder.SetIndent("", "    ")
		if err := encoder.Encode(&config); err != nil {
			tmpFile.Close()
			return fmt.Errorf("error encoding config: %v", err)
		}
		if err := tmpFile.Close(); err != nil {
			return fmt.Errorf("error closing temp file: %v", err)
		}
		if fi, err := os.Stat(tmpPath); err != nil {
			return fmt.Errorf("error checking temp file: %v", err)
		} else if fi.Size() == 0 {
			return fmt.Errorf("refusing to persist empty configuration for pier %s", pier)
		}
		if err := os.Rename(tmpPath, path); err != nil {
			return fmt.Errorf("error moving temp file: %v", err)
		}
	}
	return nil
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
