package session

import (
	"fmt"
	"go.uber.org/zap"
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

type ConnectionRegistry struct {
	authClients   map[string][]*MuConn
	unauthClients map[string][]*MuConn
}

type sessionBroadcaster struct {
	mu sync.Mutex
}

func newSessionBroadcaster() *sessionBroadcaster {
	return &sessionBroadcaster{}
}

func newConnectionRegistry(authClients, unauthClients map[string][]*MuConn) *ConnectionRegistry {
	return &ConnectionRegistry{authClients: authClients, unauthClients: unauthClients}
}

func (registry *ConnectionRegistry) ensureTargetsInitialized() {
	if registry == nil {
		return
	}
	if registry.authClients == nil {
		registry.authClients = make(map[string][]*MuConn)
	}
	if registry.unauthClients == nil {
		registry.unauthClients = make(map[string][]*MuConn)
	}
}

func (registry *ConnectionRegistry) findMuConn(conn *websocket.Conn) *MuConn {
	for _, clients := range registry.authClients {
		for _, client := range clients {
			if client != nil && client.Conn == conn {
				return client
			}
		}
	}
	for _, clients := range registry.unauthClients {
		for _, client := range clients {
			if client != nil && client.Conn == conn {
				return client
			}
		}
	}
	return nil
}

func (registry *ConnectionRegistry) hasConnInBucket(clientsByToken map[string][]*MuConn, conn *websocket.Conn) bool {
	for _, clients := range clientsByToken {
		for _, existing := range clients {
			if existing != nil && existing.Conn == conn {
				return true
			}
		}
	}
	return false
}

func (registry *ConnectionRegistry) hasConnInAuthClientBucket(token string, conn *websocket.Conn) bool {
	for _, existing := range registry.authClients[token] {
		if existing != nil && existing.Conn == conn {
			return true
		}
	}
	return false
}

func (registry *ConnectionRegistry) clientBuckets() []map[string][]*MuConn {
	return []map[string][]*MuConn{registry.authClients, registry.unauthClients}
}

func (registry *ConnectionRegistry) bucket(targetAuth bool) map[string][]*MuConn {
	if targetAuth {
		return registry.authClients
	}
	return registry.unauthClients
}

func (registry *ConnectionRegistry) snapshot(targetAuth bool) map[string][]*MuConn {
	source := registry.bucket(targetAuth)
	snapshot := make(map[string][]*MuConn, len(source))
	for token, clients := range source {
		snapshot[token] = append([]*MuConn(nil), clients...)
	}
	return snapshot
}

func (registry *ConnectionRegistry) transitionClientState(id string, client *MuConn, targetAuth bool) {
	registry.enqueueFakeConnIfNilOrUnconnected(id, client, targetAuth)
	if client == nil || client.Conn == nil {
		return
	}
	if targetAuth && !client.Active {
		registry.authClients[id] = appendMuConnIfMissing(registry.authClients[id], client)
		return
	}
	client.Active = true
	registry.removeClientFromOtherBucket(id, client.Conn, targetAuth)
	registry.bucket(targetAuth)[id] = appendMuConnIfMissing(registry.bucket(targetAuth)[id], client)
}

func (registry *ConnectionRegistry) enqueueFakeConnIfNilOrUnconnected(id string, client *MuConn, targetAuth bool) {
	if client != nil && client.Conn != nil {
		return
	}
	bucket := registry.bucket(targetAuth)
	fakeConn := &MuConn{}
	bucket[id] = append(bucket[id], fakeConn)
}

func (registry *ConnectionRegistry) removeClientFromOtherBucket(id string, conn *websocket.Conn, targetAuth bool) {
	sourceMap := registry.unauthClients
	if targetAuth {
		sourceMap = registry.unauthClients
	} else {
		sourceMap = registry.authClients
	}
	source := sourceMap[id]
	if len(source) == 0 {
		return
	}
	filtered, removed := removeConnFromBucket(source, conn)
	if !removed {
		return
	}
	if len(filtered) == 0 {
		delete(sourceMap, id)
		return
	}
	sourceMap[id] = filtered
}

func (registry *ConnectionRegistry) removeConnectionByConn(conn *websocket.Conn) bool {
	removed := false
	for _, clients := range registry.clientBuckets() {
		for key, bucketClients := range clients {
			filtered, bucketRemoved := removeConnFromBucket(bucketClients, conn)
			if !bucketRemoved {
				continue
			}
			removed = true
			if len(filtered) == 0 {
				delete(clients, key)
				continue
			}
			clients[key] = filtered
		}
	}
	return removed
}

func (registry *ConnectionRegistry) cleanupStaleBucket(clientsByToken map[string][]*MuConn, cutoff time.Time) {
	for token, clients := range clientsByToken {
		filtered := clients[:0]
		for _, client := range clients {
			if client == nil || client.Conn == nil {
				filtered = append(filtered, client)
				continue
			}
			if client.Active && client.LastActive.Before(cutoff) {
				continue
			}
			filtered = append(filtered, client)
		}
		if len(filtered) == 0 {
			delete(clientsByToken, token)
			continue
		}
		clientsByToken[token] = filtered
	}
}

func (registry *ConnectionRegistry) hasAuthSession() bool {
	for _, clients := range registry.authClients {
		for _, client := range clients {
			if client.Active {
				return true
			}
		}
	}
	return false
}

func (registry *ConnectionRegistry) broadcast(target map[string][]*MuConn, data []byte, _ string, onFailure func(token string, conn *websocket.Conn, err error)) {
	for token, clients := range target {
		for _, client := range clients {
			if client == nil || !client.Active {
				continue
			}
			if err := client.Write(data); err != nil {
				onFailure(token, client.Conn, err)
				continue
			}
			client.LastActive = time.Now()
		}
	}
}

func (broadcaster *sessionBroadcaster) broadcast(clientsByToken map[string][]*MuConn, data []byte, _ string, onFailure func(token string, conn *websocket.Conn, err error)) {
	broadcaster.mu.Lock()
	defer broadcaster.mu.Unlock()
	for token, clients := range clientsByToken {
		for _, client := range clients {
			if client == nil || !client.Active {
				continue
			}
			if err := client.Write(data); err != nil {
				onFailure(token, client.Conn, err)
				continue
			}
			client.LastActive = time.Now()
		}
	}
}

// wrappers for mutexed token:websocket maps
type ClientManager struct {
	Mu          sync.RWMutex
	store       *ConnectionRegistry
	broadcaster *sessionBroadcaster
}

func (cm *ClientManager) ensureState() {
	if cm == nil {
		return
	}
	if cm.store == nil {
		cm.store = newConnectionRegistry(nil, nil)
	}
	cm.store.ensureTargetsInitialized()
	if cm.broadcaster == nil {
		cm.broadcaster = newSessionBroadcaster()
	}
}

// register a new connection
func (cm *ClientManager) NewConnection(conn *websocket.Conn, tokenId string) *MuConn {
	if cm == nil {
		return nil
	}
	muConn := &MuConn{Conn: conn, Active: true, LastActive: time.Now()}
	return muConn
}

// find *muconn in map or create new one if not present
func (cm *ClientManager) GetMuConn(conn *websocket.Conn, tokenId string) *MuConn {
	if cm == nil {
		return nil
	}
	if conn == nil {
		return nil
	}
	cm.ensureState()
	cm.Mu.RLock()
	if existing := cm.store.findMuConn(conn); existing != nil {
		cm.Mu.RUnlock()
		return existing
	}
	cm.Mu.RUnlock()

	newMuConn := &MuConn{
		Conn:       conn,
		Active:     true,
		LastActive: time.Now(),
	}
	cm.Mu.Lock()
	if existing := cm.store.findMuConn(conn); existing != nil {
		cm.Mu.Unlock()
		return existing
	}
	cm.store.unauthClients[tokenId] = appendMuConnIfMissing(cm.store.unauthClients[tokenId], newMuConn)
	cm.Mu.Unlock()
	return newMuConn
}

func (cm *ClientManager) FindMuConn(conn *websocket.Conn) *MuConn {
	if cm == nil {
		return nil
	}
	if conn == nil {
		return nil
	}
	cm.ensureState()
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	return cm.store.findMuConn(conn)
}

func (cm *ClientManager) HasAuthConnection(token string, conn *websocket.Conn) bool {
	if conn == nil || cm == nil {
		return false
	}
	cm.ensureState()
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	return cm.store.hasConnInAuthClientBucket(token, conn)
}

func (cm *ClientManager) HasAnyAuthConnection(conn *websocket.Conn) bool {
	if conn == nil || cm == nil {
		return false
	}
	cm.ensureState()
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	return cm.store.hasConnInBucket(cm.store.authClients, conn)
}

func (cm *ClientManager) HasAuthSession() bool {
	if cm == nil {
		return false
	}
	cm.ensureState()
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	return cm.store.hasAuthSession()
}

func (cm *ClientManager) HasAuthToken(token string) bool {
	if token == "" || cm == nil {
		return false
	}
	cm.ensureState()
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	_, exists := cm.store.authClients[token]
	return exists
}

func (cm *ClientManager) DeactivateConnection(conn *websocket.Conn) bool {
	if cm == nil {
		return false
	}
	if conn == nil {
		return false
	}
	cm.ensureState()
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	if existing := cm.store.findMuConn(conn); existing != nil {
		existing.Active = false
		return true
	}
	return false
}

func (cm *ClientManager) handleWriteFailure(conn *websocket.Conn, scope, token string, err error) {
	if cm == nil {
		return
	}
	if conn == nil {
		return
	}
	cm.ensureState()
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	removed := cm.removeConnectionByConnLocked(conn)
	if !removed {
		return
	}
	if err != nil {
		zap.L().Warn(fmt.Sprintf("closing websocket client for %q (%s) after write failure: %v", token, scope, err))
	}
	if closeErr := conn.Close(); closeErr != nil {
		zap.L().Warn(fmt.Sprintf("closing websocket client for %q (%s) failed: %v", token, scope, closeErr))
	}
}

func (cm *ClientManager) removeConnectionByConnLocked(conn *websocket.Conn) bool {
	if cm == nil {
		return false
	}
	if conn == nil {
		return false
	}
	return cm.store.removeConnectionByConn(conn)
}

// register wrappers for tests and external state migration
func (cm *ClientManager) RemoveAuthorizedToken(token string) {
	if cm == nil {
		return
	}
	cm.ensureState()
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	delete(cm.store.authClients, token)
}

func (cm *ClientManager) RemoveUnauthorizedToken(token string) {
	if cm == nil {
		return
	}
	cm.ensureState()
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	delete(cm.store.unauthClients, token)
}

func (cm *ClientManager) AuthClientsFor(token string) []*MuConn {
	if cm == nil {
		return nil
	}
	cm.ensureState()
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	return append([]*MuConn(nil), cm.store.authClients[token]...)
}

func (cm *ClientManager) UnauthClientsFor(token string) []*MuConn {
	if cm == nil {
		return nil
	}
	cm.ensureState()
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	return append([]*MuConn(nil), cm.store.unauthClients[token]...)
}

func (cm *ClientManager) AuthClientCount(token string) int {
	return len(cm.AuthClientsFor(token))
}

func (cm *ClientManager) UnauthClientCount(token string) int {
	return len(cm.UnauthClientsFor(token))
}

func (cm *ClientManager) AddAuthClient(id string, client *MuConn) {
	if cm == nil {
		return
	}
	cm.transitionClientState(id, client, true)
}

func (cm *ClientManager) AddUnauthClient(id string, client *MuConn) {
	if cm == nil {
		return
	}
	cm.transitionClientState(id, client, false)
}

func (cm *ClientManager) transitionClientState(id string, client *MuConn, targetAuth bool) {
	if cm == nil {
		return
	}
	cm.ensureState()
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	cm.store.transitionClientState(id, client, targetAuth)
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

func removeConnFromBucket(clients []*MuConn, conn *websocket.Conn) ([]*MuConn, bool) {
	filtered := clients[:0]
	removed := false
	for _, tracked := range clients {
		if tracked == nil || tracked.Conn != conn {
			filtered = append(filtered, tracked)
			continue
		}
		removed = true
	}
	return filtered, removed
}

func (cm *ClientManager) BroadcastUnauth(data []byte) {
	if cm == nil {
		return
	}
	cm.ensureState()
	clients := cm.snapshot(false)
	cm.broadcaster.broadcast(clients, data, "unauth broadcast", func(token string, conn *websocket.Conn, err error) {
		cm.handleWriteFailure(conn, "unauth broadcast", token, err)
	})
}

func (cm *ClientManager) BroadcastAuth(data []byte) {
	if cm == nil {
		return
	}
	cm.ensureState()
	clients := cm.snapshot(true)
	cm.broadcaster.broadcast(clients, data, "auth broadcast", func(token string, conn *websocket.Conn, err error) {
		cm.handleWriteFailure(conn, "auth broadcast", token, err)
	})
}

func (cm *ClientManager) snapshot(targetAuth bool) map[string][]*MuConn {
	if cm == nil {
		return nil
	}
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	return cm.store.snapshot(targetAuth)
}

func (cm *ClientManager) CleanupStaleSessions(timeout time.Duration) {
	if cm == nil {
		return
	}
	cm.ensureState()
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	cutoff := time.Now().Add(-timeout)
	for _, clients := range cm.store.clientBuckets() {
		cm.store.cleanupStaleBucket(clients, cutoff)
	}
}
