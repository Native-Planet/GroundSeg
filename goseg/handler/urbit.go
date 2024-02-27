package handler

import (
	"encoding/json"
	"fmt"
	"groundseg/broadcast"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/exporter"
	"groundseg/logger"
	"groundseg/startram"
	"groundseg/structs"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

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
	// show custom domain for urbit ship in UI
	case "toggle-alias":
		return toggleAlias(patp, shipConf)
		// exports
	case "export-bucket":
		return exportBucket(patp, urbitPayload, shipConf)
	case "export-ship":
		return exportShip(patp, urbitPayload, shipConf)
	// set custom domains
	case "set-urbit-domain":
		return setUrbitDomain(patp, urbitPayload, shipConf)
	case "set-minio-domain":
		return setMinIODomain(patp, urbitPayload, shipConf)
		// set whether or not ship wants startram reminders
	case "startram-reminder":
		return startramReminder(patp, urbitPayload.Payload.Remind, shipConf)
		// urbit desks
	case "install-penpai-companion":
		return installPenpaiCompanion(patp, shipConf)
	case "uninstall-penpai-companion":
		return uninstallPenpaiCompanion(patp, shipConf)
		/*
			case "install-gallseg":
				return installGallseg(patp, shipConf)
			case "uninstall-gallseg":
				return uninstallGallseg(patp, shipConf)
		*/
		// ship operations
	case "chop":
		return ChopPier(patp, shipConf)
	case "roll-chop":
		return rollChopPier(patp, shipConf)
	case "pack":
		return packPier(patp, shipConf)
	case "pack-meld":
		return packMeldPier(patp, shipConf)
	case "rebuild-container":
		return rebuildContainer(patp, shipConf)
	case "toggle-minio-link":
		return toggleMinIOLink(patp, shipConf)
		// ship configuration
	case "pause-pack-schedule":
		return pausePackSchedule(patp, urbitPayload, shipConf)
	case "schedule-pack":
		return schedulePack(patp, urbitPayload, shipConf)
	case "loom":
		return handleLoom(patp, urbitPayload, shipConf)
	case "toggle-boot-status":
		return toggleBootStatus(patp, shipConf)
	case "toggle-network":
		return toggleNetwork(patp, shipConf)
	case "toggle-devmode":
		return toggleDevMode(patp, shipConf)
	case "toggle-power":
		return togglePower(patp, shipConf)
	case "delete-ship":
		return deleteShip(patp, shipConf)
	case "toggle-chop-on-vere-update":
		return toggleChopOnVereUpdate(patp, shipConf)
	case "new-max-pier-size":
		return setNewMaxPierSize(patp, urbitPayload, shipConf)
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

// AreSubdomainsAliases checks if two subdomains are aliases of each other.
func AreSubdomainsAliases(domain1, domain2 string) (bool, error) {
	// Lookup CNAME for the first domain
	cname1, err := net.LookupCNAME(domain1)
	if err != nil {
		return false, err
	}

	// Lookup CNAME for the second domain
	cname2, err := net.LookupCNAME(domain2)
	if err != nil {
		return false, err
	}

	// Compare CNAMEs
	return cname1 == cname2, nil
}

func waitComplete(patp string) {
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			statuses, err := docker.GetShipStatus([]string{patp})
			if err != nil {
				continue
			}
			status, exists := statuses[patp]
			if !exists {
				continue
			}
			if strings.Contains(status, "Up") {
				logger.Logger.Debug(fmt.Sprintf("%s continue waiting...", patp))
				continue
			}
			logger.Logger.Debug(fmt.Sprintf("%s finished", patp))
			return
		}
	}
}

func urbitCleanDelete(patp string) error {
	getShipRunningStatus := func(patp string) (string, error) {
		statuses, err := docker.GetShipStatus([]string{patp})
		if err != nil {
			return "", fmt.Errorf("Failed to get statuses for %s: %v", patp, err)
		}
		status, exists := statuses[patp]
		if !exists {
			return "", fmt.Errorf("%s status doesn't exist", patp)
		}
		return status, nil
	}
	status, err := getShipRunningStatus(patp)
	if err == nil {
		isRunning := strings.Contains(status, "Up")
		if isRunning {
			if err := click.BarExit(patp); err != nil {
				logger.Logger.Error(fmt.Sprintf("Failed to stop %s with |exit: %v", patp, err))
			}
		}
		for {
			status, err := getShipRunningStatus(patp)
			if err != nil {
				break
			}
			logger.Logger.Warn(fmt.Sprintf("%s", status))
			if !strings.Contains(status, "Up") {
				break
			}
			time.Sleep(1 * time.Second)
		}
	}
	if err := docker.DeleteContainer(patp); err != nil {
		return fmt.Errorf("Failed to delete container %s", patp)
	}
	return nil
}

