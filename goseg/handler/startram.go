package handler

import (
	"encoding/json"
	"fmt"
	"groundseg/broadcast"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/startram"
	"groundseg/structs"
	"strings"
	"time"

	"go.uber.org/zap"
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
	case "services":
		go handleStartramServices()
	case "regions":
		go handleStartramRegions()
	case "register":
		regCode := startramPayload.Payload.Key
		region := startramPayload.Payload.Region
		go handleStartramRegister(regCode, region)
	case "toggle":
		go handleStartramToggle()
	case "restart":
		go handleStartramRestart()
	case "cancel":
		key := startramPayload.Payload.Key
		reset := startramPayload.Payload.Reset
		go handleStartramCancel(key, reset)
	case "endpoint":
		endpoint := startramPayload.Payload.Endpoint
		go handleStartramEndpoint(endpoint)
	case "reminder":
		go handleStartramReminder(startramPayload.Payload.Remind)

	default:
		return fmt.Errorf("Unrecognized startram action: %v", startramPayload.Payload.Action)
	}
	return nil
}

func handleStartramServices() {
	go broadcast.GetStartramServices()
}

func handleStartramRegions() {
	go broadcast.LoadStartramRegions()
}

func handleStartramRestart() {
	zap.L().Info("Restarting StarTram")
	startram.EventBus <- structs.Event{Type: "restart", Data: "startram"}
	conf := config.Conf()
	// only restart if startram is on
	if conf.WgOn {
		// record all remote ships
		wgShips := map[string]bool{}
		piers := conf.Piers
		pierStatus, err := docker.GetShipStatus(piers)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to retrieve ship information: %v", err))
		}
		for pier, status := range pierStatus {
			dockerConfig := config.UrbitConf(pier)
			if dockerConfig.Network == "wireguard" {
				wgShips[pier] = (status == "Up" || strings.HasPrefix(status, "Up "))
			}
		}
		zap.L().Debug(fmt.Sprintf("Containers: %+v", wgShips))
		// restart wireguard container
		if err := docker.RestartContainer("wireguard"); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't restart Wireguard: %v", err))
		}
		// operate on urbit ships
		zap.L().Info("Recreating containers")
		for patp, isRunning := range wgShips {
			if isRunning {
				if err := click.BarExit(patp); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to stop %s with |exit for startram restart: %v", patp, err))
				} else {
					for {
						exited, err := shipExited(patp)
						if err == nil {
							if !exited {
								continue
							}
						}
						break
					}
				}
			}
			// delete container
			if err := docker.DeleteContainer(patp); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to delete %s: %v", patp, err))
			}
			minio := fmt.Sprintf("minio_%s", patp)
			if err := docker.DeleteContainer(minio); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to delete %s: %v", minio, err))
			}
		}
		// delete mc
		if err := docker.DeleteContainer("mc"); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to delete minio client: %v", err))
		}
		// create startram containers
		startram.EventBus <- structs.Event{Type: "restart", Data: "urbits"}
		if err := docker.LoadUrbits(); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to load urbits: %v", err))
		}
		startram.EventBus <- structs.Event{Type: "restart", Data: "minios"}
		if err := docker.LoadMC(); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to load minio client: %v", err))
		}
		if err := docker.LoadMinIOs(); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to load minios: %v", err))
		}
		startram.EventBus <- structs.Event{Type: "restart", Data: "done"}
		time.Sleep(3 * time.Second)
		startram.EventBus <- structs.Event{Type: "restart", Data: ""}
	}
}

