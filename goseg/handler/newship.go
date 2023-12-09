package handler

import (
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/logger"
	"groundseg/shipcreator"
	"groundseg/startram"
	"groundseg/structs"
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
	err := shipcreator.CreateUrbitConfig(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
		errorCleanup(patp, errmsg)
		return
	}
	// update system.json
	err = shipcreator.AppendSysConfigPier(patp)
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
	if conf.PenpaiAllow {
		if err := docker.StopContainerByName("llama"); err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't stop Llama: %v", err))
		}
		_, err = docker.StartContainer("llama", "llama")
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't restart Llama: %v", err))
		}
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
			} else {
				continue
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
		startram.Retrieve()
		docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: "completed"}
		// restart llama if it's enabled to reload avail ships
		if conf.PenpaiAllow {
			docker.StartContainer("llama-gpt-api", "llama-api")
		}
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
	err := shipcreator.RemoveSysConfigPier(patp)
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

func newShipRegisterService(patp string) {
	if err := startram.RegisterNewShip(patp); err != nil {
		logger.Logger.Error(fmt.Sprintf("Unable to register StarTram service for %s: %v", patp, err))
	}
}
