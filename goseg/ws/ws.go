package ws

import (
	"encoding/json"
	"fmt"
	"goseg/auth"
	"goseg/broadcast"
	"goseg/config"
	"goseg/structs"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins
		},
	}
)

// switch on ws event cases
func WsHandler(w http.ResponseWriter, r *http.Request) {
	conf := config.Conf()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		config.Logger.Error(fmt.Sprintf("Couldn't upgrade websocket connection: %v", err))
		return
	}
	// keepalive for ws
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	pingInterval := 15 * time.Second
	go func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				config.Logger.Info("WS closed")
				conn.Close()
				break
			}
			config.Logger.Error(fmt.Sprintf("Error reading websocket message: %v", err))
			continue
		}
		var payload structs.WsPayload
		if err := json.Unmarshal(msg, &payload); err != nil {
			config.Logger.Warn(fmt.Sprintf("Error unmarshalling payload: %v", err))
			continue
		}
		config.Logger.Info(fmt.Sprintf("Received message: %s", string(msg)))
		var msgType structs.WsType
		err = json.Unmarshal(msg, &msgType)
		if err != nil {
			config.Logger.Warn(fmt.Sprintf("Error marshalling token (else): %v", err))
			continue
		}
		token := map[string]string{
			"id":    payload.Token.ID,
			"token": payload.Token.Token,
		}
		tokenContent, authed := auth.CheckToken(token, conn, r, conf.FirstBoot)
		token = map[string]string{
			"id":    payload.Token.ID,
			"token": tokenContent,
		}
		if authed {
			switch msgType.Payload.Type {
			case "new_ship":
				config.Logger.Info("New ship")
			case "pier_upload":
				config.Logger.Info("Pier upload")
			case "password":
				config.Logger.Info("Password")
			case "system":
				config.Logger.Info("System")
			case "startram":
				config.Logger.Info("StarTram")
			case "urbit":
				config.Logger.Info("Urbit")
				if err = urbitHandler(msg, conn); err != nil {
					config.Logger.Error(fmt.Sprintf("%v", err))
				}
			case "support":
				if err = supportHandler(msg, payload, r, conn); err != nil {
					config.Logger.Error(fmt.Sprintf("%v", err))
				}
			case "broadcast":
				if err := broadcast.BroadcastToClients(); err != nil {
					config.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
				}
			case "login":
				// already authed so lets get you on the map
				if err := auth.AddToAuthMap(conn, token, true); err != nil {
					config.Logger.Error(fmt.Sprintf("Unable to reauth: %v", err))
				}
				if err := broadcast.BroadcastToClients(); err != nil {
					config.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
				}
			case "verify":
				result := map[string]interface{}{
					"type":     "activity",
					"id":       payload.ID, // this is like the action id
					"error":    "null",
					"response": "ack",
					"token":    token,
				}
				respJson, err := json.Marshal(result)
				if err != nil {
					errmsg := fmt.Sprintf("Error marshalling token (init): %v", err)
					config.Logger.Error(errmsg)
				}
				if err := conn.WriteMessage(websocket.TextMessage, respJson); err != nil {
					continue
				}
				if err := auth.AddToAuthMap(conn, token, true); err != nil {
					config.Logger.Error(fmt.Sprintf("Unable to reauth: %v", err))
				}
				if err := broadcast.BroadcastToClients(); err != nil {
					config.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
				}
			default:
				errmsg := fmt.Sprintf("Unknown auth request type: %s", msgType.Payload.Type)
				config.Logger.Warn(errmsg)
				if err := broadcast.BroadcastToClients(); err != nil {
					config.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
				}
			}
		} else {
			switch msgType.Payload.Type {
			case "login":
				if err = loginHandler(conn, msg); err != nil {
					config.Logger.Error(fmt.Sprintf("%v", err))
				}
			case "setup":
				config.Logger.Info("Setup")
				// setup.Setup(payload)
			case "verify":
				config.Logger.Info("New client")
				// auth.CreateToken also adds to unauth map
				newToken, err := auth.CreateToken(conn, r, false)
				if err != nil {
					config.Logger.Error(fmt.Sprintf("Unable to create token: %v",err))
				}
				result := map[string]interface{}{
					"type":     "activity",
					"id":       payload.ID,
					"error":    "null",
					"response": "ack",
					"token":    newToken,
				}
				respJson, err := json.Marshal(result)
				if err != nil {
					errmsg := fmt.Sprintf("Error marshalling token (init): %v", err)
					config.Logger.Error(errmsg)
				}
				if err := conn.WriteMessage(websocket.TextMessage, respJson); err != nil {
					continue
				}
			default:
				unauthHandler(conn, r)
			}
		}
	}
}

