package leak

import (
	"fmt"
	"goseg/logger"
	"goseg/structs"
	"net"
	"reflect"
	"time"

	"github.com/stevelacy/go-urbit/noun"
)

func listener(patp string, conn net.Conn) {
	logger.Logger.Warn(fmt.Sprintf("%v", patp)) // temp
	logger.Logger.Warn(fmt.Sprintf("%v", conn)) // temp
	// put in map
	// defer remove from map
	for {
		// listen and send until end
		time.Sleep(1 * time.Hour) // temp
	}
}

func updateBroadcast(oldBroadcast, newBroadcast structs.AuthBroadcast) (structs.AuthBroadcast, error) {
	// deep equality check
	if reflect.DeepEqual(oldBroadcast, newBroadcast) {
		return oldBroadcast, nil
	}
	//newBroadcastBytes, err := json.Marshal(newBroadcast)
	/*
		_, err := json.Marshal(newBroadcast)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to marshal broadcast for lick: %v", err))
			return oldBroadcast, nil
		}
	*/
	return newBroadcast, nil
}

/*
	if patp == "dev" {
		value, exists := os.LookupEnv("SHIP")
		if exists {
			sockLocation = value
		}
*/

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
