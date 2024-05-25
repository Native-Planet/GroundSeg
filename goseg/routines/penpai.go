package routines

import (
	"groundseg/config"
	"groundseg/logger"
	"path/filepath"
	"time"
)

func TrackPenpaiModels() {
	for {
		conf := config.Conf()
		if conf.PenpaiAllow {
			dir := filepath.Join(conf.DockerData, "volumes", "llama-gpt-api", "_data")
			for _, m := range conf.PenpaiModels {
				if m.Exists {
					logger.Logger.Warn("todo: check penpai hash")
					_ = dir
					// find file in dir, if doesn't exist, download again
					// check hash
					// if incorrect hash, wipe and download again
				}
			}
		}
		/*
			if err := config.UpdateConf(map[string]interface{}{
				"diskWarning": conf.DiskWarning,
			}); err != nil {
				return fmt.Errorf("Couldn't set disk warning in config 80%:%v 90%:%v 95%:%v: %v", err, eighty, ninety, ninetyFive)
			}
		*/

		//time.Sleep(5 * time.Minute)
		time.Sleep(15 * time.Second)
	}
}
