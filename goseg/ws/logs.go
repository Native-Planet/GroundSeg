package ws

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

func LogsHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not upgrade to websocket", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Handle the WebSocket connection
	for {
		// Read message from WebSocket
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("unexpected close error: %v\n", err)
			}
			break
		}

		// Print the message received from the client
		fmt.Printf("Received: %s\n", p)

		// Echo the message back to the client
		if err := conn.WriteMessage(messageType, p); err != nil {
			fmt.Printf("error writing message: %v\n", err)
			break
		}
	}
}
