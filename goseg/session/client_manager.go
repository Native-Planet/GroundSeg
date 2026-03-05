package session

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/gorilla/websocket"
)

// wrappers for mutexed token:websocket maps
type ClientManager struct {
	Mu    sync.RWMutex
	store *ConnectionRegistry
}

func (cm *ClientManager) ensureState() {
	if cm == nil {
		return
	}
	if cm.store == nil {
		cm.store = newConnectionRegistry(nil, nil)
	}
	cm.store.ensureTargetsInitialized()
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
	return cm.store.hasAuthToken(token)
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
	delete(cm.store.authTokens, token)
}

func (cm *ClientManager) RemoveUnauthorizedToken(token string) {
	if cm == nil {
		return
	}
	cm.ensureState()
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	delete(cm.store.unauthTokens, token)
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

func (cm *ClientManager) BroadcastUnauth(data []byte) error {
	if cm == nil {
		return nil
	}
	cm.ensureState()
	return cm.sendBroadcast(cm.snapshot(false), data, "unauth broadcast")
}

func (cm *ClientManager) BroadcastAuth(data []byte) error {
	if cm == nil {
		return nil
	}
	cm.ensureState()
	return cm.sendBroadcast(cm.snapshot(true), data, "auth broadcast")
}

func (cm *ClientManager) sendBroadcast(target map[string][]*MuConn, data []byte, scope string) error {
	var sendErrors []error
	for token, clients := range target {
		for _, client := range clients {
			if client == nil || !client.Active {
				continue
			}
			if err := client.Write(data); err != nil {
				cm.handleWriteFailure(client.Conn, scope, token, err)
				sendErrors = append(sendErrors, fmt.Errorf("%s to %q: %w", scope, token, err))
				continue
			}
			client.LastActive = time.Now()
		}
	}
	if len(sendErrors) == 0 {
		return nil
	}
	return errors.Join(sendErrors...)
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