func installPenpaiCompanion(patp string, shipConf structs.UrbitDocker) error {
	// run after complete
	defer func(patp string) {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "penpaiCompanion", Event: ""}
	}(patp)

	// initial transition
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "penpaiCompanion", Event: "loading"}

	// error handling
	handleError := func(patp, errMsg string, err error) error {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "penpaiCompanion", Event: "error"}
		time.Sleep(3 * time.Second)
		return fmt.Errorf("%s: %s: %v", patp, errMsg, err)
	}

	// if not-found, |install, if suspended, |revive
	status, err := click.GetDesk(patp, "penpai", true)
	if err != nil {
		return handleError(patp, "Handler failed to get penpai desk info", err)
	}
	if status == "not-found" {
		err := click.InstallDesk(patp, "~nattyv", "penpai")
		if err != nil {
			return handleError(patp, "Handler failed to get install penpai desk", err)
		}
	} else if status == "suspended" {
		err := click.ReviveDesk(patp, "penpai")
		if err != nil {
			return handleError(patp, "Handler failed to revive penpai desk", err)
		}
	}
	// wait for complete
	for {
		time.Sleep(5 * time.Second)
		status, err := click.GetDesk(patp, "penpai", true)
		if err != nil {
			return handleError(patp, "Handler failed to get penpai desk info after installation succeeded", err)
		}
		if status == "running" {
			docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "penpaiCompanion", Event: "success"}
			time.Sleep(3 * time.Second)
			break
		}
	}
	return nil
}

func uninstallPenpaiCompanion(patp string, shipConf structs.UrbitDocker) error {
	// run after complete
	defer func(patp string) {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "penpaiCompanion", Event: ""}
	}(patp)

	// initial transition
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "penpaiCompanion", Event: "loading"}

	// error handling
	handleError := func(patp, errMsg string, err error) error {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "penpaiCompanion", Event: "error"}
		time.Sleep(3 * time.Second)
		return fmt.Errorf("%s: %s: %v", patp, errMsg, err)
	}

	// uninstall
	err := click.UninstallDesk(patp, "penpai")
	if err != nil {
		return handleError(patp, "Handler failed to install uninstall the penpai desk", err)
	}
	for {
		time.Sleep(5 * time.Second)
		status, err := click.GetDesk(patp, "penpai", true)
		if err != nil {
			return handleError(patp, "Handler failed to get penpai desk info after uninstallation succeeded", err)
		}
		if status != "running" {
			docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "penpaiCompanion", Event: "success"}
			time.Sleep(3 * time.Second)
			break
		}
	}
	return nil
}

func installGallseg(patp string, shipConf structs.UrbitDocker) error {
	// run after complete
	defer func(patp string) {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "gallseg", Event: ""}
	}(patp)

	// initial transition
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "gallseg", Event: "loading"}

	// error handling
	handleError := func(patp, errMsg string, err error) error {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "gallseg", Event: "error"}
		time.Sleep(3 * time.Second)
		return fmt.Errorf("%s: %s: %v", patp, errMsg, err)
	}

	// if not-found, |install, if suspended, |revive
	status, err := click.GetDesk(patp, "groundseg", true)
	if err != nil {
		return handleError(patp, "Handler failed to get groundseg desk info", err)
	}
	if status == "not-found" {
		err := click.InstallDesk(patp, "~tadwer-pilbud-nallux-dozryl", "groundseg")
		if err != nil {
			return handleError(patp, "Handler failed to get install groundseg desk", err)
		}
	} else if status == "suspended" {
		err := click.ReviveDesk(patp, "groundseg")
		if err != nil {
			return handleError(patp, "Handler failed to revive groundseg desk", err)
		}
	}
	// wait for complete
	for {
		time.Sleep(5 * time.Second)
		status, err := click.GetDesk(patp, "groundseg", true)
		if err != nil {
			return handleError(patp, "Handler failed to get groundseg desk info after installation succeeded", err)
		}
		if status == "running" {
			docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "gallseg", Event: "success"}
			time.Sleep(3 * time.Second)
			break
		}
	}
	return nil
}

