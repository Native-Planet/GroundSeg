package routines

import (
	"context"
	"errors"
	"fmt"
	"groundseg/dockerclient"
	"groundseg/structs"
	"groundseg/transition"
	"net/http"
	"time"

	eventtypes "github.com/docker/docker/api/types/events"
	"go.uber.org/zap"
)

var (
	eventBus           = make(chan structs.Event, 100)
	DisableShipRestart = false
)

func StartDockerHealthLoops() {
	go Check502Loop()
}

// subscribe to docker events and feed them into eventbus
func DockerListener() {
	DockerListenerWithContext(context.Background())
}

// DockerListenerWithContext runs until ctx is canceled.
func DockerListenerWithContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	cli, err := dockerclient.New()
	if err != nil {
		zap.L().Error(fmt.Sprintf("Error initializing Docker client: %v", err))
		return
	}
	messages, errs := cli.Events(ctx, eventtypes.ListOptions{})
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Stopping Docker event listener")
			return
		case event := <-messages:
			if ctx.Err() != nil {
				return
			}
			// Convert the Docker event to our custom event and send it to the EventBus
			select {
			case eventBus <- structs.Event{Type: string(event.Action), Data: event}:
			case <-ctx.Done():
				return
			}
		case err := <-errs:
			if err == nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			zap.L().Error(fmt.Sprintf("Docker event error: %v", err))
			if ctx.Err() != nil {
				return
			}
		}
	}
}

// receives events via docker.EventBus
// compares actual state to desired state
func DockerSubscriptionHandler() {
	DockerSubscriptionHandlerWithContext(context.Background())
}

func DockerSubscriptionHandlerWithContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	dockerSubscriptionHandlerWithRuntime(ctx, newDockerRoutineRuntime())
}

func dockerSubscriptionHandlerWithRuntime(ctx context.Context, rt dockerRoutineRuntime) {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Stopping Docker subscription handler")
			return
		case event := <-eventBus:
			dockerEvent, ok := event.Data.(eventtypes.Message)
			if !ok {
				zap.L().Error("Failed to assert Docker event data type")
				continue
			}
			contName := dockerEvent.Actor.Attributes["name"]
			action := transition.DockerAction(dockerEvent.Action)
			switch action {

			case transition.DockerActionStop:
				zap.L().Info(fmt.Sprintf("Docker: %s stopped", contName))

				updateContainerTransition(rt, contName, action, func(state *structs.ContainerState) error {
					state.ActualStatus = string(transition.ContainerStatusStopped)
					if state.DesiredStatus != string(transition.ContainerStatusStopped) {
						if _, err := rt.startContainer(contName, state.Type); err != nil {
							return err
						}
					}
					return nil
				})

			case transition.DockerActionStart:
				zap.L().Info(fmt.Sprintf("Docker: %s started", contName))

				updateContainerTransition(rt, contName, action, func(state *structs.ContainerState) error {
					state.ActualStatus = string(transition.ContainerStatusRunning)
					if contName == string(transition.ContainerTypeWireguard) {
						if err := rt.updateWgOn(true); err != nil {
							return err
						}
						current := rt.getState()
						current.Profile.Startram.Info.Running = true
						rt.updateBroadcast(current)
					}
					return nil
				})

			case transition.DockerActionDie:
				zap.L().Warn(fmt.Sprintf("Docker: %s died!", contName))
				updateContainerTransition(rt, contName, action, func(state *structs.ContainerState) error {
					state.ActualStatus = string(transition.ContainerStatusDied)
					if state.Type != string(transition.ContainerTypeVere) {
						return nil
					}
					if err := rt.loadUrbitConfig(contName); err != nil {
						return fmt.Errorf("Failed to load config for %s: %w", contName, err)
					}
					conf := rt.urbitConf(contName)
					if conf.DisableShipRestarts {
						zap.L().Info(fmt.Sprintf("Leaving %s container alone after death due to DisableShipRestarts=true", contName))
						state.DesiredStatus = string(transition.ContainerStatusStopped)
						rt.clearLusCode(contName)
						return nil
					}
					rt.clearLusCode(contName)
					if state.DesiredStatus != string(transition.ContainerStatusDied) && state.DesiredStatus != string(transition.ContainerStatusStopped) {
						zap.L().Info(fmt.Sprintf("Attempting to restart ship %s after death", contName))
						go func(name, containerType string) {
							rt.sleep(2 * time.Second)
							_, err := rt.startContainer(name, containerType)
							if err != nil {
								zap.L().Error(fmt.Sprintf("Failed to restart %s after death: %v", name, err))
							} else {
								zap.L().Info(fmt.Sprintf("Successfully restarted %s after death", name))
							}
						}(contName, state.Type)
					} else {
						zap.L().Info(fmt.Sprintf("Ship desired status: %s", state.DesiredStatus))
					}
					return nil
				})
			default:
				zap.L().Debug(fmt.Sprintf("%s event: %s", contName, dockerEvent.Action))
			}
		}
	}
}

