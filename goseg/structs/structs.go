package structs

import (
	"github.com/docker/docker/api/types/container"
)

// eventbus event payloads
type Event struct {
	Type string
	Data interface{}
}

// urbit transition eventbus payloads
type UrbitTransition struct {
	Patp  string
	Type  string
	Event string
}

// new ship transition eventbus payloads
type NewShipTransition struct {
	Type  string
	Event string
}

type LogsEvent struct {
	Action      bool    // stop or start
	ContainerID string  // name of container
	MuCon       *MuConn // pointer to ws session
}

// for keeping track of container desired/actual state
type ContainerState struct {
	ID             string
	Name           string
	Type           string
	Image          string
	ActualStatus   string // on/off
	DesiredStatus  string
	ActualNetwork  string // bridge/wireguard
	DesiredNetwork string
	CreatedAt      string
	Config         container.Config
	Host           container.HostConfig
}