func uninstallGallseg(patp string, shipConf structs.UrbitDocker) error {
	// run after complete
	defer func(patp string) {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "gallseg", Event: ""}
	}(patp)

	// initial transition
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "gallseg", Event: "loading"}

	// error handling
	handleError := func(patp, errMsg string, err error) error {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "gallseg", Event: "error"}
		time.Sleep(3 * time.Second)
		return fmt.Errorf("%s: %s: %v", patp, errMsg, err)
	}

	// uninstall
	err := click.UninstallDesk(patp, "groundseg")
	if err != nil {
		return handleError(patp, "Handler failed to install uninstall the groundseg desk", err)
	}
	for {
		time.Sleep(5 * time.Second)
		status, err := click.GetDesk(patp, "groundseg", true)
		if err != nil {
			return handleError(patp, "Handler failed to get groundseg desk info after uninstallation succeeded", err)
		}
		if status != "running" {
			docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "gallseg", Event: "success"}
			time.Sleep(3 * time.Second)
			break
		}
	}
	return nil
}

func startramReminder(patp string, remind bool, shipConf structs.UrbitDocker) error {
	update := make(map[string]structs.UrbitDocker)
	shipConf.StartramReminder = remind
	update[patp] = shipConf
	if err := config.UpdateUrbitConfig(update); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %v", err)
	}
	return nil
}

func packPier(patp string, shipConf structs.UrbitDocker) error {
	// error handling
	packError := func(err error) error {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "pack", Event: "error"}
		return err
	}
	// clear transition after end
	defer func() {
		time.Sleep(3 * time.Second)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "pack", Event: ""}
	}()
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "pack", Event: "packing"}
	statuses, err := docker.GetShipStatus([]string{patp})
	if err != nil {
		return packError(fmt.Errorf("Failed to get ship status for %p: %v", patp, err))
	}
	status, exists := statuses[patp]
	if !exists {
		return packError(fmt.Errorf("Failed to get ship status for %p: status doesn't exist!", patp))
	}
	// running
	if strings.Contains(status, "Up") {
		// send |pack
		if err := click.SendPack(patp); err != nil {
			return packError(fmt.Errorf("Failed to |pack to %s: %v", patp, err))
		}
		// not running
	} else {
		// switch boot status to pack
		shipConf.BootStatus = "pack"
		update := make(map[string]structs.UrbitDocker)
		update[patp] = shipConf
		err := config.UpdateUrbitConfig(update)
		if err != nil {
			return packError(fmt.Errorf("Failed to update %s urbit config to pack: %v", patp, err))
		}
		_, err = docker.StartContainer(patp, "vere")
		if err != nil {
			return packError(fmt.Errorf("Failed to urth pack %s: %v", patp, err))
		}
	}
	// set last meld
	now := time.Now().Unix()
	shipConf.MeldLast = strconv.FormatInt(now, 10)
	update := make(map[string]structs.UrbitDocker)
	update[patp] = shipConf
	err = config.UpdateUrbitConfig(update)
	if err != nil {
		return packError(fmt.Errorf("Failed to update %s urbit config with last meld time: %v", patp, err))
	}
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "pack", Event: "success"}
	return nil
}

