package handler

import (
	"context"
	"fmt"
	"groundseg/broadcast"
	"groundseg/chopsvc"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/exporter"
	"groundseg/shipcleanup"
	"groundseg/startram"
	"groundseg/structs"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

func installPenpaiCompanion(patp string, shipConf structs.UrbitDocker) error {
	return runDeskTransition(patp, "penpaiCompanion", func() error {
		status, err := click.GetDesk(patp, "penpai", true)
		if err != nil {
			return fmt.Errorf("failed to get penpai desk info: %w", err)
		}
		switch status {
		case "not-found":
			if err := click.InstallDesk(patp, "~nattyv", "penpai"); err != nil {
				return fmt.Errorf("failed to install penpai desk: %w", err)
			}
		case "suspended":
			if err := click.ReviveDesk(patp, "penpai"); err != nil {
				return fmt.Errorf("failed to revive penpai desk: %w", err)
			}
		case "running":
			return nil
		}
		if err := waitForDeskState(patp, "penpai", "running", true); err != nil {
			return fmt.Errorf("failed waiting for penpai desk installation: %w", err)
		}
		return nil
	})
}

func uninstallPenpaiCompanion(patp string, shipConf structs.UrbitDocker) error {
	return runDeskTransition(patp, "penpaiCompanion", func() error {
		if err := click.UninstallDesk(patp, "penpai"); err != nil {
			return fmt.Errorf("failed to uninstall penpai desk: %w", err)
		}
		if err := waitForDeskState(patp, "penpai", "running", false); err != nil {
			return fmt.Errorf("failed waiting for penpai desk removal: %w", err)
		}
		return nil
	})
}

func installGallseg(patp string, shipConf structs.UrbitDocker) error {
	return runDeskTransition(patp, "gallseg", func() error {
		status, err := click.GetDesk(patp, "groundseg", true)
		if err != nil {
			return fmt.Errorf("failed to get groundseg desk info: %w", err)
		}
		switch status {
		case "not-found":
			if err := click.InstallDesk(patp, "~nattyv", "groundseg"); err != nil {
				return fmt.Errorf("failed to install groundseg desk: %w", err)
			}
		case "suspended":
			if err := click.ReviveDesk(patp, "groundseg"); err != nil {
				return fmt.Errorf("failed to revive groundseg desk: %w", err)
			}
		case "running":
			return nil
		}
		if err := waitForDeskState(patp, "groundseg", "running", true); err != nil {
			return fmt.Errorf("failed waiting for groundseg desk installation: %w", err)
		}
		return nil
	})
}

func uninstallGallseg(patp string, shipConf structs.UrbitDocker) error {
	return runDeskTransition(patp, "gallseg", func() error {
		if err := click.UninstallDesk(patp, "groundseg"); err != nil {
			return fmt.Errorf("failed to uninstall groundseg desk: %w", err)
		}
		if err := waitForDeskState(patp, "groundseg", "running", false); err != nil {
			return fmt.Errorf("failed waiting for groundseg desk removal: %w", err)
		}
		return nil
	})
}

func runDeskTransition(patp, transitionType string, operation func() error) error {
	return runTransitionedOperation(patp, transitionType, "loading", "success", 3*time.Second, operation)
}

func waitForDeskState(patp, desk, expectedState string, shouldMatch bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return pollWithTimeout(ctx, 5*time.Second, func() (bool, error) {
		status, err := click.GetDesk(patp, desk, true)
		if err != nil {
			return false, fmt.Errorf("get %s desk status for %s: %w", desk, patp, err)
		}
		if shouldMatch {
			return status == expectedState, nil
		}
		return status != expectedState, nil
	})
}

func startramReminder(patp string, remind bool) error {
	if err := config.UpdateUrbit(patp, func(conf *structs.UrbitDocker) error {
		conf.StartramReminder = remind
		return nil
	}); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %w", err)
	}
	return nil
}

