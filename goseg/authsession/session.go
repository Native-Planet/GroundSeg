package authsession

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"groundseg/config"
	"groundseg/session"
	"groundseg/structs"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type PersistSessionRecord func(tokenID string, hash string, created string) error

type SessionBoundary interface {
	AddToAuthMap(conn *websocket.Conn, token map[string]string, authed bool) error
	RemoveFromAuthMap(tokenID string, fromAuthorized bool)
}

type sessionStore struct {
	now                 func() time.Time
	hashToken           func(string) string
	persistAuthorized   PersistSessionRecord
	persistUnauthorized PersistSessionRecord
	getClientManager    func() *structs.ClientManager
}

var activeSessionBoundary SessionBoundary = &sessionStore{
	now:                 func() time.Time { return time.Now() },
	hashToken:           defaultTokenHash,
	persistAuthorized:   persistAuthorizedSession,
	persistUnauthorized: persistUnauthorizedSession,
	getClientManager:    session.GetClientManager,
}

var activeSessionBoundaryMu sync.RWMutex

func activeBoundary() SessionBoundary {
	activeSessionBoundaryMu.RLock()
	defer activeSessionBoundaryMu.RUnlock()
	return activeSessionBoundary
}

func SetSessionBoundary(boundary SessionBoundary) {
	activeSessionBoundaryMu.Lock()
	defer activeSessionBoundaryMu.Unlock()
	if boundary == nil {
		activeSessionBoundary = newSessionStore()
		return
	}
	activeSessionBoundary = boundary
}

func AddToAuthMap(conn *websocket.Conn, token map[string]string, authed bool) error {
	return activeBoundary().AddToAuthMap(conn, token, authed)
}

func RemoveFromAuthMap(tokenID string, fromAuthorized bool) {
	activeBoundary().RemoveFromAuthMap(tokenID, fromAuthorized)
}

func (store *sessionStore) AddToAuthMap(conn *websocket.Conn, token map[string]string, authed bool) error {
	if token == nil {
		return fmt.Errorf("Can't add nil token to authmap")
	}
	if conn == nil {
		return fmt.Errorf("Can't add nil session to authmap")
	}
	tokenStr := token["token"]
	tokenID := token["id"]
	if tokenStr == "" || tokenID == "" {
		return fmt.Errorf("Can't add token with empty id or token value to authmap")
	}

	hashed := store.hashToken(tokenStr)
	now := store.now().Format("2006-01-02_15:04:05")
	if authed {
		if err := store.persistAuthorized(tokenID, hashed, now); err != nil {
			return err
		}
	} else {
		if err := store.persistUnauthorized(tokenID, hashed, now); err != nil {
			return err
		}
	}

	muConn := &structs.MuConn{Conn: conn, Active: true}
	if authed {
		store.getClientManager().AddAuthClient(tokenID, muConn)
		zap.L().Info(fmt.Sprintf("%s added to auth", tokenID))
	} else {
		store.getClientManager().AddUnauthClient(tokenID, muConn)
		zap.L().Info(fmt.Sprintf("%s added to unauth", tokenID))
	}

	return nil
}

func (store *sessionStore) RemoveFromAuthMap(tokenID string, fromAuthorized bool) {
	cm := store.getClientManager()
	if fromAuthorized {
		cm.RemoveAuthorizedToken(tokenID)
		return
	}
	cm.RemoveUnauthorizedToken(tokenID)
}

func defaultTokenHash(tokenValue string) string {
	hashed := sha512.Sum512([]byte(tokenValue))
	return hex.EncodeToString(hashed[:])
}

func newSessionStore() *sessionStore {
	return &sessionStore{
		now:                 func() time.Time { return time.Now() },
		hashToken:           defaultTokenHash,
		persistAuthorized:   persistAuthorizedSession,
		persistUnauthorized: persistUnauthorizedSession,
		getClientManager:    session.GetClientManager,
	}
}

func persistAuthorizedSession(tokenID, hash, created string) error {
	sessionRecord := structs.SessionInfo{
		Hash:    hash,
		Created: created,
	}
	if err := config.UpdateConfTyped(config.WithAuthorizedSession(tokenID, sessionRecord)); err != nil {
		return fmt.Errorf("Error adding session: %w", err)
	}
	return nil
}

func persistUnauthorizedSession(tokenID, hash, created string) error {
	sessionRecord := structs.SessionInfo{
		Hash:    hash,
		Created: created,
	}
	if err := config.UpdateConfTyped(config.WithUnauthorizedSession(tokenID, sessionRecord)); err != nil {
		return fmt.Errorf("Error adding session: %w", err)
	}
	return nil
}
