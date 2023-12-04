package leak

import (
	"encoding/json"
	"fmt"
	"goseg/logger"
	"goseg/structs"
	"net"
	"reflect"

	"github.com/stevelacy/go-urbit/noun"
)

func listener(patp string, conn net.Conn, info LickStatus) {
	BytesChan[patp] = make(chan string)
	// put in map
	lickMu.Lock()
	lickStatuses[patp] = info
	lickMu.Unlock()
	// defer remove from map
	defer func() {
		lickMu.Lock()
		delete(lickStatuses, patp)
		lickMu.Unlock()
		delete(BytesChan, patp)
		conn.Close()
	}()
	for {
		// listen and send until end
		broadcast := <-BytesChan[patp]
		c, err := sendBroadcast(conn, broadcast)
		if err != nil {
			return
		}
		conn = c
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
	for patp, _ := range GetLickStatuses() {
		// TODO: check auth here
		BytesChan[patp] <- string(newBroadcastBytes)
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
