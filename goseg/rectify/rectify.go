package rectify

// this package is for watching the event bus and rectifying mismatches
// between the desired and actual state
// also for digesting events from docker into broadcasts

import (
	"fmt"
	"goseg/broadcast"
	"goseg/config"
	"goseg/docker"
	"goseg/startram"
	"goseg/structs"
	"strings"
)

func UrbitTransitionHandler() {
	for {
		event := <-docker.UTransBus
		broadcast.UrbTransMu.Lock()
		switch event.Type {
		case "togglePower":
			if _, exists := broadcast.UrbitTransitions[event.Patp]; !exists {
				broadcast.UrbitTransitions[event.Patp] = structs.UrbitTransitionBroadcast{}
			}
			currentStatus := broadcast.UrbitTransitions[event.Patp]
			currentStatus.TogglePower = event.Event
			broadcast.UrbitTransitions[event.Patp] = currentStatus
			broadcast.UrbTransMu.Unlock()
			config.Logger.Info(fmt.Sprintf("Adding %v transition to \"%v\" for %v", event.Type, event.Event, event.Patp))
			broadcast.BroadcastToClients()
		default:
			config.Logger.Warn(fmt.Sprintf("Urecognized transition: %v", event.Type))
		}
	}
}

func NewShipTransitionHandler() {
	for {
		event := <-docker.NewShipTransBus
		switch event.Type {
		case "error":
			var res map[string]interface{}
			res = map[string]interface{}{
				"NewShip": map[string]interface{}{
					"Transition": map[string]interface{}{
						"Error": event.Event,
					},
				},
			}
			err := broadcast.UpdateBroadcastState(res)
			if err != nil {
				config.Logger.Warn(fmt.Sprintf("Error updating new ship transition 'error': %v", err))
			}
			broadcast.BroadcastToClients()
		case "bootStage":
			// Events
			// starting: setting up docker and config
			// creating: actually create and start the container
			// booting: waiting until +code shows up
			// completed: ready to reset
			// aborted: something went wrong and we ran the cleanup routine
			// <empty>: free for new ship
			var res map[string]interface{}
			res = map[string]interface{}{
				"NewShip": map[string]interface{}{
					"Transition": map[string]interface{}{
						"BootStage": event.Event,
					},
				},
			}
			err := broadcast.UpdateBroadcastState(res)
			if err != nil {
				config.Logger.Warn(fmt.Sprintf("Error updating new ship transition 'bootStage': %v", err))
			}
			broadcast.BroadcastToClients()
		case "patp":
			var res map[string]interface{}
			res = map[string]interface{}{
				"NewShip": map[string]interface{}{
					"Transition": map[string]interface{}{
						"Patp": event.Event,
					},
				},
			}
			err := broadcast.UpdateBroadcastState(res)
			if err != nil {
				config.Logger.Warn(fmt.Sprintf("Error updating new ship transition 'bootStage': %v", err))
			}
			broadcast.BroadcastToClients()
		default:
			config.Logger.Warn(fmt.Sprintf("Urecognized transition: %v", event.Type))
		}
	}
}

func RectifyUrbit() {
	event := <-startram.EventBus
	switch event.Type {
	case "retrieve":
		for patp, _ := range config.UrbitConfAll() {
			modified := false
			startramConfig := config.StartramConfig // a structs.StartramRetrieve
			config.LoadUrbitConfig(patp)
			local := config.UrbitConf(patp) // a structs.UrbitDocker
			for _, remote := range startramConfig.Subdomains {
				// for urbit web
				subd := strings.Split(remote.URL, ".")[0]
				if subd == patp && remote.SvcType == "urbit-web" && remote.Status == "ok" {
					// update alias
					if remote.Alias != "null" && remote.Alias != local.CustomUrbitWeb {
						local.CustomUrbitWeb = remote.Alias
						modified = true
					}
					// update www port
					if remote.Port != local.WgHTTPPort {
						local.WgHTTPPort = remote.Port
						modified = true
					}
					// update remote url
					if remote.URL != local.WgURL {
						local.WgURL = remote.URL
						modified = true
					}
					continue
				}
				// for urbit ames
				nestd := strings.Join(strings.Split(remote.URL, ".")[:2], ".")
				if nestd == "ames."+patp && remote.SvcType == "urbit-ames" && remote.Status == "ok" {
					if remote.Port != local.WgAmesPort {
						local.WgAmesPort = remote.Port
						modified = true
					}
					continue
				}
				// for minio
				if nestd == "s3."+patp && remote.SvcType == "minio" && remote.Status == "ok" {
					if remote.Port != local.WgS3Port {
						local.WgS3Port = remote.Port
						modified = true
					}
					continue
				}
				// for minio console
				consd := strings.Join(strings.Split(remote.URL, ".")[:3], ".")
				if consd == "console.s3."+patp && remote.SvcType == "minio-console" && remote.Status == "ok" {
					if remote.Port != local.WgConsolePort {
						local.WgConsolePort = remote.Port
						modified = true
					}
					continue
				}
			}
			if modified {
				config.UpdateUrbitConfig(map[string]structs.UrbitDocker{patp: local})
			}
		}
	default:
	}
}

/*
func SystemTransitionHandler() {
	for {
		event := <-docker.SysTransBus
		switch event.Type {
		case "swap":
			broadcast.SysTransMu.Lock()
			broadcast.SystemTransitions.Swap = event.Event
			broadcast.SysTransMu.Unlock()
		default:
			config.Logger.Warn(fmt.Sprintf("Urecognized transition: %v", event.Type))
		}
	}
}
*/
