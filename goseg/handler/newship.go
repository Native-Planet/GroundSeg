package handler

import (
	"fmt"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/shipcleanup"
	"groundseg/shipcreator"
	"groundseg/shipworkflow"
	"groundseg/startram"
	"groundseg/structs"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

func resetNewShip() error {
	docker.PublishNewShipTransition(structs.NewShipTransition{Type: "bootStage", Event: ""})
	docker.PublishNewShipTransition(structs.NewShipTransition{Type: "patp", Event: ""})
	docker.PublishNewShipTransition(structs.NewShipTransition{Type: "error", Event: ""})
	return nil
}

func createUrbitShip(patp string, shipPayload structs.WsNewShipPayload, customDrive string) {
	// transition: patp
	docker.PublishNewShipTransition(structs.NewShipTransition{Type: "patp", Event: patp})
	// transition: starting
	docker.PublishNewShipTransition(structs.NewShipTransition{Type: "bootStage", Event: "starting"})
	// create pier config
	err := shipcreator.CreateUrbitConfig(patp, customDrive)
	if err != nil {
		errmsg := fmt.Sprintf("failed to create urbit config: %v", err)
		zap.L().Error(errmsg)
		errorCleanup(patp, errmsg, customDrive)
		return
	}
	// update system.json
	err = shipcreator.AppendSysConfigPier(patp)
	if err != nil {
		errmsg := fmt.Sprintf("failed to add ship to system.json: %v", err)
		zap.L().Error(errmsg)
		errorCleanup(patp, errmsg, customDrive)
		return
	}
	// Prepare environment for pier
	zap.L().Info(fmt.Sprintf("Preparing environment for pier: %v", patp))
	// delete container if exists
	err = docker.DeleteContainer(patp)
	if err != nil {
		errmsg := fmt.Sprintf("delete container error: %v", err)
		zap.L().Error(errmsg)
	}
	// delete volume if exists
	err = docker.DeleteVolume(patp)
	if err != nil {
		errmsg := fmt.Sprintf("delete volume error: %v", err)
		zap.L().Error(errmsg)
	}
	// creating the volume for default settings
	if customDrive == "" {
		// create new docker volume
		err = docker.CreateVolume(patp)
		if err != nil {
			errmsg := fmt.Sprintf("create volume error: %v", err)
			zap.L().Error(errmsg)
			errorCleanup(patp, errmsg, customDrive)
			return
		}
		// write key to volume
		key := shipPayload.Payload.Key
		err = docker.WriteFileToVolume(patp, patp+".key", key)
		if err != nil {
			errmsg := fmt.Sprintf("write file to volume error: %v", err)
			zap.L().Error(errmsg)
			errorCleanup(patp, errmsg, customDrive)
			return
		}
	} else { // now this is for custom drive
		path := filepath.Join(customDrive, patp)
		filename := patp + ".key"
		key := shipPayload.Payload.Key
		// Create directory with all its parents
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			errmsg := fmt.Sprintf("write file to volume error: %v", err)
			zap.L().Error(errmsg)
			errorCleanup(patp, errmsg, customDrive)
			return
		}
		// Write content to the file
		filePath := path + "/" + filename
		if err := ioutil.WriteFile(filePath, []byte(key), 0644); err != nil {
			errmsg := fmt.Sprintf("Error writing to file: %v", err)
			zap.L().Error(errmsg)
			errorCleanup(patp, errmsg, customDrive)
			return
		}
	}
	// start container
	zap.L().Info(fmt.Sprintf("Creating Pier: %v", patp))
	docker.PublishNewShipTransition(structs.NewShipTransition{Type: "bootStage", Event: "creating"})
	info, err := docker.StartContainer(patp, "vere")
	if err != nil {
		errmsg := fmt.Sprintf("start container error: %v", err)
		zap.L().Error(errmsg)
		errorCleanup(patp, errmsg, customDrive)
		return
	}
	config.UpdateContainerState(patp, info)

	// debug, force error
	//errmsg := "Self induced error, for debugging purposes"
	//errorCleanup(patp, errmsg, customDrive)
	//return

	// if startram is registered
	conf := config.Conf()
	if conf.WgRegistered {
		// Register Services
		go newShipRegisterService(patp)
	}
	if conf.PenpaiAllow {
		if err := docker.StopContainerByName("llama"); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't stop Llama: %v", err))
		}
		_, err = docker.StartContainer("llama", "llama")
		if err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't restart Llama: %v", err))
		}
	}
	// check for +code
	go waitForShipReady(shipPayload, customDrive)
}

func waitForShipReady(shipPayload structs.WsNewShipPayload, customDrive string) {
	patp := shipPayload.Payload.Patp
	remote := shipPayload.Payload.Remote
	// transition: booting
	docker.PublishNewShipTransition(structs.NewShipTransition{Type: "bootStage", Event: "booting"})
	zap.L().Info(fmt.Sprintf("Booting ship: %v", patp))
	shipworkflow.WaitForBootCode(patp, 1*time.Second)
	conf := config.Conf()
	if conf.WgRegistered && conf.WgOn && remote {
		docker.PublishNewShipTransition(structs.NewShipTransition{Type: "bootStage", Event: "remote"})
		shipworkflow.WaitForRemoteReady(patp, 1*time.Second)
		if err := shipworkflow.SwitchShipToWireguard(patp, false); err != nil {
			errmsg := fmt.Sprintf("%v", err)
			zap.L().Error(errmsg)
			errorCleanup(patp, errmsg, customDrive)
			return
		}
	}
	startram.SyncRetrieve()
	docker.PublishNewShipTransition(structs.NewShipTransition{Type: "bootStage", Event: "completed"})
	// restart llama if it's enabled to reload avail ships
	if conf.PenpaiAllow {
		docker.StartContainer("llama-gpt-api", "llama-api")
	}
}

func errorCleanup(patp, errmsg, customDrive string) {
	// send aborted transition
	docker.PublishNewShipTransition(structs.NewShipTransition{Type: "bootStage", Event: "aborted"})
	// send error transition
	docker.PublishNewShipTransition(structs.NewShipTransition{Type: "error", Event: fmt.Sprintf("%v", errmsg)})
	// notify that we are cleaning up
	zap.L().Info(fmt.Sprintf("New ship creation failed: %s: %s", patp, errmsg))
	zap.L().Info(fmt.Sprintf("Running cleanup routine"))
	customPierPath := ""
	if customDrive != "" {
		customPierPath = filepath.Join(customDrive, patp)
	}
	if err := shipcleanup.RollbackProvisioning(patp, shipcleanup.RollbackOptions{
		CustomPierPath:       customPierPath,
		RemoveContainer:      true,
		RemoveContainerState: true,
	}); err != nil {
		zap.L().Error(fmt.Sprintf("New ship rollback encountered errors: %v", err))
	}
}

func newShipRegisterService(patp string) {
	if err := shipworkflow.RegisterShipServices(patp); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to register StarTram service for %s: %v", patp, err))
	}
}