func urbitDeleteStartramService(patp string, service string, shipConf structs.UrbitDocker) error {
	settings := config.StartramSettingsSnapshot()
	// check svc type, reconstruct subdomain

	// Accessing parts of the URL
	parts := strings.Split(settings.EndpointURL, ".")
	if len(parts) < 2 {
		return fmt.Errorf("Failed to recreate subdomain for manual service deletion")
	} else {
		baseURL := parts[len(parts)-2] + "." + parts[len(parts)-1]
		var subdomain string
		switch service {
		case "urbit-web":
			subdomain = fmt.Sprintf("%s.%s", patp, baseURL)
		case "urbit-ames":
			subdomain = fmt.Sprintf("%s.%s.%s", "ames", patp, baseURL)
		case "minio":
			subdomain = fmt.Sprintf("%s.%s.%s", "s3", patp, baseURL)
		case "minio-console":
			subdomain = fmt.Sprintf("%s.%s.%s", "console.s3", patp, baseURL)
		case "minio-bucket":
			subdomain = fmt.Sprintf("%s.%s.%s", "bucket.s3", patp, baseURL)
		default:
			return fmt.Errorf("Invalid service type: unable to manually delete service")
		}
		if err := startram.SvcDelete(subdomain, service); err != nil {
			return fmt.Errorf("Failed to delete startram service: %w", err)
		} else {
			_, err := startram.SyncRetrieve()
			if err != nil {
				return fmt.Errorf("Failed to retrieve after manual service deletion: %w", err)
			}
		}
		return nil
	}
}

func packPier(patp string, shipConf structs.UrbitDocker) error {
	// error handling
	packError := func(err error) error {
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "pack", Event: "error"})
		return err
	}
	// clear transition after end
	defer func() {
		time.Sleep(3 * time.Second)
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "pack", Event: ""})
	}()
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "pack", Event: "packing"})
	statuses, err := docker.GetShipStatus([]string{patp})
	if err != nil {
		return packError(fmt.Errorf("Failed to get ship status for %s: %v", patp, err))
	}
	status, exists := statuses[patp]
	if !exists {
		return packError(fmt.Errorf("Failed to get ship status for %s: status doesn't exist!", patp))
	}
	// running
	if strings.Contains(status, "Up") {
		// send |pack
		if err := click.SendPack(patp); err != nil {
			return packError(fmt.Errorf("Failed to |pack to %s: %v", patp, err))
		}
		// not running
	} else {
		// set DesiredStatus to prevent auto-restart when pack container exits
		if containerState, exists := config.GetContainerState()[patp]; exists {
			containerState.DesiredStatus = "stopped"
			config.UpdateContainerState(patp, containerState)
		}
		// switch boot status to pack
		shipConf.BootStatus = "pack"
		err := persistShipConf(patp, shipConf)
		if err != nil {
			return packError(fmt.Errorf("Failed to update %s urbit config to pack: %v", patp, err))
		}
		_, err = docker.StartContainer(patp, "vere")
		if err != nil {
			return packError(fmt.Errorf("Failed to urth pack %s: %v", patp, err))
		}
		// wait for pack to complete before marking success
		if err := WaitComplete(patp); err != nil {
			return packError(fmt.Errorf("Failed waiting for pack completion on %s: %w", patp, err))
		}
	}
	// set last meld
	now := time.Now().Unix()
	shipConf.MeldLast = strconv.FormatInt(now, 10)
	err = persistShipConf(patp, shipConf)
	if err != nil {
		return packError(fmt.Errorf("Failed to update %s urbit config with last meld time: %v", patp, err))
	}
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "pack", Event: "success"})
	return nil
}

