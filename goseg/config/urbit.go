package config

// functions related to managing urbit config jsons & corresponding structs

import (
	"context"
	"encoding/json"
	"fmt"
	"groundseg/defaults"
	"groundseg/structs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

var (
	UrbitsConfig = make(map[string]structs.UrbitDocker)
	urbitMutex   sync.RWMutex
	AzimuthPoints map[string]*structs.Point
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
		errmsg := fmt.Sprintf("Unable to load %s config: %v", pier, err)
		return fmt.Errorf(errmsg)
		// todo: write a new conf
	}
	// Unmarshal JSON
	var targetStruct structs.UrbitDocker
	if err := json.Unmarshal(file, &targetStruct); err != nil {
		errmsg := fmt.Sprintf("Error decoding %s JSON: %v", pier, err)
		return fmt.Errorf(errmsg)
	}
	// set startram reminder
	if targetStruct.StartramReminder == nil {
		targetStruct.StartramReminder = defaults.UrbitConfig.StartramReminder
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
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "    ")
		if err := encoder.Encode(&config); err != nil {
			return err
		}
	}
	return nil
}

func getImageTagByContainerName(containerName string) (string, error) {
	ctx := context.Background()

	// Create a new Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	// Set up a filter to search for the container by name using a filter
	filterArgs := filters.NewArgs()
	filterArgs.Add("name", containerName)

	// List containers using the filter
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{Filters: filterArgs, All: true})
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
