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

// find *muconn in map or create new one if not present
func (cm *ClientManager) GetMuConn(conn *websocket.Conn, tokenId string) *MuConn {
	if conn == nil {
		return nil
	}
	if existing := cm.FindMuConn(conn); existing != nil {
		return existing
	}

	newMuConn := &MuConn{
		Conn:       conn,
		Active:     true,
		LastActive: time.Now(),
	}
	cm.Mu.Lock()
	if existing := cm.findMuConnLocked(conn); existing != nil {
		cm.Mu.Unlock()
		return existing
	}
	cm.UnauthClients[tokenId] = append(cm.UnauthClients[tokenId], newMuConn)
	cm.Mu.Unlock()
	return newMuConn
}

func (cm *ClientManager) findMuConnLocked(conn *websocket.Conn) *MuConn {
	for _, muConns := range cm.AuthClients {
		for _, muConn := range muConns {
			if muConn != nil && muConn.Conn == conn {
				return muConn
			}
		}
	}
	for _, muConns := range cm.UnauthClients {
		for _, muConn := range muConns {
			if muConn != nil && muConn.Conn == conn {
				return muConn
			}
		}
	}
	return nil
}

func (cm *ClientManager) FindMuConn(conn *websocket.Conn) *MuConn {
	if conn == nil {
		return nil
	}
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	return cm.findMuConnLocked(conn)
}

func (cm *ClientManager) HasAuthConnection(token string, conn *websocket.Conn) bool {
	if conn == nil {
		return false
	}
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	for _, existing := range cm.AuthClients[token] {
		if existing != nil && existing.Conn == conn {
			return true
		}
	}
	return false
}

func (cm *ClientManager) HasAnyAuthConnection(conn *websocket.Conn) bool {
	if conn == nil {
		return false
	}
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	for _, clients := range cm.AuthClients {
		for _, existing := range clients {
			if existing != nil && existing.Conn == conn {
				return true
			}
		}
	}
	return false
}

func (cm *ClientManager) HasConnection(conn *websocket.Conn) bool {
	return cm.FindMuConn(conn) != nil
}

func (cm *ClientManager) HasAuthSession() bool {
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	for _, clients := range cm.AuthClients {
		for _, client := range clients {
			if client.Active {
				return true
			}
		}
	}
	return false
}

func (cm *ClientManager) DeactivateConnection(conn *websocket.Conn) bool {
	if conn == nil {
		return false
	}
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	if existing := cm.findMuConnLocked(conn); existing != nil {
		existing.Active = false
		return true
	}
	return false
}

// wrappers for mutexed token:websocket maps
// the maps are also mutexed as wholes
type ClientManager struct {
	AuthClients   map[string][]*MuConn
	UnauthClients map[string][]*MuConn
	Mu            sync.RWMutex
	broadcastMu   sync.Mutex
}

// register a new connection
func (cm *ClientManager) NewConnection(conn *websocket.Conn, tokenId string) *MuConn {
	muConn := &MuConn{Conn: conn, Active: true, LastActive: time.Now()}
	return muConn
}

func (cm *ClientManager) AddAuthClient(id string, client *MuConn) {
	cm.transitionClientStateLocked(id, client, true)
}

func (cm *ClientManager) AddUnauthClient(id string, client *MuConn) {
	cm.transitionClientStateLocked(id, client, false)
}

func (cm *ClientManager) transitionClientStateLocked(id string, client *MuConn, targetAuth bool) {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	cm.enqueueFakeConnIfNilOrUnconnected(id, client, targetAuth)
	if client == nil || client.Conn == nil {
		return
	}
	if targetAuth && !client.Active {
		cm.AuthClients[id] = appendMuConnIfMissing(cm.AuthClients[id], client)
		return
	}
	client.Active = true
	cm.removeClientFromOtherBucket(id, client.Conn, targetAuth)
	if targetAuth {
		cm.AuthClients[id] = appendMuConnIfMissing(cm.AuthClients[id], client)
		return
	}
	cm.UnauthClients[id] = appendMuConnIfMissing(cm.UnauthClients[id], client)
}

func (cm *ClientManager) enqueueFakeConnIfNilOrUnconnected(id string, client *MuConn, targetAuth bool) {
	if client != nil && client.Conn != nil {
		return
	}
	fakeConn := &MuConn{}
	if targetAuth {
		cm.AuthClients[id] = append(cm.AuthClients[id], fakeConn)
		return
	}
	cm.UnauthClients[id] = append(cm.UnauthClients[id], fakeConn)
}

func (cm *ClientManager) removeClientFromOtherBucket(id string, conn *websocket.Conn, targetAuth bool) {
	sourceMap := cm.UnauthClients
	if targetAuth {
		sourceMap = cm.UnauthClients
	} else {
		sourceMap = cm.AuthClients
	}
	source := sourceMap[id]
	if len(source) == 0 {
		return
	}
	for i, con := range source {
		if con != nil && con.Conn == conn {
			source = append(source[:i], source[i+1:]...)
			break
		}
	}
	if len(source) == 0 {
		delete(sourceMap, id)
		return
	}
	sourceMap[id] = source
}

func appendMuConnIfMissing(existingClients []*MuConn, client *MuConn) []*MuConn {
	if client == nil {
		return existingClients
	}
	for _, existingClient := range existingClients {
		if existingClient != nil && existingClient.Conn == client.Conn {
			return existingClients
		}
	}
	existingClients = append(existingClients, client)
	return existingClients
}

func (cm *ClientManager) BroadcastUnauth(data []byte) {
	cm.broadcastMu.Lock()
	defer cm.broadcastMu.Unlock()
	for _, clients := range cm.UnauthClients {
		for _, client := range clients {
			if client == nil || !client.Active {
				continue
			}
			if err := client.Write(data); err != nil {
				client.Active = false
				continue
			}
			client.LastActive = time.Now()
		}
	}
}

func (cm *ClientManager) BroadcastAuth(data []byte) {
	cm.broadcastMu.Lock()
	defer cm.broadcastMu.Unlock()
	for _, clients := range cm.AuthClients {
		for _, client := range clients {
			if client == nil || !client.Active {
				continue
			}
			if err := client.Write(data); err != nil {
				client.Active = false
				continue
			}
			client.LastActive = time.Now()
		}
	}
}

func (cm *ClientManager) CleanupStaleSessions(timeout time.Duration) {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	cutoff := time.Now().Add(-timeout)
	for token, clients := range cm.AuthClients {
		for i := len(clients) - 1; i >= 0; i-- {
			client := clients[i]
			if client == nil || client.Conn == nil {
				continue
			}
			if client.Active && client.LastActive.Before(cutoff) {
				clients = append(clients[:i], clients[i+1:]...)
			}
		}
		if len(clients) == 0 {
			delete(cm.AuthClients, token)
			continue
		}
		cm.AuthClients[token] = clients
	}
	for token, clients := range cm.UnauthClients {
		for i := len(clients) - 1; i >= 0; i-- {
			client := clients[i]
			if client == nil || client.Conn == nil {
				continue
			}
			if client.Active && client.LastActive.Before(cutoff) {
				clients = append(clients[:i], clients[i+1:]...)
			}
		}
		if len(clients) == 0 {
			delete(cm.UnauthClients, token)
			continue
		}
		cm.UnauthClients[token] = clients
	}
}
