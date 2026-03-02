package shipworkflow

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"groundseg/config"
	"groundseg/docker/events"
	"groundseg/docker/network"
	"groundseg/docker/orchestration"
	"groundseg/shipcleanup"
	"groundseg/shipcreator"
	"groundseg/startram"
	"groundseg/structs"
)

func ProvisionShip(patp string, shipPayload structs.WsNewShipPayload, customDrive string) error {
	err := shipcreator.CreateUrbitConfig(patp, customDrive)
	if err != nil {
		errmsg := fmt.Sprintf("failed to create urbit config: %v", err)
		zap.L().Error(errmsg)
		return handleNewShipErrorCleanup(patp, errmsg, customDrive)
	}

	if err = shipcreator.AppendSysConfigPier(patp); err != nil {
		errmsg := fmt.Sprintf("failed to add ship to system.json: %v", err)
		zap.L().Error(errmsg)
		return handleNewShipErrorCleanup(patp, errmsg, customDrive)
	}

	zap.L().Info(fmt.Sprintf("Preparing environment for pier: %v", patp))
	if err = orchestration.DeleteContainer(patp); err != nil {
		zap.L().Error(fmt.Sprintf("delete container error: %v", err))
	}
	if err = network.NewNetworkRuntime().DeleteVolume(patp); err != nil {
		zap.L().Error(fmt.Sprintf("delete volume error: %v", err))
	}

	if customDrive == "" {
		if err = network.NewNetworkRuntime().CreateVolume(patp); err != nil {
			errmsg := fmt.Sprintf("create volume error: %v", err)
			zap.L().Error(errmsg)
			return handleNewShipErrorCleanup(patp, errmsg, customDrive)
		}
		key := shipPayload.Payload.Key
		if err = network.NewNetworkRuntime().WriteFileToVolume(patp, patp+".key", key); err != nil {
			errmsg := fmt.Sprintf("write file to volume error: %v", err)
			zap.L().Error(errmsg)
			return handleNewShipErrorCleanup(patp, errmsg, customDrive)
		}
	} else {
		path := filepath.Join(customDrive, patp)
		filename := patp + ".key"
		key := shipPayload.Payload.Key
		if err = os.MkdirAll(path, os.ModePerm); err != nil {
			errmsg := fmt.Sprintf("write file to volume error: %v", err)
			zap.L().Error(errmsg)
			return handleNewShipErrorCleanup(patp, errmsg, customDrive)
		}
		filePath := path + "/" + filename
		if err = os.WriteFile(filePath, []byte(key), 0644); err != nil {
			errmsg := fmt.Sprintf("Error writing to file: %v", err)
			zap.L().Error(errmsg)
			return handleNewShipErrorCleanup(patp, errmsg, customDrive)
		}
	}

	zap.L().Info(fmt.Sprintf("Creating Pier: %v", patp))
	events.PublishNewShipTransition(structs.NewShipTransition{Type: "bootStage", Event: "creating"})
	info, err := orchestration.StartContainer(patp, "vere")
	if err != nil {
		errmsg := fmt.Sprintf("start container error: %v", err)
		zap.L().Error(errmsg)
		return handleNewShipErrorCleanup(patp, errmsg, customDrive)
	}
	config.UpdateContainerState(patp, info)

	conf := config.Conf()
	if conf.WgRegistered {
		go RegisterNewShipServices(patp)
	}
	if conf.PenpaiAllow {
		if err := orchestration.StopContainerByName("llama"); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't stop Llama: %v", err))
		}
		_, err := orchestration.StartContainer("llama", "llama")
		if err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't restart Llama: %v", err))
		}
	}

	go waitForNewShipReady(shipPayload, customDrive)
	return nil
}

func waitForNewShipReady(shipPayload structs.WsNewShipPayload, customDrive string) {
	patp := shipPayload.Payload.Patp
	remote := shipPayload.Payload.Remote

	events.PublishNewShipTransition(structs.NewShipTransition{Type: "bootStage", Event: "booting"})
	zap.L().Info(fmt.Sprintf("Booting ship: %v", patp))
	WaitForBootCode(patp, 1*time.Second)

	conf := config.Conf()
	if conf.WgRegistered && conf.WgOn && remote {
		events.PublishNewShipTransition(structs.NewShipTransition{Type: "bootStage", Event: "remote"})
		WaitForRemoteReady(patp, 1*time.Second)
		if err := SwitchShipToWireguard(patp, false); err != nil {
			errmsg := fmt.Sprintf("%v", err)
			zap.L().Error(errmsg)
			handleNewShipErrorCleanup(patp, errmsg, customDrive)
			return
		}
	}

	startram.SyncRetrieve()
	events.PublishNewShipTransition(structs.NewShipTransition{Type: "bootStage", Event: "completed"})
	if conf.PenpaiAllow {
		orchestration.StartContainer("llama-gpt-api", "llama-api")
	}
}

func RegisterNewShipServices(patp string) {
	if err := RegisterShipServices(patp); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to register StarTram service for %s: %v", patp, err))
	}
}

func handleNewShipErrorCleanup(patp, errmsg, customDrive string) error {
	events.PublishNewShipTransition(structs.NewShipTransition{Type: "bootStage", Event: "aborted"})
	events.PublishNewShipTransition(structs.NewShipTransition{Type: "error", Event: fmt.Sprintf("%v", errmsg)})
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
	return fmt.Errorf("%s", errmsg)
}