func packMeldPier(patp string, shipConf structs.UrbitDocker) error {
	packMeldError := func(err error) error {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "packMeld", Event: "error"}
		return err
	}
	// clear transition after end
	defer func() {
		time.Sleep(3 * time.Second)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "packMeld", Event: ""}
	}()
	statuses, err := docker.GetShipStatus([]string{patp})
	if err != nil {
		return packMeldError(fmt.Errorf("Failed to get ship status for %p: %v", patp, err))
	}
	status, exists := statuses[patp]
	if !exists {
		return packMeldError(fmt.Errorf("Failed to get ship status for %p: status doesn't exist!", patp))
	}
	isRunning := strings.Contains(status, "Up")
	if isRunning {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "packMeld", Event: "stopping"}
		if err := click.BarExit(patp); err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to stop ship with |exit for pack & meld %s: %v", patp, err))
			if err = docker.StopContainerByName(patp); err != nil {
				logger.Logger.Error(fmt.Sprintf("Failed to stop ship for pack & meld %s: %v", patp, err))
			}
		}
	}
	// stop ship
	// start ship as pack
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "packMeld", Event: "packing"}
	logger.Logger.Info(fmt.Sprintf("Attempting to urth pack %s", patp))
	shipConf.BootStatus = "pack"
	update := make(map[string]structs.UrbitDocker)
	update[patp] = shipConf
	err = config.UpdateUrbitConfig(update)
	if err != nil {
		return packMeldError(fmt.Errorf("Failed to update %s urbit config to pack: %v", patp, err))
	}
	_, err = docker.StartContainer(patp, "vere")
	if err != nil {
		return packMeldError(fmt.Errorf("Failed to urth pack %s: %v", patp, err))
	}

	logger.Logger.Info(fmt.Sprintf("Waiting for urth pack to complete for %s", patp))
	waitComplete(patp)

	// start ship as meld
	logger.Logger.Info(fmt.Sprintf("Attempting to urth meld %s", patp))
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "packMeld", Event: "melding"}
	shipConf.BootStatus = "meld"
	update = make(map[string]structs.UrbitDocker)
	update[patp] = shipConf
	err = config.UpdateUrbitConfig(update)
	if err != nil {
		return packMeldError(fmt.Errorf("Failed to update %s urbit config to meld: %v", patp, err))
	}
	_, err = docker.StartContainer(patp, "vere")
	if err != nil {
		return packMeldError(fmt.Errorf("Failed to urth meld %s: %v", patp, err))
	}

	logger.Logger.Info(fmt.Sprintf("Waiting for urth meld to complete for %s", patp))
	waitComplete(patp)

	// start ship if "boot"
	if isRunning {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "packMeld", Event: "starting"}
		shipConf.BootStatus = "boot"
		update := make(map[string]structs.UrbitDocker)
		update[patp] = shipConf
		err := config.UpdateUrbitConfig(update)
		if err != nil {
			return packMeldError(fmt.Errorf("Failed to update %s urbit config to meld: %v", patp, err))
		}
		_, err = docker.StartContainer(patp, "vere")
		if err != nil {
			return packMeldError(fmt.Errorf("Failed to urth meld %s: %v", patp, err))
		}
	}
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "packMeld", Event: "success"}
	return nil
}

func toggleAlias(patp string, shipConf structs.UrbitDocker) error {
	if shipConf.ShowUrbitWeb == "custom" {
		shipConf.ShowUrbitWeb = "default"
	} else {
		shipConf.ShowUrbitWeb = "custom"
	}
	update := make(map[string]structs.UrbitDocker)
	update[patp] = shipConf
	if err := config.UpdateUrbitConfig(update); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %v", err)
	}
	return nil
}

func setUrbitDomain(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	defer func() {
		time.Sleep(1 * time.Second)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "urbitDomain", Event: ""}
	}()
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "urbitDomain", Event: "loading"}
	// check if new domain is valid
	alias := urbitPayload.Payload.Domain
	oldDomain := shipConf.WgURL
	areAliases, err := AreSubdomainsAliases(alias, oldDomain)
	if err != nil {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "urbitDomain", Event: "error"}
		return fmt.Errorf("Failed to check Urbit domain alias for %s: %v", patp, err)
	}
	if !areAliases {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "urbitDomain", Event: "error"}
		return fmt.Errorf("Invalid Urbit domain alias for %s", patp)
	}
	// Creae Alias
	if err := startram.AliasCreate(patp, alias); err != nil {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "urbitDomain", Event: "error"}
		return err
	}
	shipConf.CustomUrbitWeb = alias
	shipConf.ShowUrbitWeb = "custom" // or "default"
	update := make(map[string]structs.UrbitDocker)
	update[patp] = shipConf
	if err := config.UpdateUrbitConfig(update); err != nil {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "urbitDomain", Event: "error"}
		return fmt.Errorf("Couldn't update urbit config: %v", err)
	}
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "urbitDomain", Event: "success"}
	time.Sleep(3 * time.Second)
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "urbitDomain", Event: "done"}
	return nil
}