func handleStartramToggle() {
	startram.EventBus <- structs.Event{Type: "toggle", Data: "loading"}
	conf := config.Conf()
	if conf.WgOn {
		if err := config.UpdateConf(map[string]interface{}{
			"wgOn": false,
		}); err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		}
		err := docker.StopContainerByName("wireguard")
		if err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		}
		// toggle ships back to local
		for _, patp := range conf.Piers {
			dockerConfig := config.UrbitConf(patp)
			if dockerConfig.Network == "wireguard" {
				payload := structs.WsUrbitPayload{
					Payload: structs.WsUrbitAction{
						Type:   "urbit",
						Action: "toggle-network",
						Patp:   patp,
					},
				}
				jsonData, err := json.Marshal(payload)
				if err != nil {
					zap.L().Error(fmt.Sprintf("Error marshalling JSON for %v:", patp, err))
					continue
				}
				if err := UrbitHandler(jsonData); err != nil {
					zap.L().Error(fmt.Sprintf("Error sending action to UrbitHandler for %v:", patp, err))
				}
			}
		}
	} else {
		if err := config.UpdateConf(map[string]interface{}{
			"wgOn": true,
		}); err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		}
		_, err := docker.StartContainer("wireguard", "wireguard")
		if err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		}
	}
	// delete mc
	if err := docker.DeleteContainer("mc"); err != nil {
		zap.L().Error(fmt.Sprintf("Failed to delete minio client: %v", err))
	}
	// load mc
	if err := docker.LoadMC(); err != nil {
		zap.L().Error(fmt.Sprintf("Failed to load minio client: %v", err))
	}
	for _, patp := range conf.Piers {
		// delete minio
		minio := fmt.Sprintf("minio_%s", patp)
		if err := docker.DeleteContainer(minio); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to delete %s: %v", minio, err))
		}
	}
	// load minio
	if err := docker.LoadMinIOs(); err != nil {
		zap.L().Error(fmt.Sprintf("Failed to load minios: %v", err))
	}
	startram.EventBus <- structs.Event{Type: "toggle", Data: nil}
}

func handleStartramRegister(regCode, region string) {
	// error handling
	handleError := func(errmsg string) {
		msg := fmt.Sprintf("Error: %s", errmsg)
		zap.L().Error(errmsg)
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
		if err := docker.StopContainerByName("wireguard"); err != nil {
			handleError(fmt.Sprintf("%v", err))
			return
		}
	}
	// Wireguard registered
	if conf.WgRegistered {
		// unregister startram services if exists
		startram.EventBus <- structs.Event{Type: "endpoint", Data: "unregistering"}
		for _, p := range conf.Piers {
			if err := startram.SvcDelete(p, "urbit"); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't remove urbit anchor for %v", p))
			}
			if err := startram.SvcDelete("s3."+p, "s3"); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't remove s3 anchor for %v", p))
			}
		}
	}
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
		"endpointUrl":  endpoint,
		"wgRegistered": false,
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

// cancel subscription with reg code
func handleStartramCancel(key string, reset bool) {
	handleError := func(errmsg string) {
		msg := fmt.Sprintf("Error: %s", errmsg)
		zap.L().Error(errmsg)
		startram.EventBus <- structs.Event{Type: "cancelSub", Data: msg}
		time.Sleep(3 * time.Second)
		startram.EventBus <- structs.Event{Type: "cancelSub", Data: nil}
	}
	if reset {
		for _, svc := range config.StartramConfig.Subdomains {
			if err := startram.SvcDelete(svc.URL, svc.SvcType); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't delete service %v: %v", svc.URL, err))
			}
		}
	}
	if err := startram.CancelSub(key); err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't cancel subscription: %v", err))
		return
	}
	if err := config.CycleWgKey(); err != nil {
		handleError(fmt.Sprintf("%v", err))
		return
	}
	return
}

func handleStartramReminder(remind bool) {
	conf := config.Conf()
	for _, patp := range conf.Piers {
		update := make(map[string]structs.UrbitDocker)
		shipConf := config.UrbitConf(patp)
		shipConf.StartramReminder = remind
		update[patp] = shipConf
		if err := config.UpdateUrbitConfig(update); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't update urbit config: %v", err))
		}
	}
}

// temp
func handleNotImplement(action string) {
	zap.L().Error(fmt.Sprintf("temp error: %v not implemented", action))
}

func shipExited(patp string) (bool, error) {
	for {
		statuses, err := docker.GetShipStatus([]string{patp})
		if err != nil {
			return false, fmt.Errorf("Failed to get statuses for %s: %v", patp, err)
		}
		status, exists := statuses[patp]
		if !exists {
			return false, fmt.Errorf("%s status doesn't exist", patp)
		}
		if strings.Contains(status, "Up") {
			continue
		}
		return true, nil
	}
}