func packMeldPier(patp string, shipConf structs.UrbitDocker) error {
	packMeldError := func(err error) error {
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "packMeld", Event: "error"})
		return err
	}
	// clear transition after end
	defer func() {
		time.Sleep(3 * time.Second)
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "packMeld", Event: ""})
	}()
	statuses, err := docker.GetShipStatus([]string{patp})
	if err != nil {
		return packMeldError(fmt.Errorf("Failed to get ship status for %s: %v", patp, err))
	}
	status, exists := statuses[patp]
	if !exists {
		return packMeldError(fmt.Errorf("Failed to get ship status for %s: status doesn't exist!", patp))
	}
	isRunning := strings.Contains(status, "Up")
	// set DesiredStatus to prevent auto-restart from die/stop event handlers during maintenance
	if containerState, exists := config.GetContainerState()[patp]; exists {
		containerState.DesiredStatus = "stopped"
		config.UpdateContainerState(patp, containerState)
	}
	if isRunning {
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "packMeld", Event: "stopping"})
		if err := click.BarExit(patp); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to stop ship with |exit for pack & meld %s: %v", patp, err))
			if err = docker.StopContainerByName(patp); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to stop ship for pack & meld %s: %v", patp, err))
			}
		}
		if err := WaitComplete(patp); err != nil {
			return packMeldError(fmt.Errorf("Failed waiting for stop completion on %s before pack & meld: %w", patp, err))
		}
	}
	// stop ship
	// start ship as pack
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "packMeld", Event: "packing"})
	zap.L().Info(fmt.Sprintf("Attempting to urth pack %s", patp))
	shipConf.BootStatus = "pack"
	err = persistShipConf(patp, shipConf)
	if err != nil {
		return packMeldError(fmt.Errorf("Failed to update %s urbit config to pack: %v", patp, err))
	}
	_, err = docker.StartContainer(patp, "vere")
	if err != nil {
		return packMeldError(fmt.Errorf("Failed to urth pack %s: %v", patp, err))
	}

	zap.L().Info(fmt.Sprintf("Waiting for urth pack to complete for %s", patp))
	if err := WaitComplete(patp); err != nil {
		return packMeldError(fmt.Errorf("Failed waiting for pack completion on %s: %w", patp, err))
	}

	// start ship as meld
	zap.L().Info(fmt.Sprintf("Attempting to urth meld %s", patp))
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "packMeld", Event: "melding"})
	shipConf.BootStatus = "meld"
	err = persistShipConf(patp, shipConf)
	if err != nil {
		return packMeldError(fmt.Errorf("Failed to update %s urbit config to meld: %v", patp, err))
	}
	_, err = docker.StartContainer(patp, "vere")
	if err != nil {
		return packMeldError(fmt.Errorf("Failed to urth meld %s: %v", patp, err))
	}

	zap.L().Info(fmt.Sprintf("Waiting for urth meld to complete for %s", patp))
	if err := WaitComplete(patp); err != nil {
		return packMeldError(fmt.Errorf("Failed waiting for meld completion on %s: %w", patp, err))
	}

	// start ship if "boot"
	if isRunning {
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "packMeld", Event: "starting"})
		// restore DesiredStatus so normal auto-restart behavior resumes
		if containerState, exists := config.GetContainerState()[patp]; exists {
			containerState.DesiredStatus = "running"
			config.UpdateContainerState(patp, containerState)
		}
		shipConf.BootStatus = "boot"
		err := persistShipConf(patp, shipConf)
		if err != nil {
			return packMeldError(fmt.Errorf("Failed to update %s urbit config to meld: %v", patp, err))
		}
		_, err = docker.StartContainer(patp, "vere")
		if err != nil {
			return packMeldError(fmt.Errorf("Failed to urth meld %s: %v", patp, err))
		}
	}
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "packMeld", Event: "success"})
	return nil
}

func toggleAlias(patp string, shipConf structs.UrbitDocker) error {
	if shipConf.ShowUrbitWeb == "custom" {
		shipConf.ShowUrbitWeb = "default"
	} else {
		shipConf.ShowUrbitWeb = "custom"
	}
	if err := persistShipConf(patp, shipConf); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %w", err)
	}
	return nil
}

func setUrbitDomain(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	return runTransitionedOperation(patp, "urbitDomain", "loading", "done", time.Second, func() error {
		alias := urbitPayload.Payload.Domain
		oldDomain := shipConf.WgURL
		areAliases, err := AreSubdomainsAliases(alias, oldDomain)
		if err != nil {
			return fmt.Errorf("Failed to check Urbit domain alias for %s: %v", patp, err)
		}
		if !areAliases {
			return fmt.Errorf("Invalid Urbit domain alias for %s", patp)
		}
		if err := startram.AliasCreate(patp, alias); err != nil {
			return err
		}
		shipConf.CustomUrbitWeb = alias
		shipConf.ShowUrbitWeb = "custom"
		if err := persistShipConf(patp, shipConf); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		}
		emitUrbitTransition(patp, "urbitDomain", "success")
		sleepForWorkflow(3 * time.Second)
		return nil
	})
}