func updateContainerTransition(rt dockerRoutineRuntime, contName string, action transition.DockerAction, mutate func(*structs.ContainerState) error) {
	containerState, exists := rt.getContainerState()[contName]
	if !exists {
		return
	}

	if err := mutate(&containerState); err != nil {
		zap.L().Warn(fmt.Sprintf("Docker transition failed for %s: %v", contName, err))
		return
	}
	rt.updateContainerState(contName, containerState)
	makeBroadcastWithRuntime(rt, contName, string(action))
}

func makeBroadcast(contName string, status string) {
	makeBroadcastWithRuntime(newDockerRoutineRuntime(), contName, status)
}

func makeBroadcastWithRuntime(rt dockerRoutineRuntime, contName string, status string) {
	switch contName {
	case "wireguard":
		var wgOn bool
		// set wgOn to false
		if status == string(transition.DockerActionDie) {
			wgOn = false
		}
		// set wgOn to true
		if status == string(transition.DockerActionStart) {
			wgOn = true
		}
		if err := rt.updateWgOn(wgOn); err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		}
		// update profile
		current := rt.getState()
		current.Profile.Startram.Info.Running = wgOn
		rt.updateBroadcast(current)
	}
	rt.broadcastClients()
}

// loop to make sure ships are reachable
// if 502 2x in 2 min, restart wg container
func Check502Loop() {
	check502Loop(newDockerRoutineRuntime())
}

func check502Loop(rt dockerRoutineRuntime) {
	status := make(map[string]bool)
	rt.sleep(180 * time.Second)
	for {
		rt.sleep(120 * time.Second)
		settings := rt.getCheck502Settings()
		pierStatus, err := rt.getShipStatus(settings.Piers)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't get pier status: %v", err))
			continue
		}
		for _, pier := range settings.Piers {
			err := rt.loadUrbitConfig(pier)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Error loading %s config: %v", pier, err))
				continue
			}
			shipConf := rt.urbitConf(pier)
			pierNetwork, err := rt.getContainerNetwork(pier)
			if err != nil {
				zap.L().Warn(fmt.Sprintf("Couldn't get network for %v: %v", pier, err))
				continue
			}
			turnedOn := false
			if transition.IsContainerUpStatus(pierStatus[pier]) {
				turnedOn = true
			}
			if turnedOn && pierNetwork != "default" && settings.WgOn {
				if _, err := rt.getLusCode(pier); err != nil {
					zap.L().Warn(fmt.Sprintf("%v is not booted yet, skipping", pier))
					continue
				}
				resp, err := rt.httpGet("https://" + shipConf.WgURL)
				if err != nil {
					zap.L().Error(fmt.Sprintf("Error remote polling %v: %v", pier, err))
					continue
				}
				resp.Body.Close()
				zap.L().Debug(fmt.Sprintf("%v 502 check: %v", pier, resp.StatusCode))
				if resp.StatusCode == http.StatusBadGateway {
					zap.L().Warn(fmt.Sprintf("Got 502 response for %v", pier))
					if _, found := status[pier]; found && !settings.Disable502 {
						// found = 2x in a row
						zap.L().Warn(fmt.Sprintf("502 second strike for %v", pier))
						if err := rt.recoverWireguard(settings.Piers, false); err != nil {
							zap.L().Error(fmt.Sprintf("Wireguard fleet recovery failed: %v", err))
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

func GracefulShipExit() error {
	return gracefulShipExit(newDockerRoutineRuntime())
}

func gracefulShipExit(rt dockerRoutineRuntime) error {
	DisableShipRestart = true
	defer func() {
		DisableShipRestart = false
	}()
	getShipRunningStatus := func(patp string) (string, error) {
		statuses, err := rt.getShipStatus([]string{patp})
		if err != nil {
			return "", fmt.Errorf("Failed to get statuses for %s: %w", patp, err)
		}
		status, exists := statuses[patp]
		if !exists {
			return "", fmt.Errorf("%s status doesn't exist", patp)
		}
		return status, nil
	}
	piers := rt.getShipSettings().Piers
	pierStatus, err := rt.getShipStatus(piers)
	if err != nil {
		return fmt.Errorf("Failed to retrieve ship information: %w", err)
	}
	var stepErrors []error
	for patp, status := range pierStatus {
		if transition.IsContainerUpStatus(status) {
			if err := rt.barExit(patp); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to stop %s with |exit for daemon restart: %v", patp, err))
				stepErrors = append(stepErrors, fmt.Errorf("failed to stop %s with |exit for daemon restart: %w", patp, err))
				continue
			}
			for {
				status, err := getShipRunningStatus(patp)
				if err != nil {
					stepErrors = append(stepErrors, fmt.Errorf("failed to poll %s status during graceful exit: %w", patp, err))
					break
				}
				zap.L().Debug(fmt.Sprintf("%s", status))
				if !transition.IsContainerUpStatus(status) {
					break
				}
				rt.sleep(1 * time.Second)
			}
		}
	}
	if joined := errors.Join(stepErrors...); joined != nil {
		return fmt.Errorf("one or more ships failed graceful shutdown: %w", joined)
	}
	return nil
}