func setMinIODomain(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	defer func() {
		time.Sleep(1 * time.Second)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "minioDomain", Event: ""}
	}()
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "minioDomain", Event: "loading"}
	// check if new domain is valid
	alias := urbitPayload.Payload.Domain
	oldDomain := fmt.Sprintf("s3.%s", shipConf.WgURL)
	areAliases, err := AreSubdomainsAliases(alias, oldDomain)
	if err != nil {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "minioDomain", Event: "error"}
		return fmt.Errorf("Failed to check MinIO domain alias for %s: %v", patp, err)
	}
	if !areAliases {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "minioDomain", Event: "error"}
		return fmt.Errorf("Invalid MinIO domain alias for %s", patp)
	}
	// Creae Alias
	if err := startram.AliasCreate(fmt.Sprintf("s3.%s", patp), alias); err != nil {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "minioDomain", Event: "error"}
		return err
	}
	shipConf.CustomS3Web = alias
	update := make(map[string]structs.UrbitDocker)
	update[patp] = shipConf
	if err := config.UpdateUrbitConfig(update); err != nil {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "minioDomain", Event: "error"}
		return fmt.Errorf("Couldn't update urbit config: %v", err)
	}
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "minioDomain", Event: "success"}
	time.Sleep(3 * time.Second)
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "minioDomain", Event: "done"}
	return nil
}

func ChopPier(patp string, shipConf structs.UrbitDocker) error {
	logger.Logger.Info(fmt.Sprintf("Chop called for %s", patp))
	// error handling
	chopError := func(err error) error {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "chop", Event: "error"}
		return err
	}
	// clear transition after end
	defer func() {
		time.Sleep(3 * time.Second)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "chop", Event: ""}
		logger.Logger.Info(fmt.Sprintf("Chop for %s, ran defer", patp))
	}()
	statuses, err := docker.GetShipStatus([]string{patp})
	if err != nil {
		return chopError(fmt.Errorf("Failed to get ship status for %p: %v", patp, err))
	}
	status, exists := statuses[patp]
	if !exists {
		return chopError(fmt.Errorf("Failed to get ship status for %p: status doesn't exist!", patp))
	}
	isRunning := strings.Contains(status, "Up")
	// stop ship
	if isRunning {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "chop", Event: "stopping"}
		if err := click.BarExit(patp); err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to stop ship with |exit for chop %s: %v", patp, err))
			if err = docker.StopContainerByName(patp); err != nil {
				return fmt.Errorf("Failed to stop ship for chop %s: %v", patp, err)
			}
		}
	}
	// start ship as chop
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "chop", Event: "chopping"}
	logger.Logger.Info(fmt.Sprintf("Attempting to chop %s", patp))
	shipConf.BootStatus = "chop"
	update := make(map[string]structs.UrbitDocker)
	update[patp] = shipConf
	err = config.UpdateUrbitConfig(update)
	if err != nil {
		return chopError(fmt.Errorf("Failed to update %s urbit config to chop: %v", patp, err))
	}
	_, err = docker.StartContainer(patp, "vere")
	if err != nil {
		return chopError(fmt.Errorf("Failed to chop %s: %v", patp, err))
	}

	logger.Logger.Info(fmt.Sprintf("Waiting for chop to complete for %s", patp))
	waitComplete(patp)

	// start ship if "boot"
	if isRunning {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "chop", Event: "starting"}
		shipConf.BootStatus = "boot"
		update := make(map[string]structs.UrbitDocker)
		update[patp] = shipConf
		err := config.UpdateUrbitConfig(update)
		if err != nil {
			return chopError(fmt.Errorf("Failed to update %s urbit config to chop: %v", patp, err))
		}
		_, err = docker.StartContainer(patp, "vere")
		if err != nil {
			return chopError(fmt.Errorf("Failed to chop %s: %v", patp, err))
		}
	}
	docker.ForceUpdateContainerStats(patp)
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "chop", Event: "success"}
	return nil
}

func toggleChopOnVereUpdate(patp string, shipConf structs.UrbitDocker) error {
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "chopOnUpgrade", Event: "loading"}
	defer func() {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "chopOnUpgrade", Event: ""}
	}()
	update := make(map[string]structs.UrbitDocker)
	if shipConf.ChopOnUpgrade == false {
		shipConf.ChopOnUpgrade = true
	} else {
		shipConf.ChopOnUpgrade = false
	}
	update[patp] = shipConf
	if err := config.UpdateUrbitConfig(update); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %v", err)
	}
	return nil
}

