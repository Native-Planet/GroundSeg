package ws

import (
	"encoding/json"
	"fmt"
	"groundseg/auth"
	"groundseg/broadcast"
	"groundseg/config"
	"groundseg/handler"
	"groundseg/setup"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/system"
	"net/http"
	"strings"

	// "time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
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
	zap.L().Debug("New ws session")
	if err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't upgrade websocket connection: %v", err))
		return
	}
	// initial handling before we assign ws session to mutex
	_, msg, err := conn.ReadMessage()
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) || strings.Contains(err.Error(), "broken pipe") {
			zap.L().Debug("WS closed")
			conn.Close()
			// mute the session
			auth.WsNilSession(conn)
		}
		zap.L().Debug(fmt.Sprintf("WS error: %v", err))
	}
	// get the token from payload and check vs auth map
	var payload structs.WsPayload
	if err := json.Unmarshal(msg, &payload); err != nil {
		zap.L().Error(fmt.Sprintf("Error unmarshalling payload: %v", err))
	}
	tokenId := payload.Token.ID
	zap.L().Debug(fmt.Sprintf("New WS session for %v", tokenId))
	MuCon := auth.ClientManager.GetMuConn(conn, tokenId)
	token := map[string]string{
		"id":    payload.Token.ID,
		"token": payload.Token.Token,
	}
	tokenContent, authed := auth.CheckToken(token, r)
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
		zap.L().Error(errmsg)
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
			zap.L().Error(fmt.Sprintf("Couldn't marshal startram regions: %v", err))
		}
		MuCon.Write(respJSON)
	} else if !authed {
		var ack string
		token, err = auth.CreateToken(conn, r, false)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Unable to create token: %v", err))
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
			zap.L().Error(errmsg)
		}
		MuCon.Write(respJson)
	}
	if err := auth.AddToAuthMap(conn, token, authed); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to track auth session: %v", err))
	}
	for {
		// mutexed read operations
		_, msg, err := MuCon.Read(auth.ClientManager)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) || strings.Contains(err.Error(), "broken pipe") {
				zap.L().Debug("WS closed")
				conn.Close()
				// mute the session
				auth.WsNilSession(conn)
			}
			zap.L().Debug(fmt.Sprintf("WS error: %v", err))
			break
		}
		if err := json.Unmarshal(msg, &payload); err != nil {
			zap.L().Error(fmt.Sprintf("Error unmarshalling payload: %v", err))
		}
		tokenId := payload.Token.ID
		zap.L().Debug(fmt.Sprintf("New WS session for %v", tokenId))
		MuCon := auth.ClientManager.GetMuConn(conn, tokenId)
		token := map[string]string{
			"id":    payload.Token.ID,
			"token": payload.Token.Token,
		}
		tokenContent, authed := auth.CheckToken(token, r)
		token = map[string]string{
			"id":    payload.Token.ID,
			"token": tokenContent,
		}
		var payload structs.WsPayload
		if err := json.Unmarshal(msg, &payload); err != nil {
			zap.L().Error(fmt.Sprintf("Error unmarshalling payload: %v", err))
			continue
		}
		var msgType structs.WsType
		err = json.Unmarshal(msg, &msgType)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Error marshalling token (else): %v", err))
			continue
		}
		ack := "ack"
		if msgType.Payload.Type == "c2c" && isC2C {
			/*
				var payload structs.C2CPayload
				if err := json.Unmarshal(msg, &payload); err != nil {
					zap.L().Error(fmt.Sprintf("Error unmarshalling C2C payload: %v", err))
					continue
				}
			*/
			resp, err := handler.C2CHandler(msg)
			if err != nil {
				zap.L().Warn(fmt.Sprintf("Unable to generate c2c payload: %v", err))
				continue
			}
			MuCon.Write(resp)
			continue
		}
		if authed || conf.Setup != "complete" {
			switch msgType.Payload.Type {
			case "dev":
				if err = handler.DevHandler(msg); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
			case "penpai":
				if err = handler.PenpaiHandler(msg); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
			case "new_ship":
				if err = handler.NewShipHandler(msg); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
			case "pier_upload":
				if err = handler.UploadHandler(msg); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
			case "password":
				if err = handler.PwHandler(msg, false); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					ack = "nack"
				} else {
					resp, err := handler.UnauthHandler()
					if err != nil {
						zap.L().Warn(fmt.Sprintf("Unable to generate deauth payload: %v", err))
					}
					MuCon.Write(resp)
				}
			case "system":
				if err = handler.SystemHandler(msg); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
			case "startram":
				if err = handler.StartramHandler(msg); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
			case "urbit":
				if err = handler.UrbitHandler(msg); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
			case "support":
				if err := handler.SupportHandler(msg); err != nil {
					zap.L().Error(fmt.Sprintf("Error creating bug report: %v", err))
					ack = "nack"
				}
			case "broadcast":
				if err := broadcast.BroadcastToClients(); err != nil {
					zap.L().Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
					ack = "nack"
				}
			case "login":
				// already authed so lets get you on the map
				if err := auth.AddToAuthMap(conn, token, true); err != nil {
					zap.L().Error(fmt.Sprintf("Unable to reauth: %v", err))
					ack = "nack"
				}
				if err := broadcast.BroadcastToClients(); err != nil {
					zap.L().Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
					ack = "nack"
				}
			case "logout":
				if err := handler.LogoutHandler(msg); err != nil {
					zap.L().Error(fmt.Sprintf("Error logging out client: %v", err))
					ack = "nack"
				}
				resp, err := handler.UnauthHandler()
				if err != nil {
					zap.L().Warn(fmt.Sprintf("Unable to generate deauth payload: %v", err))
				}
				MuCon.Write(resp)
			case "verify":
				zap.L().Debug("Handling verify for auth")
				authed := true
				if conf.Setup != "complete" {
					authed = false
				}
				if err := auth.AddToAuthMap(conn, token, authed); err != nil {
					zap.L().Error(fmt.Sprintf("Unable to reauth: %v", err))
					ack = "nack"
				}
				if !authed && conf.Setup == "complete" {
					zap.L().Debug("Not authed in auth flow")
					resp, err := handler.UnauthHandler()
					if err != nil {
						zap.L().Warn(fmt.Sprintf("Unable to generate deauth payload: %v", err))
					}
					MuCon.Write(resp)
				}
				if err := broadcast.BroadcastToClients(); err != nil {
					zap.L().Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
				}
			case "setup":
				if err = setup.Setup(msg, MuCon, token); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
				conf = config.Conf()
			default:
				errmsg := fmt.Sprintf("Unknown auth request type: %s", msgType.Payload.Type)
				zap.L().Warn(errmsg)
				if err := broadcast.BroadcastToClients(); err != nil {
					zap.L().Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
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
					zap.L().Error(fmt.Sprintf("Couldn't marshal startram regions: %v", err))
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
				zap.L().Error(errmsg)
			}
			MuCon.Write(respJson)
			// unauthenticated action handlers
		} else {
			switch msgType.Payload.Type {
			case "login":
				newToken, err := handler.LoginHandler(MuCon, msg)
				if err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					ack = "nack"
				} else {
					token = newToken
				}
				broadcast.BroadcastToClients()
			case "verify":
				zap.L().Info("New client")
				// auth.CreateToken also adds to unauth map
				// newToken, err := auth.CreateToken(conn, r, false)
				// if err != nil {
				// 	zap.L().Error(fmt.Sprintf("Unable to create token: %v", err))
				// 	ack = "nack"
				// }
				// token = newToken
				tokenContent, authed := auth.CheckToken(token, r)
				token = map[string]string{
					"id":    payload.Token.ID,
					"token": tokenContent,
				}
				zap.L().Debug(fmt.Sprintf("Verify %v check result: %v", payload.Token.ID, authed))
			default:
				resp, err := handler.UnauthHandler()
				if err != nil {
					zap.L().Warn(fmt.Sprintf("Unable to generate deauth payload: %v", err))
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
				zap.L().Error(fmt.Sprintf("Error marshalling token (init): %v", err))
			}
			MuCon.Write(respJson)
		}
	}
}
