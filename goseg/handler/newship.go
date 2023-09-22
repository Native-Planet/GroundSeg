package handler

import (
	"fmt"
	"goseg/click"
	"goseg/config"
	"goseg/defaults"
	"goseg/docker"
	"goseg/logger"
	"goseg/startram"
	"goseg/structs"
	"strings"
	"time"
)

func resetNewShip() error {
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: ""}
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "patp", Event: ""}
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "error", Event: ""}
	return nil
}

func createUrbitShip(patp string, shipPayload structs.WsNewShipPayload) {
	// transition: patp
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "patp", Event: patp}
	// transition: starting
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: "starting"}
	// create pier config
	err := createUrbitConfig(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
		errorCleanup(patp, errmsg)
		return
	}
	// update system.json
	err = appendSysConfigPier(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
		errorCleanup(patp, errmsg)
		return
	}
	// Prepare environment for pier
	logger.Logger.Info(fmt.Sprintf("Preparing environment for pier: %v", patp))
	// delete container if exists
	err = docker.DeleteContainer(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
	}
	// delete volume if exists
	err = docker.DeleteVolume(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
	}
	// create new docker volume
	err = docker.CreateVolume(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
		errorCleanup(patp, errmsg)
		return
	}
	// write key to volume
	key := shipPayload.Payload.Key
	err = docker.WriteFileToVolume(patp, patp+".key", key)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
		errorCleanup(patp, errmsg)
		return
	}
	// start container
	logger.Logger.Info(fmt.Sprintf("Creating Pier: %v", patp))
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: "creating"}
	info, err := docker.StartContainer(patp, "vere")
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
		errorCleanup(patp, errmsg)
		return
	}
	config.UpdateContainerState(patp, info)

	// debug, force error
	//errmsg := "Self induced error, for debugging purposes"
	//errorCleanup(patp, errmsg)
	//return

	// if startram is registered
	conf := config.Conf()
	if conf.WgRegistered {
		// Register Services
		go newShipRegisterService(patp)
	}
	// check for +code
	go waitForShipReady(shipPayload)
}

func waitForShipReady(shipPayload structs.WsNewShipPayload) {
	patp := shipPayload.Payload.Patp
	remote := shipPayload.Payload.Remote
	// transition: booting
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: "booting"}
	logger.Logger.Info(fmt.Sprintf("Booting ship: %v", patp))
	lusCodeTicker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-lusCodeTicker.C:
			code, err := click.GetLusCode(patp)
			if err != nil {
				continue
			}
			if len(code) == 27 {
				break
			}
		}
		conf := config.Conf()
		if conf.WgRegistered && conf.WgOn && remote {
			newShipToggleRemote(patp)
			shipConf := config.UrbitConf(patp)
			shipConf.Network = "wireguard"
			update := make(map[string]structs.UrbitDocker)
			update[patp] = shipConf
			if err := config.UpdateUrbitConfig(update); err != nil {
				errmsg := fmt.Sprintf("Failed to update urbit config for new ship: %v", err)
				errorCleanup(patp, errmsg)
				return
			}
			if err := docker.DeleteContainer(patp); err != nil {
				errmsg := fmt.Sprintf("Failed to delete local container for new ship: %v", err)
				logger.Logger.Error(errmsg)
			}
			docker.StartContainer("minio_"+patp, "minio")
			info, err := docker.StartContainer(patp, "vere")
			if err != nil {
				errmsg := fmt.Sprintf("%v", err)
				logger.Logger.Error(errmsg)
				errorCleanup(patp, errmsg)
				return
			}
			config.UpdateContainerState(patp, info)
		}
		docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: "completed"}
		return
	}
}

func newShipToggleRemote(patp string) {
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: "remote"}
	remoteTicker := time.NewTicker(1 * time.Second)
	// break if all subdomains with this patp has status of "ok"
	for {
		select {
		case <-remoteTicker.C:
			tramConf := config.StartramConfig
			for _, subd := range tramConf.Subdomains {
				if strings.Contains(subd.URL, patp) {
					if subd.Status != "ok" {
						continue
					}
				}
			}
			return
		}
	}
}

func createUrbitConfig(patp string) error {
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

func errorCleanup(patp string, errmsg string) {
	// send aborted transition
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: "aborted"}
	// send error transition
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "error", Event: fmt.Sprintf("%v", errmsg)}
	// notify that we are cleaning up
	logger.Logger.Info(fmt.Sprintf("New ship creation failed: %s: %s", patp, errmsg))
	logger.Logger.Info(fmt.Sprintf("Running cleanup routine"))
	// remove <patp>.json
	logger.Logger.Info(fmt.Sprintf("Removing Urbit Config: %s", patp))
	if err := config.RemoveUrbitConfig(patp); err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
	}
	// remove patp from system.json
	logger.Logger.Info(fmt.Sprintf("Removing pier entry from System Config: %v", patp))
	err := removeSysConfigPier(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
	}
	// remove docker volume
	err = docker.DeleteVolume(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
	}
}

// Remove all instances of patp from system config Piers
func removeSysConfigPier(patp string) error {
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

func appendSysConfigPier(patp string) error {
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

func newShipRegisterService(patp string) {
	if err := startram.RegisterNewShip(patp); err != nil {
		logger.Logger.Error(fmt.Sprintf("Unable to register StarTram service for %s: %v", patp, err))
	}
}
