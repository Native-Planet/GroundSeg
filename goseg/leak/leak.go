package leak

import (
	"fmt"
	"goseg/config"
	"goseg/logger"
	"goseg/structs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// needs to stop disconnecting at every broaddcast
// check for auth
// urbit checker for ui
// install app from ui
// frequency slider
// shut

var (
	LeakChan     = make(chan structs.AuthBroadcast)
	lickStatuses = make(map[string]LickStatus)
	lickMu       sync.RWMutex
)

func GetLickStatuses() map[string]LickStatus {
	lickMu.RLock()
	defer lickMu.RUnlock()
	return lickStatuses
}

func SetLickStatus(patp string, newStatus LickStatus) error {
	lickMu.Lock()
	defer lickMu.Unlock()
	lickStatuses[patp] = newStatus
	return nil
}

func LookForPorts() bool {
	// symlink path
	symlinkPath := "/np/d/gs"
	// socket file name
	sockFile := "groundseg.sock"
	// always checking
	for {
		conf := config.Conf()
		dockerDir := config.DockerDir
		// check for every ship that exists in groundseg
		for _, patp := range conf.Piers {
			// decide based on existence of info
			statuses := GetLickStatuses()
			info, exists := statuses[patp]
			if exists {
				continue
			}
			// check if sock exists
			sockLocation := filepath.Join(dockerDir, patp, "_data", patp, ".urb", "dev", "groundseg")
			sock := filepath.Join(sockLocation, sockFile)
			_, err := os.Stat(sock)
			if err != nil {
				// socket doesn't exist
				continue
			}
			// socket exists, shorten path with symlink
			info.Symlink, err = makeSymlink(patp, sockLocation, symlinkPath)
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("%v", err))
				continue
			}
			// Auth is false by default
			info.Auth = false

			// attempt connection to socket
			conn := makeConnection(filepath.Join(info.Symlink, sockFile))
			if conn == nil {
				continue
			}
			go listener(patp, conn)
			/*
				info.Conn = conn
				if err := SetLickStatus(patp, info); err != nil {
					continue
				}
			*/
		}
		time.Sleep(1 * time.Second) // interval
	}
}

func StartLeak() {
	go LookForPorts()
	oldBroadcast := structs.AuthBroadcast{}
	var err error

	for {
		var newBroadcast structs.AuthBroadcast
		newBroadcast = <-LeakChan
		for patp, status := range GetLickStatuses() {
			logger.Logger.Warn(fmt.Sprintf("patp: %v, status: %+v", patp, status))
		}
		newBroadcast.Type = "structure"
		newBroadcast.AuthLevel = "authorized"
		// result of broadcastUpdate becomes the new-oldBroadcast
		oldBroadcast, err = updateBroadcast(oldBroadcast, newBroadcast)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to update leak broadcast: %v", err))
		}
	}
}
