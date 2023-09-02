package handler

import (
	"fmt"
	"goseg/config"
	"goseg/docker"
	"goseg/structs"
)

func createUrbitShip(patp string, shipPayload structs.WsNewShipPayload) {
	// transition: starting
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "bootStage", Event: "starting"}
	// transition: patp
	docker.NewShipTransBus <- structs.NewShipTransition{Type: "patp", Event: patp}
	// get unused http and ames ports
	//httpPort, amesPort := getOpenUrbitPorts()
	// update system.json
	err := updateSystemConfig(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		config.Logger.Error(errmsg)
	}
	// create pier config

	// delete container if exists

	// delete volume if exists

	// create new docker volume
	err = docker.CreateVolume(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		config.Logger.Error(errmsg)
	}

	// write key to volume

	// persist config

	// transition: creating

	// create docker container

	// register startram service (goroutine)

	// transition: registering

	// if remote is true, wait until fully booted to toggle to remote (goroutine)

	// once +code is successful

	// transition: completed

	//config.Logger.Info(fmt.Sprintf("%+v", shipPayload.Payload))
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