func setMinIODomain(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	return runTransitionedOperation(patp, "minioDomain", "loading", "done", time.Second, func() error {
		alias := urbitPayload.Payload.Domain
		oldDomain := fmt.Sprintf("s3.%s", shipConf.WgURL)
		areAliases, err := AreSubdomainsAliases(alias, oldDomain)
		if err != nil {
			return fmt.Errorf("Failed to check MinIO domain alias for %s: %v", patp, err)
		}
		if !areAliases {
			return fmt.Errorf("Invalid MinIO domain alias for %s", patp)
		}
		if err := startram.AliasCreate(fmt.Sprintf("s3.%s", patp), alias); err != nil {
			return err
		}
		shipConf.CustomS3Web = alias
		if err := persistShipConf(patp, shipConf); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		}
		emitUrbitTransition(patp, "minioDomain", "success")
		sleepForWorkflow(3 * time.Second)
		return nil
	})
}

func ChopPier(patp string, shipConf structs.UrbitDocker) error {
	return chopsvc.ChopPier(patp, shipConf)
}

func toggleChopOnVereUpdate(patp string, shipConf structs.UrbitDocker) error {
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "chopOnUpgrade", Event: "loading"})
	defer func() {
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "chopOnUpgrade", Event: ""})
	}()
	if shipConf.ChopOnUpgrade == false {
		shipConf.ChopOnUpgrade = true
	} else {
		shipConf.ChopOnUpgrade = false
	}
	if err := persistShipConf(patp, shipConf); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %w", err)
	}
	return nil
}

func deleteShip(patp string, shipConf structs.UrbitDocker) error {
	settings := config.StartramSettingsSnapshot()
	// update DesiredStatus to 'stopped'
	contConf := config.GetContainerState()
	patpConf := contConf[patp]
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "deleteShip", Event: "stopping"})
	patpConf.DesiredStatus = "stopped"
	contConf[patp] = patpConf
	config.UpdateContainerState(patp, patpConf)
	if err := click.BarExit(patp); err != nil {
		zap.L().Error(fmt.Sprintf("%v", err))
		if err := docker.StopContainerByName(patp); err != nil {
			return fmt.Errorf("Couldn't stop docker container for %v: %v", patp, err)
		}
	}
	if err := docker.DeleteContainer(patp); err != nil {
		return fmt.Errorf("Couldn't delete docker container for %v: %v", patp, err)
	}
	if settings.WgRegistered {
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "deleteShip", Event: "removing-services"})
		if err := startram.SvcDelete(patp, "urbit"); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't remove urbit anchor for %v: %v", patp, err))
		}
		if err := startram.SvcDelete("s3."+patp, "s3"); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't remove s3 anchor for %v: %v", patp, err))
		}
		if err := docker.DeleteContainer("minio_" + patp); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't delete minio docker container for %v: %v", patp, err))
		}
	}
	// get custom directory info before deleting config
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "deleteShip", Event: "deleting"})
	customPath := ""
	if customLoc, ok := shipConf.CustomPierLocation.(string); ok {
		customPath = customLoc
	}
	if err := shipcleanup.RollbackProvisioning(patp, shipcleanup.RollbackOptions{
		CustomPierPath:       customPath,
		RemoveContainerState: true,
	}); err != nil {
		zap.L().Error(fmt.Sprintf("Ship cleanup encountered errors for %v: %v", patp, err))
	}
	zap.L().Info(fmt.Sprintf("%v container deleted", patp))
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "deleteShip", Event: "success"})
	time.Sleep(3 * time.Second)
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "deleteShip", Event: "done"})

	time.Sleep(1 * time.Second)
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "deleteShip", Event: ""})
	// remove from broadcast
	if err := broadcast.ReloadUrbits(); err != nil {
		zap.L().Error(fmt.Sprintf("Error updating broadcast: %v", err))
	}
	return nil
}

