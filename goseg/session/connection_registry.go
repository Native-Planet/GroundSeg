package session

import (
	"time"

	"github.com/gorilla/websocket"
)

type ConnectionRegistry struct {
	authClients   map[string][]*MuConn
	unauthClients map[string][]*MuConn
	authTokens    map[string]struct{}
	unauthTokens  map[string]struct{}
}

func newConnectionRegistry(authClients, unauthClients map[string][]*MuConn) *ConnectionRegistry {
	return &ConnectionRegistry{
		authClients:   authClients,
		unauthClients: unauthClients,
		authTokens:    make(map[string]struct{}),
		unauthTokens:  make(map[string]struct{}),
	}
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
	if registry.authTokens == nil {
		registry.authTokens = make(map[string]struct{})
	}
	if registry.unauthTokens == nil {
		registry.unauthTokens = make(map[string]struct{})
	}
}

func (registry *ConnectionRegistry) findMuConn(conn *websocket.Conn) *MuConn {
	if conn == nil {
		return nil
	}
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
	return hasConnInBuckets(clientsByToken, conn, func(muConn *MuConn) *websocket.Conn { return muConn.Conn })
}

func (registry *ConnectionRegistry) hasConnInAuthClientBucket(token string, conn *websocket.Conn) bool {
	for _, existing := range registry.authClients[token] {
		if existing != nil && existing.Conn == conn {
			return true
		}
	}
	return false
}

func (registry *ConnectionRegistry) hasAuthToken(token string) bool {
	_, ok := registry.authTokens[token]
	return ok
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
	return snapshotBuckets(registry.bucket(targetAuth), func(muConn *MuConn) *MuConn { return muConn })
}

func (registry *ConnectionRegistry) transitionClientState(id string, client *MuConn, targetAuth bool) {
	if id == "" {
		return
	}
	registry.registerToken(id, targetAuth)
	if client == nil || client.Conn == nil || !client.Active {
		return
	}
	client.Active = true
	registry.removeClientFromOtherBucket(id, client.Conn, targetAuth)
	registry.bucket(targetAuth)[id] = appendMuConnIfMissing(registry.bucket(targetAuth)[id], client)
}

func (registry *ConnectionRegistry) registerToken(id string, authed bool) {
	if id == "" {
		return
	}
	if authed {
		registry.authTokens[id] = struct{}{}
		return
	}
	registry.unauthTokens[id] = struct{}{}
}

func (registry *ConnectionRegistry) removeToken(id string, authed bool) {
	if id == "" {
		return
	}
	if authed {
		delete(registry.authTokens, id)
		return
	}
	delete(registry.unauthTokens, id)
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
	return removeConnFromBuckets(registry.authClients, conn, func(muConn *MuConn) *websocket.Conn { return muConn.Conn }) ||
		removeConnFromBuckets(registry.unauthClients, conn, func(muConn *MuConn) *websocket.Conn { return muConn.Conn })
}

func (registry *ConnectionRegistry) cleanupStaleBucket(clientsByToken map[string][]*MuConn, cutoff time.Time) {
	for token, clients := range clientsByToken {
		filtered := clients[:0]
		for _, client := range clients {
			if client == nil || client.Conn == nil {
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
			if client != nil && client.Active {
				return true
			}
		}
	}
	return false
}

func appendMuConnIfMissing(existingClients []*MuConn, client *MuConn) []*MuConn {
	return appendIfConnMissing(existingClients, client, func(muConn *MuConn) *websocket.Conn { return muConn.Conn })
}

func removeConnFromBucket(clients []*MuConn, conn *websocket.Conn) ([]*MuConn, bool) {
	return removeConn(clients, conn, func(muConn *MuConn) *websocket.Conn { return muConn.Conn })
}
