package routines

import (
	"context"
	"fmt"
	"groundseg/broadcast"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/structs"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

var (
	eventBus           = make(chan structs.Event, 100)
	DisableShipRestart = false
)

func init() {
	go Check502Loop()
}

// subscribe to docker events and feed them into eventbus
func DockerListener() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Error initializing Docker client: %v", err))
		return
	}
	messages, errs := cli.Events(ctx, types.EventsOptions{})
	for {
		select {
		case event := <-messages:
			// Convert the Docker event to our custom event and send it to the EventBus
			eventBus <- structs.Event{Type: event.Action, Data: event}
		case err := <-errs:
			if err != nil {
				zap.L().Error(fmt.Sprintf("Docker event error: %v", err))
			}
		}
	}
}

// receives events via docker.EventBus
// compares actual state to desired state
func DockerSubscriptionHandler() {
	for {
		event := <-eventBus
		dockerEvent, ok := event.Data.(events.Message)
		if !ok {
			zap.L().Error("Failed to assert Docker event data type")
			continue
		}
		contName := dockerEvent.Actor.Attributes["name"]
		switch dockerEvent.Action {

		case "stop":
			zap.L().Info(fmt.Sprintf("Docker: %s stopped", contName))

			if containerState, exists := config.GetContainerState()[contName]; exists {
				containerState.ActualStatus = "stopped"
				config.UpdateContainerState(contName, containerState)
				// start it again if this isn't what the user wants
				if containerState.DesiredStatus != "stopped" {
					docker.StartContainer(contName, containerState.Type)
				}
				makeBroadcast(contName, dockerEvent.Action)
			}

		case "start":
			zap.L().Info(fmt.Sprintf("Docker: %s started", contName))

			containerState, exists := config.GetContainerState()[contName]
			if exists {
				containerState.ActualStatus = "running"
				config.UpdateContainerState(contName, containerState)
				switch contName {
				case "wireguard":
					// set wgOn to false
					err := config.UpdateConf(map[string]interface{}{
						"wgOn": true,
					})
					if err != nil {
						zap.L().Error(fmt.Sprintf("%v", err))
					}
					// update profile
					current := broadcast.GetState()
					current.Profile.Startram.Info.Running = true
					broadcast.UpdateBroadcast(current)
				}
				makeBroadcast(contName, dockerEvent.Action)
			}

		case "die":
			zap.L().Warn(fmt.Sprintf("Docker: %s died!", contName))
			if containerState, exists := config.GetContainerState()[contName]; exists {
				containerState.ActualStatus = "died"
				// we don't want infinite restart loop
				containerState.DesiredStatus = "died"
				if containerState.Type == "vere" {
					click.ClearLusCode(contName)
				}
				config.UpdateContainerState(contName, containerState)
				makeBroadcast(contName, dockerEvent.Action)
			}

		default:
			zap.L().Debug(fmt.Sprintf("%s event: %s", contName, dockerEvent.Action))
		}
	}
}

func makeBroadcast(contName string, status string) {
	switch contName {
	case "wireguard":
		var wgOn bool
		// set wgOn to false
		if status == "die" {
			wgOn = false
		}
		// set wgOn to true
		if status == "start" {
			wgOn = true
		}
		if err := config.UpdateConf(map[string]interface{}{"wgOn": wgOn}); err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		}
		// update profile
		current := broadcast.GetState()
		current.Profile.Startram.Info.Running = wgOn
		broadcast.UpdateBroadcast(current)
	}
	broadcast.BroadcastToClients()
}

// loop to make sure ships are reachable
// if 502 2x in 2 min, restart wg container
func Check502Loop() {
	status := make(map[string]bool)
	time.Sleep(180 * time.Second)
	for {
		time.Sleep(120 * time.Second)
		conf := config.Conf()
		pierStatus, err := docker.GetShipStatus(conf.Piers)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't get pier status: %v", err))
			continue
		}
		for _, pier := range conf.Piers {
			err := config.LoadUrbitConfig(pier)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Error loading %s config: %v", pier, err))
				continue
			}
			shipConf := config.UrbitConf(pier)
			pierNetwork, err := docker.GetContainerNetwork(pier)
			if err != nil {
				zap.L().Warn(fmt.Sprintf("Couldn't get network for %v: %v", pier, err))
				continue
			}
			turnedOn := false
			if strings.Contains(pierStatus[pier], "Up") {
				turnedOn = true
			}
			if turnedOn && pierNetwork != "default" && conf.WgOn {
				if _, err := click.GetLusCode(pier); err != nil {
					zap.L().Warn(fmt.Sprintf("%v is not booted yet, skipping", pier))
					continue
				}
				resp, err := http.Get("https://" + shipConf.WgURL)
				if err != nil {
					zap.L().Error(fmt.Sprintf("Error remote polling %v: %v", pier, err))
					continue
				}
				resp.Body.Close()
				zap.L().Debug(fmt.Sprintf("%v 502 check: %v", pier, resp.StatusCode))
				if resp.StatusCode == http.StatusBadGateway {
					zap.L().Warn(fmt.Sprintf("Got 502 response for %v", pier))
					if _, found := status[pier]; found {
						// found = 2x in a row
						zap.L().Warn(fmt.Sprintf("502 second strike for %v", pier))
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

						// restart wireguard container
						if err := docker.RestartContainer("wireguard"); err != nil {
							zap.L().Error(fmt.Sprintf("Couldn't restart Wireguard: %v", err))
						}
						// operate on urbit ships
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
								zap.L().Error(fmt.Sprintf("Failed to delete %s: %v", patp, err))
							}
						}
						// create startram containers
						if err := docker.LoadUrbits(); err != nil {
							zap.L().Error(fmt.Sprintf("Failed to load urbits: %v", err))
						}
						if err := docker.LoadMC(); err != nil {
							zap.L().Error(fmt.Sprintf("Failed to load minio client: %v", err))
						}
						if err := docker.LoadMinIOs(); err != nil {
							zap.L().Error(fmt.Sprintf("Failed to load minios: %v", err))
						}
						// remove from map after restart
						delete(status, pier)
					} else {
						// first 502
						status[pier] = true
					}
				} else if _, found := status[pier]; found {
					// if not 502 and pier is in status map, remove it
					delete(status, pier)
				}
			}
		}
	}
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

func GracefulShipExit() error {
	DisableShipRestart = true
	defer func() {
		DisableShipRestart = false
	}()
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
	conf := config.Conf()
	piers := conf.Piers
	pierStatus, err := docker.GetShipStatus(piers)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to retrieve ship information: %v", err))
	}
	for patp, status := range pierStatus {
		if status == "Up" || strings.HasPrefix(status, "Up ") {
			if err := click.BarExit(patp); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to stop %s with |exit for daemon restart: %v", patp, err))
				continue
			}
			for {
				status, err := getShipRunningStatus(patp)
				if err != nil {
					break
				}
				zap.L().Debug(fmt.Sprintf("%s", status))
				if !strings.Contains(status, "Up") {
					break
				}
				time.Sleep(1 * time.Second)
			}
		}
	}
	return nil
}
