package leak

import (
	"encoding/json"
	"fmt"
	"goseg/config"
	"goseg/logger"
	"goseg/structs"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/stevelacy/go-urbit/noun"
)

// needs to stop disconnecting at every broaddcast
// check for auth
// urbit checker for ui
// install app from ui
// frequency slider
// shut

var (
	LeakChan = make(chan structs.AuthBroadcast)
	patpChan = make(map[string]chan structs.AuthBroadcast)
	ports    = make(map[string]PortStatus)
	portsMu  sync.RWMutex
)

func HasOpenPorts() bool {
	return true
}

func StartLeak() {
	//go handleGallseg()
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

func updateBroadcast(oldBroadcast, newBroadcast structs.AuthBroadcast) (structs.AuthBroadcast, error) {
	// deep equality check
	if reflect.DeepEqual(oldBroadcast, newBroadcast) {
		return oldBroadcast, nil
	}
	newBroadcastBytes, err := json.Marshal(newBroadcast)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to marshal broadcast for lick: %v", err))
		return oldBroadcast, nil
	}
	// dev broadcast
	devLocation, devExists := checkIPCExists("dev")
	if devExists {
		conn, err := handleIPC("dev", devLocation)
		if err != nil {
			logger.Logger.Debug(fmt.Sprintf("Dev socket %v failed: %v", devLocation, err))
		}
		sendBroadcast(conn, "dev", string(newBroadcastBytes))
	}
	conf := config.Conf()
	for _, patp := range conf.Piers {
		// get ship info
		urbit, exists := newBroadcast.Urbits[patp]
		// urbit info doesn't exist
		// gallseg not installed
		// ipc not connected
		ipcLocation, ipcExists := checkIPCExists(patp)
		if !exists || !urbit.Info.Gallseg || !ipcExists {
			continue
		}
		conn, err := handleIPC(patp, ipcLocation)
		if err != nil {
		}
		sendBroadcast(conn, patp, string(newBroadcastBytes))
	}
	return newBroadcast, nil
}

func checkIPCExists(patp string) (string, bool) {
	dockerDir := config.DockerDir
	sockLocation := filepath.Join(dockerDir, patp, "_data", patp, ".urb", "dev", "groundseg")
	if patp == "dev" {
		value, exists := os.LookupEnv("SHIP")
		if exists {
			sockLocation = value
		}
	}
	sock := filepath.Join(sockLocation, "groundseg.sock")
	_, err := os.Stat(sock)
	// Check if error is due to the file not existing
	if err != nil {
		return "", false
	}
	return sockLocation, true
}

func createSymlink(shortPath, symlink, original, patp string) {
	err := os.MkdirAll(shortPath, 0755)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to create directory: %v : %v", shortPath, err))
	}
	// Check if the symlink already exists
	info, err := os.Lstat(symlink)
	if err == nil {
		// If it's a symlink
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(symlink)
			if err != nil {
				logger.Logger.Debug(fmt.Sprintf("Error reading the symlink: %v", err))
				return
			}

			// Check if it points to the desired target
			if target == original {
				//logger.Logger.Error(fmt.Sprintf("Symlink already exists."))
				return
			}
		}

		// Remove if it's a different symlink or a file
		err = os.Remove(symlink)
		if err != nil {
			logger.Logger.Debug(fmt.Sprintf("Error removing existing file or symlink: %v", err))
			return
		}
	} else if !os.IsNotExist(err) {
		logger.Logger.Debug(fmt.Sprintf("Error checking symlink: %v", err))
		return
	}

	// Create the symlink
	err = os.Symlink(original, symlink)
	if err != nil {
		logger.Logger.Debug(fmt.Sprintf("Failed to symlink %v to %v: %v", original, symlink, err))
	} else {
		logger.Logger.Info(fmt.Sprintf("Gallseg Symlink created successfully for %v", patp))
	}
}

func handleIPC(patp, original string) (net.Conn, error) {
	shortPath := "/np/d/gs"
	symlink := filepath.Join(shortPath, patp)
	createSymlink(shortPath, symlink, original, patp)
	conn, err := connectToIPC(filepath.Join(symlink, "groundseg.sock"))
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("err: %v", err))
		return conn, err
	}
	return conn, nil
}

func sendBroadcast(conn net.Conn, patp, broadcast string) {
	//nounContents := <-LeakChan
	nounType := noun.Cell{
		Head: noun.MakeNoun("broadcast"),
		Tail: noun.MakeNoun(broadcast),
	}
	n := noun.MakeNoun(nounType)
	jBytes := toBytes(noun.Jam(n))
	if conn != nil {
		_, err := conn.Write(jBytes)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Send broadcast to %s: err: %v", patp, err))
		}
	}
}
