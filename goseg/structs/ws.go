package structs

// contained herein: structs for managing mutexed maps of
// mutexed websocket connections to avoid panics;
// actual writing is done via broadcast package;
// auth map management is done via auth package
// 🐝 Careful! ❤️

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	WsEventBus      = make(chan WsChanEvent, 100)
	InactiveSession = &MuConn{}
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
	ws.Mu.Lock()
	if err := ws.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		ws.Mu.Unlock()
		return err
	}
	ws.Mu.Unlock()
	return nil
}

// func (ws *MuConn) Write(data []byte) {
// 	WsEventBus <- WsChanEvent{Conn: ws, Data: data}
// }

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
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	cm.UnauthClients[tokenId] = append(cm.UnauthClients[tokenId], muConn)
	return muConn
}

func (cm *ClientManager) AddAuthClient(id string, client *MuConn) {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	if client != nil && client.Conn != nil {
		client.Active = true
		if _, ok := cm.UnauthClients[id]; ok {
			// remove from UnauthClients
			for i, con := range cm.UnauthClients[id] {
				if con.Conn == client.Conn {
					cm.UnauthClients[id] = append(cm.UnauthClients[id][:i], cm.UnauthClients[id][i+1:]...)
					break
				}
			}
			if len(cm.UnauthClients[id]) == 0 {
				delete(cm.UnauthClients, id)
			}
		}
		cm.AuthClients[id] = append(cm.AuthClients[id], client)
	} else {
		fakeConn := &MuConn{}
		cm.UnauthClients[id] = append(cm.UnauthClients[id], fakeConn)
	}
}

func (cm *ClientManager) AddUnauthClient(id string, client *MuConn) {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	if client != nil && client.Conn != nil {
		// remove from AuthClients if present
		if _, ok := cm.AuthClients[id]; ok {
			for i, con := range cm.AuthClients[id] {
				if con.Conn == client.Conn {
					cm.AuthClients[id] = append(cm.AuthClients[id][:i], cm.AuthClients[id][i+1:]...)
					break
				}
			}
			if len(cm.AuthClients[id]) == 0 {
				delete(cm.AuthClients, id)
			}
		}
		cm.UnauthClients[id] = append(cm.UnauthClients[id], client)
	} else {
		fakeConn := &MuConn{}
		cm.UnauthClients[id] = append(cm.UnauthClients[id], fakeConn)
	}
}

func (cm *ClientManager) BroadcastUnauth(data []byte) {
	cm.broadcastMu.Lock()
	defer cm.broadcastMu.Unlock()
	for _, clients := range cm.UnauthClients {
		for _, client := range clients {
			if client != nil && client.Active {
				if err := client.Write(data); err != nil {
					client.Active = false
				} else {
					client.LastActive = time.Now()
				}
			}
		}
	}
}

func (cm *ClientManager) BroadcastAuth(data []byte) {
	cm.broadcastMu.Lock()
	defer cm.broadcastMu.Unlock()
	for _, clients := range cm.AuthClients {
		for _, client := range clients {
			if client != nil && client.Active {
				if err := client.Write(data); err != nil {
					client.Active = false
				} else {
					client.LastActive = time.Now()
				}
			}
		}
	}
}

func (cm *ClientManager) CleanupStaleSessions(timeout time.Duration) {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	now := time.Now()
	for token, clients := range cm.AuthClients {
		for i := len(clients) - 1; i >= 0; i-- {
			client := clients[i]
			if client != nil && now.Sub(client.LastActive) > timeout {
				cm.AuthClients[token] = append(cm.AuthClients[token][:i], cm.AuthClients[token][i+1:]...)
			}
		}
		if len(cm.AuthClients[token]) == 0 {
			delete(cm.AuthClients, token)
		}
	}
	for token, clients := range cm.UnauthClients {
		for i := len(clients) - 1; i >= 0; i-- {
			client := clients[i]
			if client != nil && now.Sub(client.LastActive) > timeout {
				cm.UnauthClients[token] = append(cm.UnauthClients[token][:i], cm.UnauthClients[token][i+1:]...)
			}
		}
		if len(cm.UnauthClients[token]) == 0 {
			delete(cm.UnauthClients, token)
		}
	}
}

type WsType struct {
	Payload struct {
		Type string `json:"type"`
	} `json:"payload"`
}

type C2CPayload struct {
	Type     string `json:"type"`
	SSID     string `json:"ssid"`
	Password string `json:"password"`
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

type WsUrbitAction struct {
	Type   string `json:"type"`
	Action string `json:"action"`
	Patp   string `json:"patp"`
	Value  int    `json:value"`
}

type WsNewShipPayload struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload WsNewShipAction `json:"payload"`
	Token   WsTokenStruct   `json:"token"`
}

type WsNewShipAction struct {
	Type    string `json:"type"`
	Action  string `json:"action"`
	Patp    string `json:"patp"`
	Key     string `json:"key"`
	Remote  bool   `json:"remote"`
	Command string `json:"command"`
}

type WsUploadPayload struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Payload WsUploadAction `json:"payload"`
	Token   WsTokenStruct  `json:"token"`
}

type WsUploadAction struct {
	Type     string `json:"type"`
	Action   string `json:"action"`
	Endpoint string `json:"endpoint"`
	Remote   bool   `json:"remote"`
	Fix      bool   `json:"fix"`
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

type WsSystemAction struct {
	Type     string `json:"type"`
	Action   string `json:"action"`
	Command  string `json:"command"`
	Value    int    `json:"value"`
	Update   string `json:"update"`
	SSID     string `json:"ssid"`
	Password string `json:"password"`
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

type WsSetupPayload struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Payload WsSetupAction `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsSetupAction struct {
	Type     string `json:"type"`
	Action   string `json:"action"`
	Password string `json:"password"`
	Key      string `json:"key"`
	Region   string `json:"region"`
}

type WsSupportPayload struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload WsSupportAction `json:"payload"`
	Token   WsTokenStruct   `json:"token"`
}

type WsSupportAction struct {
	Type        string   `json:"type"`
	Action      string   `json:"action"`
	Contact     string   `json:"contact"`
	Description string   `json:"description"`
	Ships       []string `json:"ships"`
	CPUProfile  bool     `json:"cpu_profile"`
}

type WsC2cPayload struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"` // "c2c"
	Payload WsC2cAction `json:"payload"`
}

type WsC2cAction struct {
	Type     string `json:"type"`
	Action   string `json:"action"`
	SSID     string `json:"ssid"`
	Password string `json:"password"`
}
