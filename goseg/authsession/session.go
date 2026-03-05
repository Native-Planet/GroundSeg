package authsession

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"groundseg/config"
	"groundseg/session"
	"groundseg/structs"
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

func (store *sessionStore) clientManager() *structs.ClientManager {
	if store == nil || store.getClientManager == nil {
		return nil
	}
	return store.getClientManager()
}

func defaultSessionBoundary() SessionBoundary {
	return &sessionStore{
		now:                 func() time.Time { return time.Now() },
		hashToken:           defaultTokenHash,
		persistAuthorized:   persistAuthorizedSession,
		persistUnauthorized: persistUnauthorizedSession,
		getClientManager:    session.GetClientManager,
	}
}

func AddToAuthMapWithBoundary(boundary SessionBoundary, conn *websocket.Conn, token map[string]string, authed bool) error {
	if boundary == nil {
		boundary = defaultSessionBoundary()
	}
	return boundary.AddToAuthMap(conn, token, authed)
}

func RemoveFromAuthMapWithBoundary(boundary SessionBoundary, tokenID string, fromAuthorized bool) {
	if boundary == nil {
		boundary = defaultSessionBoundary()
	}
	boundary.RemoveFromAuthMap(tokenID, fromAuthorized)
}

func AddToAuthMap(conn *websocket.Conn, token map[string]string, authed bool) error {
	return AddToAuthMapWithBoundary(nil, conn, token, authed)
}

func RemoveFromAuthMap(tokenID string, fromAuthorized bool) {
	RemoveFromAuthMapWithBoundary(nil, tokenID, fromAuthorized)
}

func (store *sessionStore) AddToAuthMap(conn *websocket.Conn, token map[string]string, authed bool) error {
	if token == nil {
		return fmt.Errorf("add token to authmap: token is nil")
	}
	if conn == nil {
		return fmt.Errorf("add token to authmap: session is nil")
	}
	tokenStr := token["token"]
	tokenID := token["id"]
	if tokenStr == "" || tokenID == "" {
		return fmt.Errorf("add token to authmap: token id or value is empty")
	}
	clientManager := store.clientManager()
	if clientManager == nil {
		return fmt.Errorf("add token to authmap: client manager unavailable")
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
		clientManager.AddAuthClient(tokenID, muConn)
		zap.L().Info(fmt.Sprintf("%s added to auth", tokenID))
	} else {
		clientManager.AddUnauthClient(tokenID, muConn)
		zap.L().Info(fmt.Sprintf("%s added to unauth", tokenID))
	}

	return nil
}

func (store *sessionStore) RemoveFromAuthMap(tokenID string, fromAuthorized bool) {
	cm := store.clientManager()
	if cm == nil {
		return
	}
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
	if err := config.UpdateConfigTyped(config.WithAuthorizedSession(tokenID, sessionRecord)); err != nil {
		return fmt.Errorf("adding authorized session %s: %w", tokenID, err)
	}
	return nil
}

func persistUnauthorizedSession(tokenID, hash, created string) error {
	sessionRecord := structs.SessionInfo{
		Hash:    hash,
		Created: created,
	}
	if err := config.UpdateConfigTyped(config.WithUnauthorizedSession(tokenID, sessionRecord)); err != nil {
		return fmt.Errorf("adding unauthorized session %s: %w", tokenID, err)
	}
	return nil
}