func deleteShip(patp string, shipConf structs.UrbitDocker) error {
	conf := config.Conf()
	// update DesiredStatus to 'stopped'
	contConf := config.GetContainerState()
	patpConf := contConf[patp]
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "deleteShip", Event: "stopping"}
	patpConf.DesiredStatus = "stopped"
	contConf[patp] = patpConf
	config.UpdateContainerState(patp, patpConf)
	if err := click.BarExit(patp); err != nil {
		logger.Logger.Error(fmt.Sprintf("%v", err))
		if err := docker.StopContainerByName(patp); err != nil {
			return fmt.Errorf(fmt.Sprintf("Couldn't stop docker container for %v: %v", patp, err))
		}
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
		if err := docker.DeleteContainer("minio_" + patp); err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't delete minio docker container for %v: %v", patp, err))
		}
	}
	// get custom directory info before deleting config
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "deleteShip", Event: "deleting"}
	if err := config.RemoveUrbitConfig(patp); err != nil {
		logger.Logger.Error(fmt.Sprintf("Couldn't remove config for %v: %v", patp, err))
	}
	conf = config.Conf()
	piers := cutSlice(conf.Piers, patp)
	if err := config.UpdateConf(map[string]interface{}{
		"piers": piers,
	}); err != nil {
		logger.Logger.Error(fmt.Sprintf("Error updating config: %v", err))
	}
	customLoc, ok := shipConf.CustomPierLocation.(string) // Type assertion to string
	if ok {
		if _, err := os.Stat(customLoc); !os.IsNotExist(err) {
			if err := os.RemoveAll(customLoc); err != nil {
				return fmt.Errorf("couldn't remove directory at %s: %v", customLoc, err)
			}
		}
	} else {
		if err := docker.DeleteVolume(patp); err != nil {
			return fmt.Errorf(fmt.Sprintf("Couldn't remove docker volume for %v: %v", patp, err))
		}
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
}

func exportShip(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "exportShip", Event: "stopping"}
	update := make(map[string]structs.UrbitDocker)
	shipConf.BootStatus = "noboot"
	update[patp] = shipConf
	if err := config.UpdateUrbitConfig(update); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %v", err)
	}
	// stop container
	if err := click.BarExit(patp); err != nil {
		logger.Logger.Error(fmt.Sprintf("%v", err))
		if err := docker.StopContainerByName(patp); err != nil {
			return err
		}
	}
	// whitelist the patp token pair
	if err := exporter.WhitelistContainer(patp, urbitPayload.Token); err != nil {
		return err
	}
	// transition: ready
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "exportShip", Event: "ready"}
	return nil
}

func exportBucket(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	containerName := fmt.Sprintf("minio_%s", patp)
	// whitelist the patp token pair
	if err := exporter.WhitelistContainer(containerName, urbitPayload.Token); err != nil {
		return err
	}
	// transition: ready
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "exportBucket", Event: "ready"}
	return nil
}

func togglePower(patp string, shipConf structs.UrbitDocker) error {
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
		err := click.BarExit(patp)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("%v", err))
			if err := docker.StopContainerByName(patp); err != nil {
				logger.Logger.Error(fmt.Sprintf("%v", err))
			}
		}
	}
	return nil
}

func toggleDevMode(patp string, shipConf structs.UrbitDocker) error {
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
	if err := urbitCleanDelete(patp); err != nil {
		logger.Logger.Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
	}
	_, err := docker.StartContainer(patp, "vere")
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("%v", err))
	}
	return nil
}

func rebuildContainer(patp string, shipConf structs.UrbitDocker) error {
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "rebuildContainer", Event: "loading"}
	if err := urbitCleanDelete(patp); err != nil {
		logger.Logger.Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
	}
	if shipConf.BootStatus != "noboot" {
		_, err := docker.StartContainer(patp, "vere")
		if err != nil {
			docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "rebuildContainer", Event: "error"}
			time.Sleep(3 * time.Second)
			docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "rebuildContainer", Event: ""}
			return fmt.Errorf("Failed to start for rebuild container %s: %v", patp, err)
		}
	} else {
		_, err := docker.CreateContainer(patp, "vere")
		if err != nil {
			docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "rebuildContainer", Event: "error"}
			time.Sleep(3 * time.Second)
			docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "rebuildContainer", Event: ""}
			return fmt.Errorf("Failed to create for rebuild container %s: %v", patp, err)
		}
	}
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "rebuildContainer", Event: "success"}
	time.Sleep(3 * time.Second)
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "rebuildContainer", Event: ""}
	return nil
}