func exportShip(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "exportShip", Event: "stopping"})
	shipConf.BootStatus = "noboot"
	if err := persistShipConf(patp, shipConf); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %w", err)
	}
	// stop container
	if err := click.BarExit(patp); err != nil {
		zap.L().Error(fmt.Sprintf("%v", err))
		if err := docker.StopContainerByName(patp); err != nil {
			return err
		}
	}
	// whitelist the patp token pair
	if err := exporter.WhitelistContainer(patp, urbitPayload.Token); err != nil {
		return err
	}
	// transition: ready
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "exportShip", Event: "ready"})
	return nil
}

func exportBucket(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	containerName := fmt.Sprintf("minio_%s", patp)
	// whitelist the patp token pair
	if err := exporter.WhitelistContainer(containerName, urbitPayload.Token); err != nil {
		return err
	}
	// transition: ready
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "exportBucket", Event: "ready"})
	return nil
}

func togglePower(patp string, shipConf structs.UrbitDocker) error {
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "togglePower", Event: "loading"})
	defer func() {
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "togglePower", Event: ""})
	}()
	statuses, err := docker.GetShipStatus([]string{patp})
	if err != nil {
		return fmt.Errorf("Failed to get ship status for %s: %v", patp, err)
	}
	status, exists := statuses[patp]
	if !exists {
		return fmt.Errorf("Failed to get ship status for %s: status doesn't exist!", patp)
	}
	isRunning := strings.Contains(status, "Up")
	if shipConf.BootStatus == "noboot" {
		shipConf.BootStatus = "boot"
		if err := persistShipConf(patp, shipConf); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		}
		_, err := docker.StartContainer(patp, "vere")
		if err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		}
	} else if shipConf.BootStatus == "boot" && isRunning {
		// set DesiredStatus before stopping to prevent auto-restart from die/stop event handlers
		if containerState, exists := config.GetContainerState()[patp]; exists {
			containerState.DesiredStatus = "stopped"
			config.UpdateContainerState(patp, containerState)
		}
		shipConf.BootStatus = "noboot"
		if err := persistShipConf(patp, shipConf); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		}
		err := click.BarExit(patp)
		if err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
			if err := docker.StopContainerByName(patp); err != nil {
				zap.L().Error(fmt.Sprintf("%v", err))
			}
		}
	} else if shipConf.BootStatus == "boot" && !isRunning {
		_, err := docker.StartContainer(patp, "vere")
		if err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		}
	}
	return nil
}

func toggleDevMode(patp string, shipConf structs.UrbitDocker) error {
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "toggleDevMode", Event: "loading"})
	defer func() {
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "toggleDevMode", Event: ""})
	}()
	if shipConf.DevMode == true {
		shipConf.DevMode = false
	} else {
		shipConf.DevMode = true
	}
	if err := persistShipConf(patp, shipConf); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %w", err)
	}
	if err := urbitCleanDelete(patp); err != nil {
		zap.L().Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
	}
	_, err := docker.StartContainer(patp, "vere")
	if err != nil {
		zap.L().Error(fmt.Sprintf("%v", err))
	}
	return nil
}

func rebuildContainer(patp string, shipConf structs.UrbitDocker) error {
	return runTransitionedOperation(patp, "rebuildContainer", "loading", "success", 3*time.Second, func() error {
		if err := urbitCleanDelete(patp); err != nil {
			zap.L().Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
		}
		if shipConf.BootStatus != "noboot" {
			if _, err := docker.StartContainer(patp, "vere"); err != nil {
				return fmt.Errorf("Failed to start for rebuild container %s: %v", patp, err)
			}
			return nil
		}
		if _, err := docker.CreateContainer(patp, "vere"); err != nil {
			return fmt.Errorf("Failed to create for rebuild container %s: %v", patp, err)
		}
		return nil
	})
}

