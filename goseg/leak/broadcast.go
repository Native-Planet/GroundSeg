package leak

import (
	"encoding/json"
	"fmt"
	"groundseg/logger"
	"groundseg/structs"
	"net"
	"reflect"

	"github.com/stevelacy/go-urbit/noun"
)

func listener(patp string, conn net.Conn, info LickStatus) {
	BytesChan[patp] = make(chan string)
	readChan := make(chan []byte)
	lickMu.Lock()
	// put in status map
	lickStatuses[patp] = info
	lickMu.Unlock()
	defer func() {
		logger.Logger.Info(fmt.Sprintf("Closing lick channel for %s", patp))
		lickMu.Lock()
		// remove from status map
		delete(lickStatuses, patp)
		lickMu.Unlock()
		// remove from channel map
		delete(BytesChan, patp)
		// close connection
		if conn != nil {
			conn.Close()
		}
		logger.Logger.Info(fmt.Sprintf("Closed lick channel for %s", patp))
	}()
	// Goroutine to read from connection
	go func() {
		buf := make([]byte, 1024) // buffer size can be adjusted
		for {
			n, err := conn.Read(buf)
			if err != nil {
				// handle error or end of read
				close(readChan)
				return
			}
			readData := make([]byte, n)
			copy(readData, buf[:n])
			readChan <- readData
		}
	}()
	for {
		select {
		// listen to broadcast and send
		case broadcast := <-BytesChan[patp]:
			c, err := sendBroadcast(conn, broadcast)
			if err != nil {
				return
			}
			conn = c
		// listen to lick port
		case action, ok := <-readChan:
			if !ok {
				return
			}
			go handleAction(patp, action)
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
	statuses := GetLickStatuses()
	for patp, status := range statuses {
		if status.Auth {
			go func(patp string) {
				BytesChan[patp] <- string(newBroadcastBytes)
			}(patp)
		} else {
			go func(patp string) {
				shipInfo, exists := newBroadcast.Urbits[patp]
				if exists {
					urbits := make(map[string]structs.Urbit)
					urbits[patp] = shipInfo
					wrappedUrbit := structs.AuthBroadcast{
						Type:      "structure",
						AuthLevel: patp,
						Urbits:    urbits,
					}
					wrappedUrbit.Profile.Startram.Info.Registered = newBroadcast.Profile.Startram.Info.Registered
					wrappedUrbit.Profile.Startram.Info.Running = newBroadcast.Profile.Startram.Info.Running

					wrapperUrbitBytes, err := json.Marshal(wrappedUrbit)
					if err == nil {

						BytesChan[patp] <- string(wrapperUrbitBytes)
					}
				}
			}(patp)
		}
	}
	return newBroadcast, nil
}

func sendBroadcast(conn net.Conn, broadcast string) (net.Conn, error) {
	nounType := noun.Cell{
		Head: noun.MakeNoun("broadcast"),
		Tail: noun.MakeNoun(broadcast),
	}
	n := noun.MakeNoun(nounType)
	jBytes := toBytes(noun.Jam(n))
	if conn != nil {
		_, err := conn.Write(jBytes)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Send broadcast error: %v", err))
			return nil, err
		}
	}
	return conn, nil
}
