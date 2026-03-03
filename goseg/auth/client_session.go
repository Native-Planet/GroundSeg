package auth

import (
	"errors"
	"fmt"
	"groundseg/authsession"
	"groundseg/session"
	"groundseg/structs"
	"strings"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

func NewClientManager() *structs.ClientManager {
	return session.NewClientManager()
}

func GetClientManager() *structs.ClientManager {
	return session.GetClientManager()
}

func SetClientManager(cm *structs.ClientManager) {
	if cm == nil {
		zap.L().Warn("ignoring nil auth client manager assignment")
		return
	}
	session.SetClientManager(cm)
}

func IsTokenIdAuthed(token string) bool {
	if strings.TrimSpace(token) == "" {
		return false
	}
	return TokenIdAuthed(GetClientManager(), token)
}

func GetMuConn(conn *websocket.Conn, tokenId string) *structs.MuConn {
	if conn == nil || tokenId == "" {
		return nil
	}
	return GetClientManager().GetMuConn(conn, tokenId)
}

func ReadMuConn(muConn *structs.MuConn) (int, []byte, error) {
	if muConn == nil {
		return 0, nil, fmt.Errorf("invalid websocket session")
	}
	return muConn.Read(GetClientManager())
}

func WsIsAuthenticated(conn *websocket.Conn, token string) bool {
	if conn == nil || strings.TrimSpace(token) == "" {
		return false
	}
	return GetClientManager().HasAuthConnection(token, conn)
}

func WsAuthCheck(conn *websocket.Conn) bool {
	if conn == nil {
		return false
	}
	return GetClientManager().HasAnyAuthConnection(conn)
}

func WsNilSession(conn *websocket.Conn) error {
	if conn == nil {
		return fmt.Errorf("invalid session")
	}
	if GetClientManager().DeactivateConnection(conn) {
		return nil
	}
	return fmt.Errorf("Session not in client manager")
}

func TokenIdAuthed(clientManager *structs.ClientManager, token string) bool {
	if clientManager == nil || strings.TrimSpace(token) == "" {
		return false
	}
	exists := clientManager.HasAuthToken(token)
	zap.L().Debug(fmt.Sprintf("%s present in authmap: %v", token, exists))
	return exists
}

func AddToAuthMap(conn *websocket.Conn, token map[string]string, authed bool) error {
	if conn == nil {
		return errors.New("missing websocket connection")
	}
	if token == nil {
		return errors.New("missing token map")
	}
	if token["id"] == "" || token["token"] == "" {
		return errors.New("token map must include id and token")
	}
	return authsession.AddToAuthMap(conn, token, authed)
}

func RemoveFromAuthMap(tokenId string, fromAuthorized bool) {
	if strings.TrimSpace(tokenId) == "" {
		return
	}
	authsession.RemoveFromAuthMap(tokenId, fromAuthorized)
}