func toggleNetwork(patp string, shipConf structs.UrbitDocker) error {
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
		if err := urbitCleanDelete(patp); err != nil {
			logger.Logger.Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
		}
	} else if currentNetwork != "wireguard" && conf.WgRegistered == true {
		shipConf.Network = "wireguard"
		update := make(map[string]structs.UrbitDocker)
		update[patp] = shipConf
		if err := config.UpdateUrbitConfig(update); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %v", err)
		}
		if err := urbitCleanDelete(patp); err != nil {
			logger.Logger.Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
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
}

func toggleBootStatus(patp string, shipConf structs.UrbitDocker) error {
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
}

func toggleMinIOLink(patp string, shipConf structs.UrbitDocker) error {
	// todo: scry for actual info
	isLinked := shipConf.MinIOLinked
	if isLinked {
		// unlink from urbit
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "toggleMinIOLink", Event: "unlinking"}
		if err := click.UnlinkStorage(patp); err != nil {
			return fmt.Errorf("Failed to unlink MinIO information %s: %v", patp, err)
		}

		// Update config
		update := make(map[string]structs.UrbitDocker)
		shipConf.MinIOLinked = false
		update[patp] = shipConf
		if err := config.UpdateUrbitConfig(update); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %v", err)
		}

		// Success
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "toggleMinIOLink", Event: "unlink-success"}
		time.Sleep(1 * time.Second)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "toggleMinIOLink", Event: ""}
	} else {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "toggleMinIOLink", Event: "linking"}
		// create service account
		svcAccount, err := docker.CreateMinIOServiceAccount(patp)
		if err != nil {
			return fmt.Errorf("Failed to create MinIO service account for %s: %v", patp, err)
		}
		// get minio endpoint
		var endpoint string
		endpoint = shipConf.CustomS3Web
		if endpoint == "" {
			endpoint = fmt.Sprintf("s3.%s", shipConf.WgURL)
		}
		// link to urbit
		if err := click.LinkStorage(patp, endpoint, svcAccount); err != nil {
			return fmt.Errorf("Failed to link MinIO information %s: %v", patp, err)
		}

		// Update config
		update := make(map[string]structs.UrbitDocker)
		shipConf.MinIOLinked = true
		update[patp] = shipConf
		if err := config.UpdateUrbitConfig(update); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %v", err)
		}

		// Success
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "toggleMinIOLink", Event: "success"}
		time.Sleep(1 * time.Second)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "toggleMinIOLink", Event: ""}
	}
	return nil
}

func handleLoom(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "loom", Event: "loading"}
	shipConf.LoomSize = urbitPayload.Payload.Value
	update := make(map[string]structs.UrbitDocker)
	update[patp] = shipConf
	if err := config.UpdateUrbitConfig(update); err != nil {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "loom", Event: "error"}
		time.Sleep(3 * time.Second)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "loom", Event: "done"}
		time.Sleep(1 * time.Second)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "loom", Event: ""}
		return fmt.Errorf("Couldn't update urbit config: %v", err)
	}
	if err := urbitCleanDelete(patp); err != nil {
		logger.Logger.Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
	}
	if shipConf.BootStatus == "boot" {
		if _, err := docker.StartContainer(patp, "vere"); err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't start %v: %v", patp, err))
		}
	}
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "loom", Event: "success"}
	time.Sleep(3 * time.Second)
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "loom", Event: "done"}
	time.Sleep(1 * time.Second)
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "loom", Event: ""}
	return nil
}

func schedulePack(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	frequency := urbitPayload.Payload.Frequency
	// frequency not 0
	if frequency < 1 {
		return fmt.Errorf("pack frequency cannot be 0!")
	}
	intervalType := urbitPayload.Payload.IntervalType
	switch intervalType {
	case "month", "week", "day":
		shipConf.MeldTime = urbitPayload.Payload.Time
		shipConf.MeldSchedule = true
		shipConf.MeldScheduleType = intervalType
		shipConf.MeldFrequency = frequency
		shipConf.MeldDay = urbitPayload.Payload.Day
		shipConf.MeldDate = urbitPayload.Payload.Date
		update := make(map[string]structs.UrbitDocker)
		update[patp] = shipConf
		if err := config.UpdateUrbitConfig(update); err != nil {
			return fmt.Errorf("Failed to update pack schedule: %v", err)
		}
	default:
		return fmt.Errorf("Schedule pack unknown interval type: %v", intervalType)
	}
	broadcast.SchedulePackBus <- "schedule"
	return nil
}

