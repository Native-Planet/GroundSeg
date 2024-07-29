package ws

import (
	"encoding/json"
	"fmt"
	"groundseg/auth"
	"groundseg/logger"
	"groundseg/structs"
	"net/http"

	"go.uber.org/zap"
)

type LogPayload struct {
	Type  string                `json:"type"`
	Token structs.WsTokenStruct `json:"token"`
}

func LogsHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not upgrade to websocket", http.StatusInternalServerError)
		return
	}

	// Handle the WebSocket connection
	for {
		// Read message from WebSocket
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			zap.L().Error(fmt.Sprintf("log socket error: %v", err))
			break
		}
		// message type is text
		if messageType != 1 {
			zap.L().Error("log socket invalid message type")
		}

		// manage payload
		var payload LogPayload
		if err := json.Unmarshal([]byte(p), &payload); err != nil {
			zap.L().Error(fmt.Sprintf("unmarshal log request error: %v", err))
			break
		}

		// verify session is authenticated
		if authed := auth.LogTokenCheck(payload.Token, r); !authed {
			zap.L().Info("log request not unauthenticated")
			break
		}
		zap.L().Info("log request authenticated")
		logger.LogSessions[payload.Type] = append(logger.LogSessions[payload.Type], conn)
	}
}
