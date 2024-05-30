package routines

import (
	"fmt"
	"groundseg/config"
	"groundseg/logger"
	"groundseg/penpai"
	"os"
	"path/filepath"
	"time"
)

func TrackPenpaiModels() {
	for {
		conf := config.Conf()
		if conf.PenpaiAllow {
			dir := filepath.Join(conf.DockerData, "volumes", "llama-gpt-api", "_data")
			for _, m := range conf.PenpaiModels {
				name, err := penpai.ExtractLlamafileName(m.ModelUrl)
				if err != nil {
					logger.Logger.Error(fmt.Sprintf("Penpai routine failed to retrieve file name: %v", err))
					continue
				}
				f := filepath.Join(dir, name)
				if _, err := os.Stat(f); err != nil {
					if !os.IsNotExist(err) {
						logger.Logger.Error(fmt.Sprintf("Penpai routine failed to check llamafile existence: %v", err))
					}
					continue
				}
				// check hash
				hash, err := penpai.GetSHA256(f)
				if err != nil {
					logger.Logger.Error(fmt.Sprintf("Penpai routine failed to check llamafile hash: %v", err))
					continue
				}
				if _, exists := penpai.Locked[m.ModelHash]; exists {
					continue
				}
				if hash != m.ModelHash {
					logger.Logger.Error(fmt.Sprintf("Penpai routine found invalid hash: %v", m.ModelName))
					continue
				}
				var val *int
				for i, model := range conf.PenpaiModels {
					if model.ModelHash == hash {
						val = &i
						break
					}
				}

				if val == nil {
					logger.Logger.Error(fmt.Sprintf("Penpai routine invalid index: %v", m.ModelName))
					continue
				}

				// update config
				if err := penpai.SetExistence(m, true, *val); err != nil {
					logger.Logger.Error(fmt.Sprintf("Penpai routine failed to update config: %v", err))
					continue
				}
			}
		}

		//time.Sleep(5 * time.Minute)
		time.Sleep(30 * time.Second)
	}
}
