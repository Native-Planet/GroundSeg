package system

import (
	"encoding/json"
	"fmt"
	"groundseg/structs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hsanjuan/go-captive"
	"go.uber.org/zap"
)

type captiveTransportAdapter struct {
	upgrader websocket.Upgrader
	mu       sync.RWMutex
	clients  map[*websocket.Conn]bool

	connectToWiFi     func(string, string) error
	restartGroundSeg  func() error
	listWirelessSSIDs func(string) []string
}

func newCaptiveTransportAdapter() *captiveTransportAdapter {
	return &captiveTransportAdapter{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		clients:           make(map[*websocket.Conn]bool),
		connectToWiFi:     ConnectToWifi,
		restartGroundSeg:  restartGroundSegService,
		listWirelessSSIDs: ListWifiSSIDs,
	}
}

func restartGroundSegService() error {
	if _, err := runCommand("systemctl", "restart", "groundseg"); err != nil {
		return fmt.Errorf("restart groundseg after captive connect: %w", err)
	}
	return nil
}

func (a *captiveTransportAdapter) runPortal(portal *captive.Portal) error {
	if err := portal.Run(); err != nil {
		return fmt.Errorf("error creating captive portal: %w", err)
	}
	return nil
}

func (a *captiveTransportAdapter) addClient(conn *websocket.Conn) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.clients[conn] = true
}

func (a *captiveTransportAdapter) deactivateClient(conn *websocket.Conn) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if conn != nil {
		a.clients[conn] = false
	}
}

func (a *captiveTransportAdapter) forEachClient(fn func(*websocket.Conn, bool)) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for conn, active := range a.clients {
		fn(conn, active)
	}
}

func (a *captiveTransportAdapter) handleAPI(w http.ResponseWriter, r *http.Request) {
	conn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't upgrade websocket connection: %v", err))
		return
	}
	a.addClient(conn)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) || strings.Contains(err.Error(), "broken pipe") {
				zap.L().Info("WS closed")
				conn.Close()
			}
			a.deactivateClient(conn)
			zap.L().Warn(fmt.Sprintf("Error reading websocket message: %v", err))
			break
		}
		if err := a.processMessage(msg); err != nil {
			zap.L().Error(fmt.Sprintf("captive websocket action failed: %v", err))
		}
	}
}

func (a *captiveTransportAdapter) processMessage(msg []byte) error {
	var payload structs.WsC2cPayload
	if err := json.Unmarshal(msg, &payload); err != nil {
		return fmt.Errorf("unmarshal c2c payload: %w", err)
	}
	if payload.Payload.Action != "connect" {
		return nil
	}
	if err := a.connectToWiFi(payload.Payload.SSID, payload.Payload.Password); err != nil {
		return fmt.Errorf("connect to wifi %s: %w", payload.Payload.SSID, err)
	}
	if err := a.restartGroundSeg(); err != nil {
		return fmt.Errorf("restart groundseg after captive connect: %w", err)
	}
	return nil
}

func (a *captiveTransportAdapter) broadcastNetworks(dev string) {
	tick := time.Tick(2 * time.Second)
	for {
		select {
		case <-tick:
			networks := a.listWirelessSSIDs(dev)
			payload := struct {
				Networks []string `json:"networks"`
			}{
				Networks: networks,
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Error marshaling payload: %v", err))
				continue
			}
			a.forEachClient(func(client *websocket.Conn, active bool) {
				if client != nil && active {
					if err := client.WriteMessage(websocket.TextMessage, payloadJSON); err != nil {
						zap.L().Error(fmt.Sprintf("Error sending message: %v", err))
						a.deactivateClient(client)
					}
				}
			})
		}
	}
}