func toggleNetwork(patp string, shipConf structs.UrbitDocker) error {
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "toggleNetwork", Event: "loading"})
	defer func() {
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "toggleNetwork", Event: ""})
	}()
	currentNetwork := shipConf.Network
	settings := config.StartramSettingsSnapshot()
	zap.L().Warn(fmt.Sprintf("%v", currentNetwork))
	if currentNetwork == "wireguard" {
		shipConf.Network = "bridge"
		if err := persistShipConf(patp, shipConf); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		}
		if err := urbitCleanDelete(patp); err != nil {
			zap.L().Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
		}
	} else if currentNetwork != "wireguard" && settings.WgRegistered {
		shipConf.Network = "wireguard"
		if err := persistShipConf(patp, shipConf); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		}
		if err := urbitCleanDelete(patp); err != nil {
			zap.L().Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
		}
	} else {
		return fmt.Errorf("No remote registration")
	}
	if shipConf.BootStatus == "boot" {
		if _, err := docker.StartContainer(patp, "vere"); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't start %v: %v", patp, err))
		}
	}
	return nil
}

func toggleBootStatus(patp string, shipConf structs.UrbitDocker) error {
	if shipConf.BootStatus == "ignore" {
		statusMap, err := docker.GetShipStatus([]string{patp})
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to get ship status for %s", patp))
		}
		status, exists := statusMap[patp]
		if !exists {
			zap.L().Error(fmt.Sprintf("Running status for %s doesn't exist", patp))
		}
		if strings.Contains(status, "Up") {
			shipConf.BootStatus = "boot"
		} else {
			shipConf.BootStatus = "noboot"
		}
	} else {
		shipConf.BootStatus = "ignore"
	}
	if err := persistShipConf(patp, shipConf); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %w", err)
	}
	return nil
}

func toggleAutoReboot(patp string, shipConf structs.UrbitDocker) error {
	if err := config.LoadUrbitConfig(patp); err != nil {
		return fmt.Errorf("Failed to load fresh urbit config: %w", err)
	}
	if boolValue, ok := shipConf.DisableShipRestarts.(bool); ok {
		shipConf.DisableShipRestarts = !boolValue
	} else {
		shipConf.DisableShipRestarts = true
	}
	if err := persistShipConf(patp, shipConf); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %w", err)
	}
	broadcast.BroadcastToClients()
	return nil
}

func toggleMinIOLink(patp string, shipConf structs.UrbitDocker) error {
	// todo: scry for actual info
	isLinked := shipConf.MinIOLinked
	if isLinked {
		// unlink from urbit
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "toggleMinIOLink", Event: "unlinking"})
		if err := click.UnlinkStorage(patp); err != nil {
			return fmt.Errorf("Failed to unlink MinIO information %s: %v", patp, err)
		}

		// Update config
		shipConf.MinIOLinked = false
		if err := persistShipConf(patp, shipConf); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		}

		// Success
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "toggleMinIOLink", Event: "unlink-success"})
		time.Sleep(1 * time.Second)
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "toggleMinIOLink", Event: ""})
	} else {
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "toggleMinIOLink", Event: "linking"})
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
		shipConf.MinIOLinked = true
		if err := persistShipConf(patp, shipConf); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		}

		// Success
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "toggleMinIOLink", Event: "success"})
		time.Sleep(1 * time.Second)
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "toggleMinIOLink", Event: ""})
	}
	return nil
}

func handleLoom(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	return runTransitionedOperation(patp, "loom", "loading", "done", time.Second, func() error {
		shipConf.LoomSize = urbitPayload.Payload.Value
		if err := persistShipConf(patp, shipConf); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		}
		if err := urbitCleanDelete(patp); err != nil {
			zap.L().Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
		}
		if shipConf.BootStatus == "boot" {
			if _, err := docker.StartContainer(patp, "vere"); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't start %v: %v", patp, err))
			}
		}
		emitUrbitTransition(patp, "loom", "success")
		sleepForWorkflow(3 * time.Second)
		return nil
	})
}

func handleSnapTime(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	return runTransitionedOperation(patp, "snapTime", "loading", "done", time.Second, func() error {
		shipConf.SnapTime = urbitPayload.Payload.Value
		if err := persistShipConf(patp, shipConf); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %w", err)
		}
		if err := urbitCleanDelete(patp); err != nil {
			zap.L().Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
		}
		if shipConf.BootStatus == "boot" {
			if _, err := docker.StartContainer(patp, "vere"); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't start %v: %v", patp, err))
			}
		}
		emitUrbitTransition(patp, "snapTime", "success")
		sleepForWorkflow(3 * time.Second)
		return nil
	})
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
		if err := persistShipConf(patp, shipConf); err != nil {
			return fmt.Errorf("Failed to update pack schedule: %w", err)
		}
	default:
		return fmt.Errorf("Schedule pack unknown interval type: %v", intervalType)
	}
	broadcast.PublishSchedulePack("schedule")
	return nil
}

