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
)

var (
	LeakChan = make(chan []byte)
	patpChan = make(map[string]chan structs.AuthBroadcast)
	ports    = make(map[string]PortStatus)
	portsMu  sync.RWMutex
)

func StartLeak() {
	//go handleGallseg()
	oldBroadcast := structs.AuthBroadcast{}
	for {
		var newBroadcast structs.AuthBroadcast
		broadcastBytes := <-LeakChan
		err := json.Unmarshal(broadcastBytes, &newBroadcast)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to unmarshal broadcastBytes from leakChan"))
			continue
		}
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
	conf := config.Conf()
	for _, patp := range conf.Piers {
		// get ship info
		urbit, exists := newBroadcast.Urbits[patp]
		// urbit info doesn't exist
		// gallseg not installed
		// ipc not connected
		if !exists || !urbit.Info.Gallseg || !ipcIsConnected(patp) {
			continue
		}
		patpChan[patp] <- newBroadcast
	}
	return newBroadcast, nil
}

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
			dockerDir := config.DockerDir
			sock := filepath.Join(dockerDir, patp, "_data", patp, ".urb", "dev", "groundseg", "groundseg")
			_, err := os.Stat(sock)
			// Check if error is due to the file not existing
			if err != nil {
				logger.Logger.Debug("port doesn't exist")
				return false
			}
			logger.Logger.Debug(fmt.Sprintf("port for %v exists", patp))
			/*
				if connected := connect(patp,sock) {
				//port.Location = dir
				//port.Connected = true
				}
			*/
			// ports[patp] = port
		}
	}
	return false
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

/*
func handleIPC(patp, loc string) {
	conn, err := connectToIPC(loc)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("%s: err: %v", loc, err))
		return
	}
	nounContents := <-LeakChan
	nounType := noun.Cell{
		Head: noun.MakeNoun("broadcast"),
		Tail: noun.MakeNoun(nounContents),
	}
	n := noun.MakeNoun(nounType)
	jBytes := toBytes(noun.Jam(n))
	_, err = conn.Write(jBytes)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("%s: err: %v", loc, err))
	}
}

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
