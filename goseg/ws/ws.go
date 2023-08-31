package ws

import (
	"encoding/json"
	"fmt"
	"goseg/auth"
	"goseg/broadcast"
	"goseg/config"
	"goseg/handler"
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
				if err = handler.SystemHandler(msg, conn); err != nil {
					config.Logger.Error(fmt.Sprintf("%v", err))
				}
			case "startram":
				if err = handler.StartramHandler(msg); err != nil {
					config.Logger.Error(fmt.Sprintf("%v", err))
				}
			case "urbit":
				if err = handler.UrbitHandler(msg, conn); err != nil {
					config.Logger.Error(fmt.Sprintf("%v", err))
				}
			case "support":
				if err = handler.SupportHandler(msg, payload, r, conn); err != nil {
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
				if err = handler.LoginHandler(conn, msg); err != nil {
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
					config.Logger.Error(fmt.Sprintf("Unable to create token: %v", err))
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
				handler.UnauthHandler(conn, r)
			}
		}
	}
}
