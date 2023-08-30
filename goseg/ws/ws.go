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
			case "support":
				if err = supportHandler(msg, payload, r, conn); err != nil {
					config.Logger.Error(fmt.Sprintf("%v", err))
				}
			case "broadcast":
				if err := broadcast.BroadcastToClients(); err != nil {
					config.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
				}
			case "login":
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
				if err = loginHandler(conn, msg, payload); err != nil {
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
					config.Logger.Error("Unable to create token")
				}
				result := map[string]interface{}{
					"type":     "activity",
					"id":       payload.ID, // this is like the action id
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
func loginHandler(conn *websocket.Conn, msg []byte, payload structs.WsPayload) error {
	config.Logger.Info("Login")
	// lets do this ugly shit to get the password out
	var msgMap map[string]interface{}
	err := json.Unmarshal(msg, &msgMap)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal login bytes: %v",err)
	}
	payloadData, ok := msgMap["payload"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("Couldn't extract payload: %v",err)
	}
	payloadBytes, err := json.Marshal(payloadData)
	if err != nil {
		return fmt.Errorf("Couldn't remarshal login data: %v",err)
	}
	var loginPayload structs.WsLoginPayload
	err = json.Unmarshal(payloadBytes, &loginPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal login payload: %v",err)
	}
	isAuthenticated := auth.AuthenticateLogin(loginPayload.Password)
	if isAuthenticated {
		token := map[string]string{
			"id":    payload.Token.ID,
			"token": payload.Token.Token,
		}
		if err := auth.AddToAuthMap(conn, token, true); err != nil {
			return fmt.Errorf("Unable to process login: %v", err)
		}
	} else {
		config.Logger.Info("Login failed")
		return fmt.Errorf("Failed auth")
	}
	if err := broadcast.BroadcastToClients(); err != nil {
		config.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
	}
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

// client send:
// {
// 	"type": "verify",
// 	"id": "jsgeneratedid",
// 	"token<optional>": {
// 	  "id": "servergeneratedid",
// 	  "token": "encryptedtext"
// 	}
// }

// 1. we decrypt the token
// 2. we modify token['authorized'] to true
// 3. remove it from 'unauthorized' in system.json
// 4. hash and add to 'authozired' in system.json
// 5. encrypt that, and send it back to the user

// server respond:
// {
// 	"type": "activity",
// 	"response": "ack/nack",
// 	"error": "null/<some_error>",
// 	"id": "jsgeneratedid",
// 	"token": { (either new token or the token the user sent us)
// 	  "id": "relevant_token_id",
// 	  "token": "encrypted_text"
// 	}
// }

func supportHandler(msg []byte, payload structs.WsPayload, r *http.Request, conn *websocket.Conn) error {
	config.Logger.Info("Support")
	return nil
}
