package handler

import (
	"encoding/json"
	"fmt"
	"goseg/broadcast"
	"goseg/config"
	"goseg/docker"
	"goseg/logger"
	"goseg/startram"
	"goseg/structs"
	"time"
)

// startram action handler
// gonna get confusing if we have varied startram structs
func StartramHandler(msg []byte) error {
	var startramPayload structs.WsStartramPayload
	err := json.Unmarshal(msg, &startramPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal startram payload: %v", err)
	}
	switch startramPayload.Payload.Action {
	case "regions":
		go handleStartramRegions()
	case "register":
		regCode := startramPayload.Payload.Key
		region := startramPayload.Payload.Region
		go handleStartramRegister(regCode, region)
	case "toggle":
		go handleNotImplement(startramPayload.Payload.Action)
	case "restart":
		go handleNotImplement(startramPayload.Payload.Action)
	case "cancel":
		key := startramPayload.Payload.Key
		reset := startramPayload.Payload.Reset
		go handleStartramCancel(key, reset)
	case "endpoint":
		endpoint := startramPayload.Payload.Endpoint
		go handleStartramEndpoint(endpoint)

	default:
		return fmt.Errorf("Unrecognized startram action: %v", startramPayload.Payload.Action)
	}
	return nil
}

func handleStartramRegions() {
	go broadcast.LoadStartramRegions()
}

func handleStartramRegister(regCode, region string) {
	// error handling
	handleError := func(errmsg string) {
		msg := fmt.Sprintf("Error: %s", errmsg)
		logger.Logger.Error(errmsg)
		startram.EventBus <- structs.Event{Type: "register", Data: msg}
		time.Sleep(3 * time.Second)
		startram.EventBus <- structs.Event{Type: "register", Data: nil}
	}
	startram.EventBus <- structs.Event{Type: "register", Data: "key"}
	// Reset Key Pair
	err := config.CycleWgKey()
	if err != nil {
		handleError(fmt.Sprintf("%v", err))
		return
	}
	// Register startram key
	if err := startram.Register(regCode, region); err != nil {
		handleError(fmt.Sprintf("Failed registration: %v", err))
		return
	}
	// Register Services
	startram.EventBus <- structs.Event{Type: "register", Data: "services"}
	if err := startram.RegisterExistingShips(); err != nil {
		handleError(fmt.Sprintf("Unable to register ships: %v", err))
		return
	}
	// Start Wireguard
	startram.EventBus <- structs.Event{Type: "register", Data: "starting"}
	if err := docker.LoadWireguard(); err != nil {
		handleError(fmt.Sprintf("Unable to start Wireguard: %v", err))
		return
	}
	// Finish
	startram.EventBus <- structs.Event{Type: "register", Data: "complete"}

	// debug
	//time.Sleep(2 * time.Second)
	//handleError("Self inflicted error for debug purposes")

	// Clear
	time.Sleep(3 * time.Second)
	startram.EventBus <- structs.Event{Type: "register", Data: nil}
}

// endpoint action
func handleStartramEndpoint(endpoint string) {
	// error handling
	handleError := func(errmsg string) {
		msg := fmt.Sprintf("Error: %s", errmsg)
		startram.EventBus <- structs.Event{Type: "endpoint", Data: msg}
		time.Sleep(3 * time.Second)
		startram.EventBus <- structs.Event{Type: "endpoint", Data: nil}
	}
	// initialize
	startram.EventBus <- structs.Event{Type: "endpoint", Data: "init"}
	conf := config.Conf()
	// stop wireguard if running
	if conf.WgOn {
		startram.EventBus <- structs.Event{Type: "endpoint", Data: "stopping"}
		// logic here
	}
	// Wireguard registered
	if conf.WgRegistered {
		// unregister startram services if exists
		startram.EventBus <- structs.Event{Type: "endpoint", Data: "unregistering"}
		// logic here
	}
	time.Sleep(1 * time.Second) // temp
	// reset pubkey
	startram.EventBus <- structs.Event{Type: "endpoint", Data: "configuring"}
	err := config.CycleWgKey()
	if err != nil {
		handleError(fmt.Sprintf("%v", err))
		return
	}
	// set endpoint to config and persist
	startram.EventBus <- structs.Event{Type: "endpoint", Data: "finalizing"}
	err = config.UpdateConf(map[string]interface{}{
		"endpointUrl": endpoint,
	})
	if err != nil {
		handleError(fmt.Sprintf("%v", err))
		return
	}

	// Finish
	startram.EventBus <- structs.Event{Type: "endpoint", Data: "complete"}

	// debug
	//time.Sleep(2 * time.Second)
	//handleError("Self inflicted error for debug purposes")

	// Clear
	time.Sleep(3 * time.Second)
	startram.EventBus <- structs.Event{Type: "endpoint", Data: nil}
}

func handleStartramCancel(key string, reset bool) {
	// cancel subscription
	// if reset is true
	// unregister startram services
	// reset wg keys
	handleNotImplement("cancel")
}

// temp
func handleNotImplement(action string) {
	logger.Logger.Error(fmt.Sprintf("temp error: %v not implemented", action))
}
