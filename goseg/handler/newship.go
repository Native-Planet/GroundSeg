package handler

import (
	"fmt"
	"goseg/config"
	"goseg/docker"
	"goseg/structs"
	"time"
)

func resetNewShip() error {
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: ""}
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "patp", Event: ""}
	return nil
}

func createUrbitShip(patp string, shipPayload structs.WsNewShipPayload) {
	// todo: add a cleanup function to run if error is present
	// transition: starting
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: "starting"}
	// transition: patp
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "patp", Event: patp}
	// get unused http and ames ports
	httpPort, amesPort := getOpenUrbitPorts()
	config.Logger.Warn(fmt.Sprintf("%v %v", httpPort, amesPort)) // temp
	// update system.json
	err := updateSystemConfig(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		config.Logger.Error(errmsg)
	}
	// todo: create pier config

	config.Logger.Info(fmt.Sprintf("Preparing environment for pier: %v", patp))
	// delete container if exists
	err = docker.DeleteContainer(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		config.Logger.Error(errmsg)
	}
	// delete volume if exists
	err = docker.DeleteVolume(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		config.Logger.Error(errmsg)
	}
	// create new docker volume
	err = docker.CreateVolume(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		config.Logger.Error(errmsg)
	}
	// write key to volume
	key := shipPayload.Payload.Key
	err = docker.WriteFileToVolume(patp, patp+".key", key)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		config.Logger.Error(errmsg)
	}
	// todo: persist config
	time.Sleep(time.Second * time.Duration(2)) // temp

	// transition: creating
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: "creating"}
	config.Logger.Info(fmt.Sprintf("Creating Pier: %v", patp))
	// todo: create docker container
	time.Sleep(time.Second * time.Duration(5)) // temp
	// register startram
	// - condition: wgRegistered
	// toggle to remote
	// - condition: wgRegistered, remote set to true, service has been registered

	docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: "booting"}
	config.Logger.Info(fmt.Sprintf("Booting ship: %v", patp))

	time.Sleep(time.Second * time.Duration(10))

	// once +code is successful
	// transition: completed
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: "completed"}
	config.Logger.Info(fmt.Sprintf("Creating Pier: %v", patp))
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
	config.Logger.Info(fmt.Sprintf("Open Urbit Ports:  http: %v , ames: %v", httpPort, amesPort))
	return httpPort, amesPort
}

func updateSystemConfig(patp string) error {
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
		"Piers": piers,
	})
	if err != nil {
		return err
	}
	return nil
}