func pausePackSchedule(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	shipConf.MeldSchedule = false
	update := make(map[string]structs.UrbitDocker)
	update[patp] = shipConf
	if err := config.UpdateUrbitConfig(update); err != nil {
		return fmt.Errorf("Failed to pause pack schedule: %v", err)
	}
	return nil
}

func setNewMaxPierSize(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	shipConf.SizeLimit = urbitPayload.Payload.Value
	update := make(map[string]structs.UrbitDocker)
	update[patp] = shipConf
	if err := config.UpdateUrbitConfig(update); err != nil {
		return fmt.Errorf("Failed to set new size limit for %s: %v", patp, err)
	}
	return nil
}

func rollChopPier(patp string, shipConf structs.UrbitDocker) error {
	rollChopError := func(err error) error {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "rollChop", Event: "error"}
		return err
	}
	// clear transition after end
	defer func() {
		time.Sleep(3 * time.Second)
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "rollChop", Event: ""}
	}()
	statuses, err := docker.GetShipStatus([]string{patp})
	if err != nil {
		return rollChopError(fmt.Errorf("Failed to get ship status for %p: %v", patp, err))
	}
	status, exists := statuses[patp]
	if !exists {
		return rollChopError(fmt.Errorf("Failed to get ship status for %p: status doesn't exist!", patp))
	}
	isRunning := strings.Contains(status, "Up")
	// stop ship
	if isRunning {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "rollChop", Event: "stopping"}
		if err := click.BarExit(patp); err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to stop ship with |exit for roll & chop %s: %v", patp, err))
			if err = docker.StopContainerByName(patp); err != nil {
				logger.Logger.Error(fmt.Sprintf("Failed to stop ship for roll & chop %s: %v", patp, err))
			}
		}
	}
	// start ship as roll
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "rollChop", Event: "rolling"}
	logger.Logger.Info(fmt.Sprintf("Attempting to roll %s", patp))
	shipConf.BootStatus = "roll"
	update := make(map[string]structs.UrbitDocker)
	update[patp] = shipConf
	err = config.UpdateUrbitConfig(update)
	if err != nil {
		return rollChopError(fmt.Errorf("Failed to update %s urbit config to roll: %v", patp, err))
	}
	_, err = docker.StartContainer(patp, "vere")
	if err != nil {
		return rollChopError(fmt.Errorf("Failed to roll %s: %v", patp, err))
	}

	logger.Logger.Info(fmt.Sprintf("Waiting for roll to complete for %s", patp))
	waitComplete(patp)

	// start ship as chop
	logger.Logger.Info(fmt.Sprintf("Attempting to chop %s", patp))
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "rollChop", Event: "chopping"}
	shipConf.BootStatus = "chop"
	update = make(map[string]structs.UrbitDocker)
	update[patp] = shipConf
	err = config.UpdateUrbitConfig(update)
	if err != nil {
		return rollChopError(fmt.Errorf("Failed to update %s urbit config to chop: %v", patp, err))
	}
	_, err = docker.StartContainer(patp, "vere")
	if err != nil {
		return rollChopError(fmt.Errorf("Failed to chop %s: %v", patp, err))
	}

	logger.Logger.Info(fmt.Sprintf("Waiting for chop to complete for %s", patp))
	waitComplete(patp)

	// start ship if "boot"
	if isRunning {
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "rollChop", Event: "starting"}
		shipConf.BootStatus = "boot"
		update := make(map[string]structs.UrbitDocker)
		update[patp] = shipConf
		err := config.UpdateUrbitConfig(update)
		if err != nil {
			return rollChopError(fmt.Errorf("Failed to update %s urbit config to chop: %v", patp, err))
		}
		_, err = docker.StartContainer(patp, "vere")
		if err != nil {
			return rollChopError(fmt.Errorf("Failed to chop %s: %v", patp, err))
		}
	}
	docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "rollChop", Event: "success"}
	return nil
}
