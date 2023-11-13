package urbit

import (
	"log"
	"net"
)

type LickSettings struct {
	Available bool
	Auth      bool
}

var (
	lickStatus = make(map[string]LickSettings)
)

func FindLickPorts() {
	// loop through conf.Piers
	// add to Lick settings if port is available
	// else remove from Lick settings
}

func ConnectToLick() {
	c, err := net.Dial("unix", "/some/dir")
	if err != nil {
		log.Fatal("Dial error", err)
	}
	defer c.Close()
	// Use c to send and receive data
}

func BroadcastToUrbit() {
	// loop through lickStatus
	// if available
	// if authed send full broadcast
	// if !authed send only ship info
	// todo: add flood control
}
