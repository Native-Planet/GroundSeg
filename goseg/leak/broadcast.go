package leak

import (
	"encoding/json"
	"errors"
	"fmt"
	"groundseg/structs"
	"net"
	"reflect"

	"github.com/stevelacy/go-urbit/noun"
	"go.uber.org/zap"
)

func listener(patp string, conn net.Conn, info LickStatus) {
	BytesChan[patp] = make(chan string)
	readChan := make(chan []byte)
	lickMu.Lock()
	// put in status map
	lickStatuses[patp] = info
	lickMu.Unlock()
	defer func() {
		zap.L().Info(fmt.Sprintf("Closing lick channel for %s", patp))
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
		zap.L().Info(fmt.Sprintf("Closed lick channel for %s", patp))
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
		return oldBroadcast, fmt.Errorf("failed to marshal full broadcast for lick: %w", err)
	}
	statuses := GetLickStatuses()
	var sendErrors []error

	for patp, status := range statuses {
		channel, exists := BytesChan[patp]
		if !exists {
			sendErrors = append(sendErrors, fmt.Errorf("no leak channel for %q", patp))
			continue
		}

		if status.Auth {
			select {
			case channel <- string(newBroadcastBytes):
			default:
				sendErrors = append(sendErrors, fmt.Errorf("dropping authorized broadcast for %q", patp))
			}
			continue
		}

		shipInfo, exists := newBroadcast.Urbits[patp]
		if !exists {
			continue
		}

		urbits := map[string]structs.Urbit{
			patp: shipInfo,
		}
		wrappedUrbit := structs.AuthBroadcast{
			Type:      "structure",
			AuthLevel: patp,
			Urbits:    urbits,
		}
		wrappedUrbit.Profile.Startram.Info.Registered = newBroadcast.Profile.Startram.Info.Registered
		wrappedUrbit.Profile.Startram.Info.Running = newBroadcast.Profile.Startram.Info.Running

		wrapperUrbitBytes, err := json.Marshal(wrappedUrbit)
		if err != nil {
			sendErrors = append(sendErrors, fmt.Errorf("marshal scoped broadcast for %q: %w", patp, err))
			continue
		}
		select {
		case channel <- string(wrapperUrbitBytes):
		default:
			sendErrors = append(sendErrors, fmt.Errorf("dropping scoped broadcast for %q", patp))
		}
	}

	if len(sendErrors) > 0 {
		return oldBroadcast, errors.Join(sendErrors...)
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
			zap.L().Error(fmt.Sprintf("Send broadcast error: %v", err))
			return nil, err
		}
	}
	return conn, nil
}
