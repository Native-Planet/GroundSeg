package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// wrapped ws+mutex
type MuConn struct {
	Conn       *websocket.Conn
	Mu         sync.RWMutex
	Active     bool
	LastActive time.Time
}

type WsChanEvent struct {
	Conn *MuConn
	Data []byte
}

// mutexed ws write
func (ws *MuConn) Write(data []byte) error {
	if ws.Active {
		ws.Mu.Lock()
		if err := ws.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			ws.Mu.Unlock()
			return err
		}
		ws.Mu.Unlock()
	}
	return nil
}

// mutexed ws read
func (ws *MuConn) Read(cm *ClientManager) (int, []byte, error) {
	if !ws.Active {
		return 0, nil, fmt.Errorf("invalid or inactive websocket connection")
	}
	ws.Mu.RLock()
	messageType, data, err := ws.Conn.ReadMessage()
	ws.Mu.RUnlock()
	return messageType, data, err
}
