package broadcast

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"goseg/logger"
	"goseg/structs"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// log bytestream to string
func extractLogMessage(data []byte) string {
	if len(data) <= 8 {
		return ""
	}
	return string(data[8:])
}

// stream logs for a given container to a ws client
func StreamLogs(MuCon *structs.MuConn, msg []byte) {
	var containerID structs.WsLogsPayload
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
	reader := bufio.NewReader(logs)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			break
		}
		if len(line) > 0 {
			logString := extractLogMessage(line)
			message := structs.WsLogMessage{}
			message.Log.ContainerID = containerID.Payload.ContainerID
			message.Log.Line = logString
			logJSON, err := json.Marshal(message)
			if err != nil {
				logger.Logger.Warn(fmt.Sprintf("Error streaming logs: %v", err))
				break
			}
			if err := MuCon.Write(logJSON); err != nil {
				logger.Logger.Warn(fmt.Sprintf("Error streaming logs: %v", err))
				break
			}
		}
		if err == io.EOF {
			break
		}
	}
}
