package routines

import (
	"context"
	"fmt"
	"goseg/config"
	"goseg/docker"
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
		config.Logger.Error(fmt.Sprintf("Error initializing Docker client: %v", err))
		return
	}
	messages, errs := cli.Events(ctx, types.EventsOptions{})
	for {
		select {
		case event := <-messages:
			// Convert the Docker event to our custom event and send it to the EventBus
			eventBus <- structs.Event{Type: event.Action, Data: event}
		case err := <-errs:
			config.Logger.Error(fmt.Sprintf("Docker event error: %v", err))
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
			config.Logger.Error("Failed to assert Docker event data type")
			continue
		}
		contName := dockerEvent.Actor.Attributes["name"]
		switch dockerEvent.Action {

		case "stop":
			config.Logger.Info(fmt.Sprintf("Docker: %s stopped", contName))

			if containerState, exists := config.GetContainerState()[contName]; exists {
				containerState.ActualStatus = "stopped"
				config.UpdateContainerState(contName, containerState)
				// start it again if this isn't what the user wants
				if containerState.DesiredStatus != "stopped" {
					docker.StartContainer(contName, containerState.Type)
				}
				//broadcast.BroadcastToClients()
			}

		case "start":
			config.Logger.Info(fmt.Sprintf("Docker: %s started", contName))

			containerState, exists := config.GetContainerState()[contName]
			config.Logger.Warn(fmt.Sprintf("%+v", containerState)) // temp
			if exists {
				containerState.ActualStatus = "running"
				config.UpdateContainerState(contName, containerState)
				//broadcast.BroadcastToClients()
			}

		case "die":
			config.Logger.Warn(fmt.Sprintf("Docker: %s died!", contName))
			if containerState, exists := config.GetContainerState()[contName]; exists {
				containerState.ActualStatus = "died"
				// we don't want infinite restart loop
				containerState.DesiredStatus = "died"
				config.UpdateContainerState(contName, containerState)
				//broadcast.BroadcastToClients()
			}

		default:
			if config.DebugMode == true {
				config.Logger.Info(fmt.Sprintf("%s event: %s", contName, dockerEvent.Action))
			}
		}
	}
}
