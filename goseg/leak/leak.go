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

var (
	LeakChan     = make(chan structs.AuthBroadcast)
	BytesChan    = make(map[string]chan string)
	lickStatuses = make(map[string]LickStatus)
	lickMu       sync.RWMutex
)

func GetLickStatuses() map[string]LickStatus {
	lickMu.RLock()
	defer lickMu.RUnlock()
	return lickStatuses
}

func StartLeak() {
	go LookForPorts()
	oldBroadcast := structs.AuthBroadcast{}
	var err error

	for {
		var newBroadcast structs.AuthBroadcast
		newBroadcast = <-LeakChan
		newBroadcast.Type = "structure"
		newBroadcast.AuthLevel = "authorized"
		// result of broadcastUpdate becomes the new-oldBroadcast
		oldBroadcast, err = updateBroadcast(oldBroadcast, newBroadcast)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to update leak broadcast: %v", err))
		}
	}
}

func LookForPorts() bool {
	// start dev
	if devSocketPath, exists := os.LookupEnv("SHIP"); exists {
		go connectDevSocket(devSocketPath)
	}
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
			go listener(patp, conn, info)
		}
		time.Sleep(1 * time.Second) // interval
	}
}

func connectDevSocket(socketPath string) {
	var info LickStatus
	info.Auth = true
	info.Symlink = socketPath
	conn := makeConnection(filepath.Join(info.Symlink, "groundseg.sock"))
	if conn == nil {
		return
	}
	go listener("dev", conn, info)
}
