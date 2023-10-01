package handler

import (
	"encoding/json"
	"fmt"
	"goseg/broadcast"
	"goseg/config"
	"goseg/docker"
	"goseg/exporter"
	"goseg/logger"
	"goseg/startram"
	"goseg/structs"
	"strings"
	"time"
)

// we'll deal with breaking up this monstrosity
// when we become better humans

// handle urbit-type events
func UrbitHandler(msg []byte) error {
	logger.Logger.Info("Urbit")
	var urbitPayload structs.WsUrbitPayload
	err := json.Unmarshal(msg, &urbitPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal urbit payload: %v", err)
	}
	patp := urbitPayload.Payload.Patp
	shipConf := config.UrbitConf(patp)
	switch urbitPayload.Payload.Action {
	case "loom":
		shipConf.LoomSize = urbitPayload.Payload.Value
		update := make(map[string]structs.UrbitDocker)
		update[patp] = shipConf
		if err := config.UpdateUrbitConfig(update); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %v", err)
		}
		if err := docker.DeleteContainer(patp); err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to delete container: %v", err))
		}
		if shipConf.BootStatus == "boot" {
			if _, err := docker.StartContainer(patp, "vere"); err != nil {
				logger.Logger.Error(fmt.Sprintf("Couldn't start %v: %v", patp, err))
			}
		}
		return nil
	case "toggle-boot-status":
		if shipConf.BootStatus == "ignore" {
			statusMap, err := docker.GetShipStatus([]string{patp})
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("Failed to get ship status for %s", patp))
			}
			status, exists := statusMap[patp]
			if !exists {
				logger.Logger.Error(fmt.Sprintf("Running status for %s doesn't exist", patp))
			}
			if strings.Contains(status, "Up") {
				shipConf.BootStatus = "boot"
			} else {
				shipConf.BootStatus = "noboot"
			}
		} else {
			shipConf.BootStatus = "ignore"
		}
		update := make(map[string]structs.UrbitDocker)
		update[patp] = shipConf
		if err := config.UpdateUrbitConfig(update); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %v", err)
		}
		return nil
	case "toggle-network":
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "toggleNetwork", Event: "loading"}
		defer func() { docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "toggleNetwork", Event: ""} }()
		currentNetwork := shipConf.Network
		conf := config.Conf()
		logger.Logger.Warn(fmt.Sprintf("%v", currentNetwork))
		if currentNetwork == "wireguard" {
			shipConf.Network = "bridge"
			update := make(map[string]structs.UrbitDocker)
			update[patp] = shipConf
			if err := config.UpdateUrbitConfig(update); err != nil {
				return fmt.Errorf("Couldn't update urbit config: %v", err)
			}
			if err := docker.DeleteContainer(patp); err != nil {
				logger.Logger.Error(fmt.Sprintf("Failed to delete container: %v", err))
			}
		} else if currentNetwork != "wireguard" && conf.WgRegistered == true {
			shipConf.Network = "wireguard"
			update := make(map[string]structs.UrbitDocker)
			update[patp] = shipConf
			if err := config.UpdateUrbitConfig(update); err != nil {
				return fmt.Errorf("Couldn't update urbit config: %v", err)
			}
			if err := docker.DeleteContainer(patp); err != nil {
				logger.Logger.Error(fmt.Sprintf("Failed to delete container: %v", err))
			}
		} else {
			return fmt.Errorf("No remote registration")
		}
		if shipConf.BootStatus == "boot" {
			if _, err := docker.StartContainer(patp, "vere"); err != nil {
				logger.Logger.Error(fmt.Sprintf("Couldn't start %v: %v", patp, err))
			}
		}
		return nil
	case "toggle-devmode":
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "toggleDevMode", Event: "loading"}
		defer func() { docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "toggleDevMode", Event: ""} }()
		if shipConf.DevMode == true {
			shipConf.DevMode = false
		} else {
			shipConf.DevMode = true
		}
		update := make(map[string]structs.UrbitDocker)
		update[patp] = shipConf
		if err := config.UpdateUrbitConfig(update); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %v", err)
		}
		if err := docker.DeleteContainer(patp); err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to delete container: %v", err))
		}
		_, err := docker.StartContainer(patp, "vere")
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("%v", err))
		}
		return nil
	case "toggle-power":
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "togglePower", Event: "loading"}
		defer func() {
			docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "togglePower", Event: ""}
		}()
		update := make(map[string]structs.UrbitDocker)
		if shipConf.BootStatus == "noboot" {
			shipConf.BootStatus = "boot"
			update[patp] = shipConf
			if err := config.UpdateUrbitConfig(update); err != nil {
				return fmt.Errorf("Couldn't update urbit config: %v", err)
			}
			_, err := docker.StartContainer(patp, "vere")
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("%v", err))
			}
		} else if shipConf.BootStatus == "boot" {
			shipConf.BootStatus = "noboot"
			update[patp] = shipConf
			if err := config.UpdateUrbitConfig(update); err != nil {
				return fmt.Errorf("Couldn't update urbit config: %v", err)
			}
			err := docker.StopContainerByName(patp)
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("%v", err))
			}
		}
		return nil
	case "export-bucket":
		containerName := fmt.Sprintf("minio_%s", patp)
		// whitelist the patp token pair
		if err := exporter.WhitelistContainer(containerName, urbitPayload.Token); err != nil {
			return err
		}
		// transition: ready
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "exportBucket", Event: "ready"}
		return nil
	case "export-ship":
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "exportShip", Event: "stopping"}
		update := make(map[string]structs.UrbitDocker)
		shipConf.BootStatus = "noboot"
		update[patp] = shipConf
		if err := config.UpdateUrbitConfig(update); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %v", err)
		}
		// stop container
		if err := docker.StopContainerByName(patp); err != nil {
			return err
		}
		// whitelist the patp token pair
		if err := exporter.WhitelistContainer(patp, urbitPayload.Token); err != nil {
			return err
		}
		// transition: ready
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "exportShip", Event: "ready"}
		return nil
	case "delete-ship":
		conf := config.Conf()
		// update DesiredStatus to 'stopped'
		contConf := config.GetContainerState()
		patpConf := contConf[patp]
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "deleteShip", Event: "stopping"}
		patpConf.DesiredStatus = "stopped"
		contConf[patp] = patpConf
		config.UpdateContainerState(patp, patpConf)
		if err := docker.StopContainerByName(patp); err != nil {
			return fmt.Errorf(fmt.Sprintf("Couldn't stop docker container for %v: %v", patp, err))
		}
		if err := docker.DeleteContainer(patp); err != nil {
			return fmt.Errorf(fmt.Sprintf("Couldn't delete docker container for %v: %v", patp, err))
		}
		if conf.WgRegistered {
			docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "deleteShip", Event: "removing-services"}
			if err := startram.SvcDelete(patp, "urbit"); err != nil {
				logger.Logger.Error(fmt.Sprintf("Couldn't remove urbit anchor for %v: %v", patp, err))
			}
			if err := startram.SvcDelete("s3."+patp, "s3"); err != nil {
				logger.Logger.Error(fmt.Sprintf("Couldn't remove s3 anchor for %v: %v", patp, err))
			}
		}
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "deleteShip", Event: "deleting"}
		if err := config.RemoveUrbitConfig(patp); err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't remove config for %v: %v", patp, err))
		}
		conf = config.Conf()
		piers := cutSlice(conf.Piers, patp)
		if err = config.UpdateConf(map[string]interface{}{
			"piers": piers,
		}); err != nil {
			logger.Logger.Error(fmt.Sprintf("Error updating config: %v", err))
		}
		if err := docker.DeleteVolume(patp); err != nil {
			return fmt.Errorf(fmt.Sprintf("Couldn't remove docker volume for %v: %v", patp, err))
		}
		config.DeleteContainerState(patp)
		logger.Logger.Info(fmt.Sprintf("%v container deleted", patp))
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "deleteShip", Event: "success"}
		time.Sleep(3 * time.Second)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "deleteShip", Event: "done"}
		time.Sleep(1 * time.Second)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "deleteShip", Event: ""}
		// remove from broadcast
		if err := broadcast.ReloadUrbits(); err != nil {
			logger.Logger.Error(fmt.Sprintf("Error updating broadcast: %v", err))
		}
		return nil
	default:
		return fmt.Errorf("Unrecognized urbit action: %v", urbitPayload.Payload.Action)
	}
}

// remove a string from a slice of strings
func cutSlice(slice []string, s string) []string {
	index := -1
	for i, v := range slice {
		if v == s {
			index = i
			break
		}
	}
	if index == -1 {
		return slice
	}
	return append(slice[:index], slice[index+1:]...)
}
