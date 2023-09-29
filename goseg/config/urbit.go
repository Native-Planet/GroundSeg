package config

// functions related to managing urbit config jsons & corresponding structs

import (
	"encoding/json"
	"fmt"
	"goseg/structs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
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
