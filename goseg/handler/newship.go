package handler

import (
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/shipcreator"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/system"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
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

	// custom drive, leave empty if not on other drive
	var customDrive string
	sel := shipPayload.Payload.SelectedDrive //string
	// user wants to install it on custom drive
	if sel != "system-drive" {
		blockDevices, err := system.ListHardDisks()
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Failed to retrieve block devices: %v", err))
			return
		}
		// we're looking for the drive the user specified
		for _, dev := range blockDevices.BlockDevices {
			// we have the drive
			if dev.Name == sel {
				for _, m := range dev.Mountpoints {
					// check if mountpoint matches groundseg's expectations
					matched, err := regexp.MatchString(`^/groundseg-\d+$`, m)
					if err != nil {
						zap.L().Error(fmt.Sprintf("Regex error for mountpoint: %v", m))
						continue
					}
					// yes
					if matched {
						customDrive = m
						// breaks inner loop, we've got our directory
						break
					}
				}
				// breaks outer loop after we finish getting the info we need
				break
			}
		}
	}
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
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: "creating"}
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
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: "booting"}
	zap.L().Info(fmt.Sprintf("Booting ship: %v", patp))
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
				errorCleanup(patp, errmsg, customDrive)
				return
			}
			if err := docker.DeleteContainer(patp); err != nil {
				errmsg := fmt.Sprintf("Failed to delete local container for new ship: %v", err)
				zap.L().Error(errmsg)
			}
			docker.StartContainer("minio_"+patp, "minio")
			info, err := docker.StartContainer(patp, "vere")
			if err != nil {
				errmsg := fmt.Sprintf("%v", err)
				zap.L().Error(errmsg)
				errorCleanup(patp, errmsg, customDrive)
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

func errorCleanup(patp, errmsg, customDrive string) {
	// send aborted transition
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: "aborted"}
	// send error transition
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "error", Event: fmt.Sprintf("%v", errmsg)}
	// notify that we are cleaning up
	zap.L().Info(fmt.Sprintf("New ship creation failed: %s: %s", patp, errmsg))
	zap.L().Info(fmt.Sprintf("Running cleanup routine"))
	// remove <patp>.json
	zap.L().Info(fmt.Sprintf("Removing Urbit Config: %s", patp))
	if err := config.RemoveUrbitConfig(patp); err != nil {
		errmsg := fmt.Sprintf("%v", err)
		zap.L().Error(errmsg)
	}
	// remove patp from system.json
	zap.L().Info(fmt.Sprintf("Removing pier entry from System Config: %v", patp))
	err := shipcreator.RemoveSysConfigPier(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		zap.L().Error(errmsg)
	}
	// remove docker volume
	err = docker.DeleteVolume(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		zap.L().Error(errmsg)
	}
	if customDrive != "" {
		drivePath := filepath.Join(customDrive, patp)
		// Check if the directory exists
		if _, err := os.Stat(drivePath); !os.IsNotExist(err) {
			os.RemoveAll(drivePath)
		}
	}
}

func newShipRegisterService(patp string) {
	if err := startram.RegisterNewShip(patp); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to register StarTram service for %s: %v", patp, err))
	}
}
