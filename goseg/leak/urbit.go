package leak

import (
	"fmt"
	"groundseg/logger"
	"log"
	"net"
	"time"

	"github.com/stevelacy/go-urbit/noun"
)

type LickSettings struct {
	Available bool
	Auth      bool
}

var (
	lickStatus = make(map[string]LickSettings)
)

func FindLickPorts() {
	// temp
	for {
		n := noun.MakeNoun("string")
		logger.Logger.Warn(fmt.Sprintf("%+v", n))
		time.Sleep(10 * time.Second)
	}

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
