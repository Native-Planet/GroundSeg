package transport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hsanjuan/go-captive"
	"go.uber.org/zap"
)

// MessageProcessor handles inbound captive-control messages.
type MessageProcessor func([]byte) error

// WirelessSSIDs lists nearby SSIDs for a given device.
type WirelessSSIDs func(string) ([]string, error)

// CaptiveTransportAdapter manages websocket clients and captive portal message flow.
type CaptiveTransportAdapter struct {
	upgrader websocket.Upgrader
	mu       sync.RWMutex
	clients  map[*websocket.Conn]bool

	listWirelessSSIDs WirelessSSIDs
	processMessage    MessageProcessor
}

// NewCaptiveTransportAdapter constructs a portal adapter with explicit dependencies.
func NewCaptiveTransportAdapter(listWirelessSSIDs WirelessSSIDs, processMessage MessageProcessor) *CaptiveTransportAdapter {
	return &CaptiveTransportAdapter{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		clients:           make(map[*websocket.Conn]bool),
		listWirelessSSIDs: listWirelessSSIDs,
		processMessage:    processMessage,
	}
}

// RunPortal starts the captive portal process.
func (a *CaptiveTransportAdapter) RunPortal(portal *captive.Portal) error {
	if err := portal.Run(); err != nil {
		return fmt.Errorf("error creating captive portal: %w", err)
	}
	return nil
}

// HandleAPI upgrades websocket connections and dispatches inbound messages.
func (a *CaptiveTransportAdapter) HandleAPI(w http.ResponseWriter, r *http.Request) {
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

// BroadcastNetworks loops and pushes discovered SSIDs to all active clients.
func (a *CaptiveTransportAdapter) BroadcastNetworks(dev string) {
	tick := time.Tick(2 * time.Second)
	for {
		select {
		case <-tick:
			networks, err := a.listWirelessSSIDs(dev)
			if err != nil {
				zap.L().Error(fmt.Sprintf("couldn't list wifi networks: %v", err))
				networks = []string{}
			}
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

func (a *CaptiveTransportAdapter) addClient(conn *websocket.Conn) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.clients[conn] = true
}

func (a *CaptiveTransportAdapter) deactivateClient(conn *websocket.Conn) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if conn != nil {
		a.clients[conn] = false
	}
}

func (a *CaptiveTransportAdapter) forEachClient(fn func(*websocket.Conn, bool)) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for conn, active := range a.clients {
		fn(conn, active)
	}
}