func pausePackSchedule(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	shipConf.MeldSchedule = false
	if err := persistShipConf(patp, shipConf); err != nil {
		return fmt.Errorf("Failed to pause pack schedule: %w", err)
	}
	return nil
}

func setNewMaxPierSize(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	shipConf.SizeLimit = urbitPayload.Payload.Value
	if err := persistShipConf(patp, shipConf); err != nil {
		return fmt.Errorf("Failed to set new size limit for %s: %v", patp, err)
	}
	return nil
}

func rollChopPier(patp string, shipConf structs.UrbitDocker) error {
	rollChopError := func(err error) error {
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "rollChop", Event: "error"})
		return err
	}
	// clear transition after end
	defer func() {
		time.Sleep(3 * time.Second)
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "rollChop", Event: ""})
	}()
	statuses, err := docker.GetShipStatus([]string{patp})
	if err != nil {
		return rollChopError(fmt.Errorf("Failed to get ship status for %s: %v", patp, err))
	}
	status, exists := statuses[patp]
	if !exists {
		return rollChopError(fmt.Errorf("Failed to get ship status for %s: status doesn't exist!", patp))
	}
	isRunning := strings.Contains(status, "Up")
	// stop ship
	if isRunning {
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "rollChop", Event: "stopping"})
		if err := click.BarExit(patp); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to stop ship with |exit for roll & chop %s: %v", patp, err))
			if err = docker.StopContainerByName(patp); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to stop ship for roll & chop %s: %v", patp, err))
			}
		}
		if err := WaitComplete(patp); err != nil {
			return rollChopError(fmt.Errorf("Failed waiting for stop completion on %s before roll & chop: %w", patp, err))
		}
	}
	// start ship as roll
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "rollChop", Event: "rolling"})
	zap.L().Info(fmt.Sprintf("Attempting to roll %s", patp))
	shipConf.BootStatus = "roll"
	err = persistShipConf(patp, shipConf)
	if err != nil {
		return rollChopError(fmt.Errorf("Failed to update %s urbit config to roll: %v", patp, err))
	}
	_, err = docker.StartContainer(patp, "vere")
	if err != nil {
		return rollChopError(fmt.Errorf("Failed to roll %s: %v", patp, err))
	}

	zap.L().Info(fmt.Sprintf("Waiting for roll to complete for %s", patp))
	if err := WaitComplete(patp); err != nil {
		return rollChopError(fmt.Errorf("Failed waiting for roll completion on %s: %w", patp, err))
	}

	// start ship as chop
	zap.L().Info(fmt.Sprintf("Attempting to chop %s", patp))
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "rollChop", Event: "chopping"})
	shipConf.BootStatus = "chop"
	err = persistShipConf(patp, shipConf)
	if err != nil {
		return rollChopError(fmt.Errorf("Failed to update %s urbit config to chop: %v", patp, err))
	}
	_, err = docker.StartContainer(patp, "vere")
	if err != nil {
		return rollChopError(fmt.Errorf("Failed to chop %s: %v", patp, err))
	}

	zap.L().Info(fmt.Sprintf("Waiting for chop to complete for %s", patp))
	if err := WaitComplete(patp); err != nil {
		return rollChopError(fmt.Errorf("Failed waiting for chop completion on %s: %w", patp, err))
	}

	// start ship if "boot"
	if isRunning {
		docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "rollChop", Event: "starting"})
		shipConf.BootStatus = "boot"
		err := persistShipConf(patp, shipConf)
		if err != nil {
			return rollChopError(fmt.Errorf("Failed to update %s urbit config to chop: %v", patp, err))
		}
		_, err = docker.StartContainer(patp, "vere")
		if err != nil {
			return rollChopError(fmt.Errorf("Failed to chop %s: %v", patp, err))
		}
	}
	docker.PublishUrbitTransition(structs.UrbitTransition{Patp: patp, Type: "rollChop", Event: "success"})
	return nil
}
