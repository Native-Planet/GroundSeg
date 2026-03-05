package leak

import (
	"errors"
	"fmt"
	"groundseg/broadcast/presenter"
	"groundseg/internal/resource"
	"groundseg/structs"
	"io"
	"net"
	"reflect"

	"github.com/stevelacy/go-urbit/noun"
	"go.uber.org/zap"
)

func listener(patp string, conn net.Conn, info LickStatus) {
	lickMu.Lock()
	BytesChan[patp] = make(chan string)
	readChan := make(chan []byte)
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
		lickMu.Lock()
		delete(BytesChan, patp)
		lickMu.Unlock()
		// close connection
		if conn != nil {
			if closeErr := resource.JoinCloseError(nil, conn, "close leak connection"); closeErr != nil {
				zap.L().Warn(fmt.Sprintf("failed to close leak connection for %s: %v", patp, closeErr))
			}
		}
		zap.L().Info(fmt.Sprintf("Closed lick channel for %s", patp))
	}()
	// Goroutine to read from connection
	go func() {
		buf := make([]byte, 1024) // buffer size can be adjusted
		for {
			n, err := conn.Read(buf)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					zap.L().Error(fmt.Sprintf("Error reading from lick connection for %s: %v", patp, err))
					reportLeakInternalError(patp, info.Auth, fmt.Sprintf("failed to read leak packet: %v", err))
				}
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
		lickMu.RLock()
		patpChan := BytesChan[patp]
		lickMu.RUnlock()
		select {
		// listen to broadcast and send
		case broadcast, ok := <-patpChan:
			if !ok {
				return
			}
			c, err := sendBroadcast(conn, broadcast)
			if err != nil {
				reportLeakInternalError(patp, info.Auth, fmt.Sprintf("failed to send broadcast: %v", err))
				return
			}
			conn = c
		// listen to lick port
		case action, ok := <-readChan:
			if !ok {
				return
			}
			if err := handleAction(patp, action); err != nil {
				if reason, isProtocol := leakProtocolErrorReason(err); isProtocol {
					zap.L().Warn(fmt.Sprintf("leak protocol error from %s: %v", patp, reason))
					continue
				}
				if reason, isInternal := leakInternalErrorReason(err); isInternal {
					reportLeakInternalError(patp, info.Auth, reason)
					continue
				}
				reportLeakInternalError(patp, info.Auth, fmt.Sprintf("failed to handle leak action: %v", err))
			}
		}
	}
}

func updateBroadcast(oldBroadcast, newBroadcast structs.AuthBroadcast) (structs.AuthBroadcast, error) {
	// deep equality check
	if reflect.DeepEqual(oldBroadcast, newBroadcast) {
		return oldBroadcast, nil
	}
	newBroadcastBytes, err := presenter.MarshalAuthorized(newBroadcast)
	if err != nil {
		return oldBroadcast, fmt.Errorf("marshal authorized broadcast for leak: %w", err)
	}
	statuses := GetLickStatuses()
	channels := GetLickChannels()
	var sendErrors []error

	for patp, status := range statuses {
		channel, exists := channels[patp]
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
		scopedBytes, exists, err := presenter.MarshalScoped(newBroadcast, patp)
		if err != nil {
			sendErrors = append(sendErrors, fmt.Errorf("marshal scoped broadcast for %q: %w", patp, err))
			continue
		}
		if !exists {
			continue
		}
		select {
		case channel <- string(scopedBytes):
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
			return nil, fmt.Errorf("send broadcast: %w", err)
		}
	}
	return conn, nil
}
