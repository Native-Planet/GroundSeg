package handler

import (
	"encoding/json"
	"fmt"
	"goseg/broadcast"
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
	//handleError := func(errmsg string) {
	//	msg := fmt.Sprintf("Error: %s", errmsg)
	//	startram.EventBus <- structs.Event{Type: "register", Data: msg}
	//}
	// Register key
	startram.EventBus <- structs.Event{Type: "register", Data: "key"}
	time.Sleep(2 * time.Second) // temp
	//if err := startram.Register(regCode, region); err != nil {
	//	handleError(fmt.Sprintf("Failed registration: %v", err))
	//}
	// Register Services
	startram.EventBus <- structs.Event{Type: "register", Data: "services"}
	time.Sleep(2 * time.Second) // temp
	//if err := startram.RegisterExistingShips(); err != nil {
	//  handleError(fmt.Sprintf("Unable to register ships: %v", err))
	//}
	// Start Wireguard
	startram.EventBus <- structs.Event{Type: "register", Data: "starting"}
	time.Sleep(2 * time.Second) // temp
	//if err := broadcast.BroadcastToClients(); err != nil {
	//	logger.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
	//}
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
	// stop wireguard
	// unregister startram services
	// set endpoint
	// reset pubkey
	handleNotImplement("endpoint")
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
