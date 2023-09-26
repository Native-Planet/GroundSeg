package shipcreator

import (
	"fmt"
	"goseg/config"
	"goseg/defaults"
	"goseg/logger"
)

func CreateUrbitConfig(patp string) error {
	// get unused http and ames ports
	httpPort, amesPort := getOpenUrbitPorts()
	// get default urbit config
	conf := defaults.UrbitConfig
	// replace values
	conf.PierName = patp
	conf.HTTPPort = httpPort
	conf.AmesPort = amesPort
	// get urbit config map
	urbConf := config.UrbitConfAll()
	// add to map
	urbConf[patp] = conf
	// persist config
	err := config.UpdateUrbitConfig(urbConf)
	return err
}

func getOpenUrbitPorts() (int, int) {
	httpPort := 8080
	amesPort := 34343
	conf := config.Conf()
	piers := conf.Piers
	for _, pier := range piers {
		uConf := config.UrbitConf(pier)
		uHTTP := uConf.HTTPPort
		uAmes := uConf.AmesPort
		if uHTTP >= httpPort {
			httpPort = uHTTP
		}
		if uAmes >= amesPort {
			amesPort = uAmes
		}
	}
	httpPort = httpPort + 1
	amesPort = amesPort + 1
	logger.Logger.Info(fmt.Sprintf("Open Urbit Ports:  http: %v , ames: %v", httpPort, amesPort))
	return httpPort, amesPort
}

func AppendSysConfigPier(patp string) error {
	conf := config.Conf()
	piers := conf.Piers
	// Check if value already exists in slice
	exists := false
	for _, v := range piers {
		if v == patp {
			exists = true
			break
		}
	}
	// Append only if it doesn't exist yet
	if !exists {
		piers = append(piers, patp)
	}
	err := config.UpdateConf(map[string]interface{}{
		"piers": piers,
	})
	if err != nil {
		return err
	}
	return nil
}

// Remove all instances of patp from system config Piers
func RemoveSysConfigPier(patp string) error {
	conf := config.Conf()
	piers := conf.Piers
	var updated []string
	for _, memShip := range piers {
		if memShip != patp {
			updated = append(updated, memShip)
		}
	}
	err := config.UpdateConf(map[string]interface{}{
		"piers": updated,
	})
	return err
}
