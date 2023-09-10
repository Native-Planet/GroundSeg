package broadcast

import (
	"context"
	"encoding/json"
	"fmt"
	"goseg/logger"
	"goseg/structs"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// stream logs for a given container to a ws client
func StreamLogs(MuCon *structs.MuConn, msg []byte) {
	var containerID  structs.WsLogsPayload
	if err := json.Unmarshal(msg, &containerID); err != nil {
		logger.Logger.Error(fmt.Sprintf("Error unmarshalling payload: %v", err))
		return
	}
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Error streaming logs: %v", err))
		return
	}
	defer dockerClient.Close()

	options := types.ContainerLogsOptions{ShowStdout: true, Follow: true}
	logs, err := dockerClient.ContainerLogs(context.TODO(), containerID.Payload.ContainerID, options)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Error streaming logs: %v", err))
		return
	}
	defer logs.Close()

	buf := make([]byte, 1024)
	for {
		n, err := logs.Read(buf)
		if err != nil && err != io.EOF {
			break
		}
		if n > 0 {
			// conn.WriteMessage(websocket.TextMessage, buf[:n])
			if err := MuCon.Write(buf[:n]); err != nil {
				logger.Logger.Warn(fmt.Sprintf("Error streaming logs: %v", err))
				break
			}
		}
	}
}
