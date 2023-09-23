package routines

import (
	"context"
	"fmt"
	"goseg/broadcast"
	"goseg/config"
	"goseg/docker"
	"goseg/logger"
	"goseg/structs"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

var (
	eventBus = make(chan structs.Event, 100)
)

// subscribe to docker events and feed them into eventbus
func DockerListener() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Error initializing Docker client: %v", err))
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
				logger.Logger.Error(fmt.Sprintf("Docker event error: %v", err))
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
			logger.Logger.Error("Failed to assert Docker event data type")
			continue
		}
		contName := dockerEvent.Actor.Attributes["name"]
		switch dockerEvent.Action {

		case "stop":
			logger.Logger.Info(fmt.Sprintf("Docker: %s stopped", contName))

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
			logger.Logger.Info(fmt.Sprintf("Docker: %s started", contName))

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
						logger.Logger.Error(fmt.Sprintf("%v", err))
					}
					// update profile
					current := broadcast.GetState()
					current.Profile.Startram.Info.Running = true
					broadcast.UpdateBroadcast(current)
				}
				makeBroadcast(contName, dockerEvent.Action)
			}

		case "die":
			logger.Logger.Warn(fmt.Sprintf("Docker: %s died!", contName))
			if containerState, exists := config.GetContainerState()[contName]; exists {
				containerState.ActualStatus = "died"
				// we don't want infinite restart loop
				containerState.DesiredStatus = "died"
				config.UpdateContainerState(contName, containerState)
				makeBroadcast(contName, dockerEvent.Action)
			}

		default:
			logger.Logger.Debug(fmt.Sprintf("%s event: %s", contName, dockerEvent.Action))
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
			logger.Logger.Error(fmt.Sprintf("%v", err))
		}
		// update profile
		current := broadcast.GetState()
		current.Profile.Startram.Info.Running = wgOn
		broadcast.UpdateBroadcast(current)
	}
	broadcast.BroadcastToClients()
}