// validate password and add to auth session map
func loginHandler(conn *websocket.Conn, msg []byte) error {
	var loginPayload structs.WsLoginPayload
	err := json.Unmarshal(msg, &loginPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal login payload: %v", err)
	}
	isAuthenticated := auth.AuthenticateLogin(loginPayload.Payload.Password)
	if isAuthenticated {
		token := map[string]string{
			"id":    loginPayload.Token.ID,
			"token": loginPayload.Token.Token,
		}
		tokenDebug, _ := json.Marshal(token)
		fmt.Printf(string(tokenDebug))
		if err := auth.AddToAuthMap(conn, token, true); err != nil {
			return fmt.Errorf("Unable to process login: %v", err)
		}
	} else {
		return fmt.Errorf("Failed auth: %v",loginPayload.Payload.Password)
	}
	if err := broadcast.BroadcastToClients(); err != nil {
		config.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
	}
	config.Logger.Info("Session %s logged in",loginPayload.Token.ID)
	return nil
}

// broadcast the unauth payload
func unauthHandler(conn *websocket.Conn, r *http.Request) {
	config.Logger.Info("Sending unauth broadcast")
	blob := structs.UnauthBroadcast{
		Type:      "structure",
		AuthLevel: "unauthorized",
		Login: struct {
			Remainder int `json:"remainder"`
		}{
			Remainder: 0,
		},
	}
	resp, err := json.Marshal(blob)
	if err != nil {
		config.Logger.Error(fmt.Sprintf("Error unmarshalling message: %v", err))
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, resp); err != nil {
		config.Logger.Error(fmt.Sprintf("Error writing unauth response: %v", err))
		return
	}
}

// handle bug report stuff
func supportHandler(msg []byte, payload structs.WsPayload, r *http.Request, conn *websocket.Conn) error {
	config.Logger.Info("Support")
	return nil
}

// handle urbit-type events
func urbitHandler(msg []byte, conn *websocket.Conn) error {
	config.Logger.Info("Urbit")
	var urbitPayload structs.WsUrbitPayload
	err := json.Unmarshal(msg, &urbitPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal urbit payload: %v", err)
	}
	patp := urbitPayload.Payload.Patp
	shipConf := config.UrbitConf(patp)
	switch urbitPayload.Payload.Action {
	case "toggle-network":
		currentNetwork := shipConf.Network
		conf := config.Conf()
		if currentNetwork == "wireguard" {
			shipConf.Network = "bridge"
			update := make(map[string]structs.UrbitDocker)
			update[patp] = shipConf
			if err := config.UpdateUrbitConfig(update); err != nil {
				return fmt.Errorf("Couldn't update urbit config: %v",err)
			}
			return nil
		} else if currentNetwork == "bridge" && conf.WgRegistered == true {
			shipConf.Network = "wireguard"
			update := make(map[string]structs.UrbitDocker)
			update[patp] = shipConf
			if err := config.UpdateUrbitConfig(update); err != nil {
				return fmt.Errorf("Couldn't update urbit config: %v",err)
			}
			if err := broadcast.BroadcastToClients(); err != nil {
				config.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
			}
			return nil
		} else {
			return fmt.Errorf("No remote registration")
		}
	case "toggle-devmode":
		if shipConf.DevMode == true {
			shipConf.DevMode = false
		} else {
			shipConf.DevMode = true
		}
		update := make(map[string]structs.UrbitDocker)
		update[patp] = shipConf
		if err := config.UpdateUrbitConfig(update); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %v",err)
		}
		if err := broadcast.BroadcastToClients(); err != nil {
			config.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
		}
		return nil
	default:
		return fmt.Errorf("Unrecognized urbit action: %v",urbitPayload.Payload.Type)
	}
}
