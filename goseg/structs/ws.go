package structs

// contained herein: structs for managing mutexed maps of
// mutexed websocket connections to avoid panics;
// actual writing is done via broadcast package;
// auth map management is done via auth package
// 🐝 Careful! ❤️

import (
	"sync"

	"github.com/gorilla/websocket"
)

var (
	WsEventBus = make(chan WsChanEvent, 100)
)

// wrapped ws+mutex
type MuConn struct {
	Conn *websocket.Conn
	Mu   sync.RWMutex
}

type WsChanEvent struct {
	Conn *MuConn
	Data []byte
}

// mutexed ws write
// func (ws *MuConn) Write(data []byte) error {
// 	ws.Mu.Lock()
// 	if err := ws.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
// 		ws.Mu.Unlock()
// 		return err
// 	}
// 	ws.Mu.Unlock()
// 	return nil
// }

func (ws *MuConn) Write(data []byte) {
	WsEventBus <- WsChanEvent{Conn: ws, Data: data}
}

// wrappers for mutexed token:websocket maps
// the maps are also mutexed as wholes
type ClientManager struct {
	AuthClients   map[string]*MuConn
	UnauthClients map[string]*MuConn
	Mu            sync.RWMutex
	broadcastMu   sync.Mutex
}

// register a new connection
func (cm *ClientManager) NewConnection(conn *websocket.Conn, tokenId string) *MuConn {
	muConn := &MuConn{Conn: conn}
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	cm.UnauthClients[tokenId] = muConn
	return muConn
}

func (cm *ClientManager) AddAuthClient(id string, client *MuConn) {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	// Add to AuthClients
	cm.AuthClients[id] = client
	// Remove from UnauthClients if present
	if _, ok := cm.UnauthClients[id]; ok {
		delete(cm.UnauthClients, id)
	}
	// Remove any other instances of the same client from UnauthClients
	for token, con := range cm.UnauthClients {
		if con.Conn == client.Conn {
			delete(cm.UnauthClients, token)
		}
	}
}

func (cm *ClientManager) AddUnauthClient(id string, client *MuConn) {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	cm.UnauthClients[id] = client
	// also remove from other map
	if _, ok := cm.AuthClients[id]; ok {
		delete(cm.AuthClients, id)
		for token, con := range cm.AuthClients {
			if con.Conn == client.Conn {
				delete(cm.AuthClients, token)
			}
		}
	}
}

func (cm *ClientManager) BroadcastUnauth(data []byte) {
	cm.broadcastMu.Lock()
	defer cm.broadcastMu.Unlock()
	for _, client := range cm.UnauthClients {
		// imported sessions will be nil until auth
		if client != nil {
			client.Write(data)
		}
	}
}

func (cm *ClientManager) BroadcastAuth(data []byte) {
	cm.broadcastMu.Lock()
	defer cm.broadcastMu.Unlock()
	for _, client := range cm.AuthClients {
		if client != nil {
			client.Write(data)
		}
	}
}

type WsType struct {
	Payload struct {
		Type string `json:"type"`
	} `json:"payload"`
}

type WsPayload struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Payload interface{}   `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsUrbitPayload struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Payload WsUrbitAction `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsNewShipPayload struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload WsNewShipAction `json:"payload"`
	Token   WsTokenStruct   `json:"token"`
}

type WsLogsPayload struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Payload WsLogsAction  `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsLogsAction struct {
	Action      bool   `json:"action"`
	ContainerID string `json:"container_id"`
}

type WsSystemPayload struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Payload WsSystemAction `json:"payload"`
	Token   WsTokenStruct  `json:"token"`
}

type WsNewShipAction struct {
	Type    string `json:"type"`
	Action  string `json:"action"`
	Patp    string `json:"patp"`
	Key     string `json:"key"`
	Remote  bool   `json:"remote"`
	Command string `json:"command"`
}

type WsUrbitAction struct {
	Type   string `json:"type"`
	Action string `json:"action"`
	Patp   string `json:"patp"`
}

type WsSystemAction struct {
	Type    string `json:"type"`
	Action  string `json:"action"`
	Command string `json:"command"`
	Value   int    `json:"value"`
}

type WsTokenStruct struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type WsLoginPayload struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Payload WsLoginAction `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsLoginAction struct {
	Type     string `json:"type"`
	Password string `json:"password"`
}

type WsPwPayload struct {
	ID      string        `json:"id"`
	Payload WsPwAction    `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsPwAction struct {
	Type     string `json:"type"`
	Action   string `json:"action"`
	Old      string `json:"old"`
	Password string `json:"password"`
}

type WsSwapPayload struct {
	ID      string        `json:"id"`
	Payload WsSwapAction  `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsSwapAction struct {
	Type   string `json:"type"`
	Action string `json:"action"`
	Value  int    `json:"value"`
}

type WsLogoutPayload struct {
	ID    string        `json:"id"`
	Token WsTokenStruct `json:"token"`
}

type WsResponsePayload struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Response string        `json:"response"`
	Error    string        `json:"error"`
	Token    WsTokenStruct `json:"token"`
}

type WsStartramPayload struct {
	ID      string           `json:"id"`
	Type    string           `json:"type"`
	Payload WsStartramAction `json:"payload"`
	Token   WsTokenStruct    `json:"token"`
}

type WsStartramAction struct {
	Type     string `json:"type"`
	Action   string `json:"action"`
	Key      string `json:"key"`
	Region   string `json:"region"`
	Endpoint string `json:"endpoint"`
	Reset    bool   `json:"reset"`
}

type WsLogMessage struct {
	Log struct {
		ContainerID string `json:"container_id"`
		Line        string `json:"line"`
	} `json:"log"`
}
