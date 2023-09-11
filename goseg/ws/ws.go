package ws

// you can pass websockets to other packages for reads, but please
// try to do all writes from here
// otherwise you have to deal with passing mutexes which is annoying
// and hard to think about

import (
	"encoding/json"
	"fmt"
	"goseg/auth"
	"goseg/broadcast"
	"goseg/config"
	"goseg/handler"
	"goseg/logger"
	"goseg/structs"
	"net/http"

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
	conf := config.Conf()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Couldn't upgrade websocket connection: %v", err))
		return
	}
	tokenId := config.RandString(32)
	MuCon := auth.ClientManager.NewConnection(conn, tokenId)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				logger.Logger.Info("WS closed")
				conn.Close()
			}
			logger.Logger.Error(fmt.Sprintf("Error reading websocket message: %v", err))
			break
		}
		var payload structs.WsPayload
		if err := json.Unmarshal(msg, &payload); err != nil {
			logger.Logger.Warn(fmt.Sprintf("Error unmarshalling payload: %v", err))
			continue
		}
		var msgType structs.WsType
		err = json.Unmarshal(msg, &msgType)
		if err != nil {
			logger.Logger.Warn(fmt.Sprintf("Error marshalling token (else): %v", err))
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
		ack := "ack"
		if authed {
			switch msgType.Payload.Type {
			case "new_ship":
				if err = handler.NewShipHandler(msg); err != nil {
					logger.Logger.Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
			case "pier_upload":
				logger.Logger.Info("Pier upload")
			case "password":
				if err = handler.PwHandler(msg); err != nil {
					logger.Logger.Error(fmt.Sprintf("%v", err))
					ack = "nack"
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
				if err = handler.SupportHandler(msg, payload, r, conn); err != nil {
					logger.Logger.Error(fmt.Sprintf("%v", err))
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
					logger.Logger.Warn(fmt.Sprintf("Unable to generate deauth payload:", err))
				}
				if err := MuCon.Write(resp); err != nil {
					logger.Logger.Warn("Unable to broadcast to unauth client")
				}
			case "verify":
				if err := auth.AddToAuthMap(conn, token, true); err != nil {
					logger.Logger.Error(fmt.Sprintf("Unable to reauth: %v", err))
					ack = "nack"
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
					Action: logPayload.Payload.Action,
					ContainerID: logPayload.Payload.ContainerID,
					MuCon: MuCon,
				}
				broadcast.LogsEventBus<-logEvent
				// broadcast.StreamLogs(MuCon, msg)
			default:
				errmsg := fmt.Sprintf("Unknown auth request type: %s", msgType.Payload.Type)
				logger.Logger.Warn(errmsg)
				if err := broadcast.BroadcastToClients(); err != nil {
					logger.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
				}
				ack = "nack"
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
			if err := MuCon.Write(respJson); err != nil {
				continue
			}
			// unauthenticated action handlers
		} else {
			switch msgType.Payload.Type {
			case "login":
				if err = handler.LoginHandler(MuCon, msg); err != nil {
					logger.Logger.Error(fmt.Sprintf("%v", err))
					ack = "nack"
				}
				broadcast.BroadcastToClients()
			case "setup":
				logger.Logger.Info("Setup")
				// setup.Setup(payload)
			case "verify":
				logger.Logger.Info("New client")
				// auth.CreateToken also adds to unauth map
				newToken, err := auth.CreateToken(conn, r, false)
				if err != nil {
					logger.Logger.Error(fmt.Sprintf("Unable to create token: %v", err))
					ack = "nack"
				}
				result := map[string]interface{}{
					"type":     "activity",
					"id":       payload.ID,
					"error":    "null",
					"response": ack,
					"token":    newToken,
				}
				respJson, err := json.Marshal(result)
				if err != nil {
					logger.Logger.Error(fmt.Sprintf("Error marshalling token (init): %v", err))
					ack = "nack"
				}
				if err := MuCon.Write(respJson); err != nil {
					continue
				}
			case "logs":
				// unauth for debugging
				var logPayload structs.WsLogsPayload
				if err := json.Unmarshal(msg, &logPayload); err != nil {
					logger.Logger.Error(fmt.Sprintf("Error unmarshalling payload: %v", err))
					continue
				}
				logEvent := structs.LogsEvent{
					Action: logPayload.Payload.Action,
					ContainerID: logPayload.Payload.ContainerID,
					MuCon: MuCon,
				}
				broadcast.LogsEventBus<-logEvent
			default:
				resp, err := handler.UnauthHandler()
				if err != nil {
					logger.Logger.Warn(fmt.Sprintf("Unable to generate deauth payload:", err))
				}
				if err := MuCon.Write(resp); err != nil {
					logger.Logger.Warn("Unable to broadcast to unauth client")
				}
				ack = "nack"
			}
		}
		// ack/nack for unauth
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
		if err := MuCon.Write(respJson); err != nil {
			continue
		}
	}
}
