package ws

import (
	"encoding/json"
	"fmt"
	"groundseg/auth"
	"groundseg/broadcast"
	"groundseg/config"
	"groundseg/handler"
	"groundseg/logger"
	"groundseg/setup"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/system"
	"net/http"
	"strings"

	// "time"

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
	conn, err := upgrader.Upgrade(w, r, nil)
	logger.Logger.Debug("New ws session")
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Couldn't upgrade websocket connection: %v", err))
		return
	}
	// initial handling before we assign ws session to mutex
	_, msg, err := conn.ReadMessage()
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) || strings.Contains(err.Error(), "broken pipe") {
			logger.Logger.Debug("WS closed")
			conn.Close()
			// cancel all log streams for this ws
			auth.ClientManager.Mu.RLock()
			for _, clients := range auth.ClientManager.AuthClients {
				for _, client := range clients {
					if client != nil && client.Conn == conn {
						logEvent := structs.LogsEvent{
							Action:      false,
							ContainerID: "all",
							MuCon:       client,
						}
						config.LogsEventBus <- logEvent
						break
					}
					break
				}
			}
			auth.ClientManager.Mu.RUnlock()
			// mute the session
			auth.WsNilSession(conn)
		}
		logger.Logger.Debug(fmt.Sprintf("WS error: %v", err))
	}
	// get the token from payload and check vs auth map
	var payload structs.WsPayload
	if err := json.Unmarshal(msg, &payload); err != nil {
		logger.Logger.Error(fmt.Sprintf("Error unmarshalling payload: %v", err))
	}
	tokenId := payload.Token.ID
	logger.Logger.Debug(fmt.Sprintf("New WS session for %v", tokenId))
	MuCon := auth.ClientManager.GetMuConn(conn, tokenId)
	token := map[string]string{
		"id":    payload.Token.ID,
		"token": payload.Token.Token,
	}
	tokenContent, authed := auth.CheckToken(token, conn, r)
	token = map[string]string{
		"id":    payload.Token.ID,
		"token": tokenContent,
	}
	conf := config.Conf()
	// if in c2cmode
	isC2C := system.IsC2CMode()
	result := map[string]interface{}{
		"type":  "c2c",
		"ssids": system.C2CStoredSSIDs,
	}
	respJson, err := json.Marshal(result)
	if err != nil {
		errmsg := fmt.Sprintf("Error marshalling c2c SSIDs: %v", err)
		logger.Logger.Error(errmsg)
	}
	MuCon.Write(respJson)
	if isC2C {
		// send setup broadcast if we're not done setting up
	} else if conf.Setup != "complete" {
		resp := structs.SetupBroadcast{
			Type:      "structure",
			AuthLevel: "setup",
			Stage:     conf.Setup,
			Page:      setup.Stages[conf.Setup],
			Regions:   startram.Regions,
		}
		respJSON, err := json.Marshal(resp)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't marshal startram regions: %v", err))
		}
		MuCon.Write(respJSON)
	} else if !authed {
		var ack string
		token, err = auth.CreateToken(conn, r, false)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Unable to create token: %v", err))
			ack = "nack"
		}
		result := map[string]interface{}{
			"type":     "activity",
			"id":       payload.ID,
			"error":    "null",
			"response": ack,
			"token":    token,
		}
		respJson, err := json.Marshal(result)
		if err != nil {
			errmsg := fmt.Sprintf("Error marshalling token (init): %v", err)
			logger.Logger.Error(errmsg)
		}
		MuCon.Write(respJson)
	}
	if err := auth.AddToAuthMap(conn, token, authed); err != nil {
		logger.Logger.Error(fmt.Sprintf("Unable to track auth session: %v", err))
	}
	for {
		// mutexed read operations
		_, msg, err := MuCon.Read(auth.ClientManager)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) || strings.Contains(err.Error(), "broken pipe") {
				logger.Logger.Debug("WS closed")
				conn.Close()
				// cancel all log streams for this ws
				auth.ClientManager.Mu.RLock()
				for _, clients := range auth.ClientManager.AuthClients {
					for _, client := range clients {
						if client != nil && client.Conn == conn {
							logEvent := structs.LogsEvent{
								Action:      false,
								ContainerID: "all",
								MuCon:       client,
							}
							config.LogsEventBus <- logEvent
							break
						}
						break
					}
				}
				auth.ClientManager.Mu.RUnlock()
				// mute the session
				auth.WsNilSession(conn)
			}
			logger.Logger.Debug(fmt.Sprintf("WS error: %v", err))
			break
		}
		if err := json.Unmarshal(msg, &payload); err != nil {
			logger.Logger.Error(fmt.Sprintf("Error unmarshalling payload: %v", err))
		}
		tokenId := payload.Token.ID
		logger.Logger.Debug(fmt.Sprintf("New WS session for %v", tokenId))
		MuCon := auth.ClientManager.GetMuConn(conn, tokenId)
		token := map[string]string{
			"id":    payload.Token.ID,
			"token": payload.Token.Token,
		}
		tokenContent, authed := auth.CheckToken(token, conn, r)
		token = map[string]string{
			"id":    payload.Token.ID,
			"token": tokenContent,
		}
		var payload structs.WsPayload
		if err := json.Unmarshal(msg, &payload); err != nil {
			logger.Logger.Error(fmt.Sprintf("Error unmarshalling payload: %v", err))
			continue
		}
		var msgType structs.WsType
		err = json.Unmarshal(msg, &msgType)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Error marshalling token (else): %v", err))
			continue
		}
		ack := "ack"
		if msgType.Payload.Type == "c2c" && isC2C {
			/*
				var payload structs.C2CPayload
				if err := json.Unmarshal(msg, &payload); err != nil {
					logger.Logger.Error(fmt.Sprintf("Error unmarshalling C2C payload: %v", err))
					continue
				}
			*/
			resp, err := handler.C2CHandler(msg)
			if err != nil {
				logger.Logger.Warn(fmt.Sprintf("Unable to generate c2c payload: %v", err))
				continue
			}
			MuCon.Write(resp)
			continue
		}
		if authed || conf.Setup != "complete" {
			switch msgType.Payload.Type {
			case "penpai":
				if err = handler.PenpaiHandler(msg); err != nil {
					logger.Logger.Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
			case "new_ship":
				if err = handler.NewShipHandler(msg); err != nil {
					logger.Logger.Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
			case "pier_upload":
				if err = handler.UploadHandler(msg); err != nil {
					logger.Logger.Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
			case "password":
				if err = handler.PwHandler(msg, false); err != nil {
					logger.Logger.Error(fmt.Sprintf("%v", err))
					ack = "nack"
				} else {
					resp, err := handler.UnauthHandler()
					if err != nil {
						logger.Logger.Warn(fmt.Sprintf("Unable to generate deauth payload: %v", err))
					}
					MuCon.Write(resp)
				}
			case "system":
				if err = handler.SystemHandler(msg); err != nil {
					logger.Logger.Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
			case "startram":
				if err = handler.StartramHandler(msg); err != nil {
					logger.Logger.Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
			case "urbit":
				if err = handler.UrbitHandler(msg); err != nil {
					logger.Logger.Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
			case "support":
				if err := handler.SupportHandler(msg); err != nil {
					logger.Logger.Error(fmt.Sprintf("Error creating bug report: %v", err))
					ack = "nack"
				}
			case "broadcast":
				if err := broadcast.BroadcastToClients(); err != nil {
					logger.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
					ack = "nack"
				}
			case "login":
				// already authed so lets get you on the map
				if err := auth.AddToAuthMap(conn, token, true); err != nil {
					logger.Logger.Error(fmt.Sprintf("Unable to reauth: %v", err))
					ack = "nack"
				}
				if err := broadcast.BroadcastToClients(); err != nil {
					logger.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
					ack = "nack"
				}
			case "logout":
				if err := handler.LogoutHandler(msg); err != nil {
					logger.Logger.Error(fmt.Sprintf("Error logging out client: %v", err))
					ack = "nack"
				}
				resp, err := handler.UnauthHandler()
				if err != nil {
					logger.Logger.Warn(fmt.Sprintf("Unable to generate deauth payload: %v", err))
				}
				MuCon.Write(resp)
			case "verify":
				logger.Logger.Debug("Handling verify for auth")
				authed := true
				if conf.Setup != "complete" {
					authed = false
				}
				if err := auth.AddToAuthMap(conn, token, authed); err != nil {
					logger.Logger.Error(fmt.Sprintf("Unable to reauth: %v", err))
					ack = "nack"
				}
				if !authed && conf.Setup == "complete" {
					logger.Logger.Debug("Not authed in auth flow")
					resp, err := handler.UnauthHandler()
					if err != nil {
						logger.Logger.Warn(fmt.Sprintf("Unable to generate deauth payload: %v", err))
					}
					MuCon.Write(resp)
				}
				if err := broadcast.BroadcastToClients(); err != nil {
					logger.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
				}
			case "logs":
				var logPayload structs.WsLogsPayload
				if err := json.Unmarshal(msg, &logPayload); err != nil {
					logger.Logger.Error(fmt.Sprintf("Error unmarshalling payload: %v", err))
					continue
				}
				logEvent := structs.LogsEvent{
					Action:      logPayload.Payload.Action,
					ContainerID: logPayload.Payload.ContainerID,
					MuCon:       MuCon,
				}
				config.LogsEventBus <- logEvent
			case "setup":
				if err = setup.Setup(msg, MuCon, token); err != nil {
					logger.Logger.Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
				conf = config.Conf()
			default:
				errmsg := fmt.Sprintf("Unknown auth request type: %s", msgType.Payload.Type)
				logger.Logger.Warn(errmsg)
				if err := broadcast.BroadcastToClients(); err != nil {
					logger.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
				}
				ack = "nack"
			}
			if conf.Setup != "complete" {
				resp := structs.SetupBroadcast{
					Type:      "structure",
					AuthLevel: "setup",
					Stage:     conf.Setup,
					Page:      setup.Stages[conf.Setup],
					Regions:   startram.Regions,
				}
				respJSON, err := json.Marshal(resp)
				if err != nil {
					logger.Logger.Error(fmt.Sprintf("Couldn't marshal startram regions: %v", err))
				}
				MuCon.Write(respJSON)
			}
			// ack/nack for auth
			result := map[string]interface{}{
				"type":     "activity",
				"id":       payload.ID,
				"error":    "null",
				"response": ack,
				"token":    token,
			}
			respJson, err := json.Marshal(result)
			if err != nil {
				errmsg := fmt.Sprintf("Error marshalling token (init): %v", err)
				logger.Logger.Error(errmsg)
			}
			MuCon.Write(respJson)
			// unauthenticated action handlers
		} else {
			switch msgType.Payload.Type {
			case "login":
				newToken, err := handler.LoginHandler(MuCon, msg)
				if err != nil {
					logger.Logger.Error(fmt.Sprintf("%v", err))
					ack = "nack"
				} else {
					token = newToken
				}
				broadcast.BroadcastToClients()
			case "verify":
				logger.Logger.Info("New client")
				// auth.CreateToken also adds to unauth map
				// newToken, err := auth.CreateToken(conn, r, false)
				// if err != nil {
				// 	logger.Logger.Error(fmt.Sprintf("Unable to create token: %v", err))
				// 	ack = "nack"
				// }
				// token = newToken
				tokenContent, authed := auth.CheckToken(token, conn, r)
				token = map[string]string{
					"id":    payload.Token.ID,
					"token": tokenContent,
				}
				logger.Logger.Debug(fmt.Sprintf("Verify %v check result: %v", payload.Token.ID, authed))
			default:
				resp, err := handler.UnauthHandler()
				if err != nil {
					logger.Logger.Warn(fmt.Sprintf("Unable to generate deauth payload: %v", err))
				}
				MuCon.Write(resp)
				ack = "nack"
			}
			// ack/nack for unauth broadcast
			result := map[string]interface{}{
				"type":     "activity",
				"id":       payload.ID,
				"error":    "null",
				"response": ack,
				"token":    token,
			}
			respJson, err := json.Marshal(result)
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("Error marshalling token (init): %v", err))
			}
			MuCon.Write(respJson)
		}
	}
}
