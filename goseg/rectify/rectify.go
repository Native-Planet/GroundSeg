package rectify

// this package is for watching the event bus and rectifying mismatches
// between the desired and actual state

import (
	"fmt"
	"goseg/broadcast"
	"goseg/config"
	"goseg/docker"

	"github.com/docker/docker/api/types/events"
)

// receives events via docker.EventBus
// compares actual state to desired state
func DockerSubscriptionHandler() {
	for {
		event := <-docker.EventBus
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
				broadcast.BroadcastToClients()
			}

		case "start":
			config.Logger.Info(fmt.Sprintf("Docker: %s started", contName))

			if containerState, exists := config.GetContainerState()[contName]; exists {
				containerState.ActualStatus = "running"
				config.UpdateContainerState(contName, containerState)
				broadcast.BroadcastToClients()
			}

		case "die":
			config.Logger.Warn(fmt.Sprintf("Docker: %s died!", contName))
			if containerState, exists := config.GetContainerState()[contName]; exists {
				containerState.ActualStatus = "died"
				// we don't want infinite restart loop
				containerState.DesiredStatus = "died"
				config.UpdateContainerState(contName, containerState)
				broadcast.BroadcastToClients()
			}

		default:
			if config.DebugMode == true {
				config.Logger.Info(fmt.Sprintf("%s event: %s", contName, dockerEvent.Action))
			}
		}
	}
}
