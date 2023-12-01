package leak

import (
	"encoding/json"
	"fmt"
	"goseg/config"
	"goseg/logger"
	"goseg/structs"
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

func StartLeak() {
	//go handleGallseg()
	oldBroadcast := structs.AuthBroadcast{}
	var err error
	for {
		var newBroadcast structs.AuthBroadcast
		newBroadcast = <-LeakChan
		// result of broadcastUpdate becomes the new-oldBroadcast
		oldBroadcast, err = updateBroadcast(oldBroadcast, newBroadcast)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to update leak broadcast: %v", err))
		}
	}
	/*
		fakeLoc := "/home/nal/NP/fakezods/"
		portsCopy := deepCopy()
		for {
			for patp, status := range portsCopy {
				loc := fakeLoc + patp + "/.urb/dev/leak/leak.sock"
				logger.Logger.Warn(fmt.Sprintf("%s: status: %+v", loc, status))
				go handleIPC(patp, loc)
			}
			time.Sleep(10 * time.Second)
		}
	*/
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
		handleIPC(patp, ipcLocation, string(newBroadcastBytes))
		/*
			patpChan[patp] <- newBroadcast
		*/
	}
	return newBroadcast, nil
}

/*
	func ipcIsConnected(patp string) bool {
		portsMu.Lock()
		defer portsMu.Unlock()
		var port PortStatus
		port, exists := ports[patp]
		if !exists {
			port.Connected = false
		}
		if !port.Connected {
			if port.Location == "" {
*/
func checkIPCExists(patp string) (string, bool) {
	dockerDir := config.DockerDir
	sockLocation := filepath.Join(dockerDir, patp, "_data", patp, ".urb", "dev", "groundseg")
	sock := filepath.Join(sockLocation, "groundseg.sock")
	_, err := os.Stat(sock)
	// Check if error is due to the file not existing
	if err != nil {
		return "", false
	}
	return sockLocation, true
}

/*
// connects to IPC port
// func connect(patp, socketPath string) (net.Conn) {
func connect(patp, socketPath string) bool {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		//logger.Logger.Error(fmt.Sprintf("
		return nil
	}
	return false
	//return conn
}
*/

/*
func handleGallseg() {
	for {
		patp := <-patpChan
		// check if IPC is connected from map
		// if no, spin up a routine with oldBroadcast, newBroadcast and patp
		// in the routine:
		// try connecting to IPC
		// if succeed
		// create a channel
		// check if admin
		// if admin send full
		// if ship, deepEqual, if same, continue
		// else, send up only the ship stuff
	}
}
*/

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

func handleIPC(patp, original, broadcast string) {
	shortPath := "/np/d/gs"
	symlink := filepath.Join(shortPath, patp)
	createSymlink(shortPath, symlink, original, patp)
	conn, err := connectToIPC(filepath.Join(symlink, "groundseg.sock"))
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("err: %v", err))
		return
	}
	//nounContents := <-LeakChan
	nounType := noun.Cell{
		Head: noun.MakeNoun("broadcast"),
		Tail: noun.MakeNoun(broadcast),
	}
	n := noun.MakeNoun(nounType)
	jBytes := toBytes(noun.Jam(n))
	_, err = conn.Write(jBytes)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("%s: err: %v", symlink, err))
	}
}

/*
func deepCopy() map[string]*PortStatus {
	statusMu.Lock()
	defer statusMu.Unlock()
	newMap := make(map[string]*PortStatus)
	for key, value := range ports {
		newMap[key] = &PortStatus{
			Connected: value.Connected,
		}
	}
	return newMap
}

func int64ToLittleEndianBytes(num int64) ([]byte, error) {
	buf := new(bytes.Buffer)
	// uint32 for 4 bytes
	err := binary.Write(buf, binary.LittleEndian, uint32(num))
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func toBytes(num *big.Int) []byte {
	var padded []byte
	// version: 0
	version := []byte{0}

	// length: 4 bytes
	length := noun.ByteLen(num)
	lenBytes, err := int64ToLittleEndianBytes(length)
	if err != nil {
		fmt.Println(fmt.Sprintf("%v", err))
	}
	fmt.Println(fmt.Sprintf("%v", lenBytes))

	bytes := makeBytes(num)
	padded = append(padded, version...)
	padded = append(padded, lenBytes...)
	padded = append(padded, bytes...)

	return padded
}

func makeBytes(num *big.Int) []byte {
	byteSlice := num.Bytes()
	// Reverse the slice for little-endian
	for i, j := 0, len(byteSlice)-1; i < j; i, j = i+1, j-1 {
		byteSlice[i], byteSlice[j] = byteSlice[j], byteSlice[i]
	}
	return byteSlice
}
*/
