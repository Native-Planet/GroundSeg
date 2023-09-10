package broadcast

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"goseg/logger"
	"goseg/structs"
	"io"
	"io/ioutil"
	"strings"
	"time"

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

// get the last line so we know when to start streaming
func getLastLogLine(logs []byte) string {
	lines := strings.Split(string(logs), "\n")
	if len(lines) > 0 {
		return lines[len(lines)-1]
	}
	return ""
}

func extractTimestamp(logLine string) (time.Time, error) {
	if len(logLine) < 19 {
		return time.Time{}, errors.New("log line too short")
	}
	layout := "2006-01-02 15:04:05"
	timestampStr := logLine[:19]
	return time.Parse(layout, timestampStr)
}

// stream logs for a given container to a ws client
// func StreamLogs(MuCon *structs.MuConn, msg []byte) {
// 	var containerID structs.WsLogsPayload
// 	if err := json.Unmarshal(msg, &containerID); err != nil {
// 		logger.Logger.Error(fmt.Sprintf("Error unmarshalling payload: %v", err))
// 		return
// 	}
// 	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
// 	if err != nil {
// 		logger.Logger.Error(fmt.Sprintf("Error streaming logs: %v", err))
// 		return
// 	}
// 	defer dockerClient.Close()
// 	options := types.ContainerLogsOptions{ShowStdout: true, Tail: "all"}
// 	existingLogs, err := dockerClient.ContainerLogs(context.TODO(), containerID, options)
// 	if err != nil {
// 		logger.Logger.Error(fmt.Sprintf("Error streaming previous logs: %v", err))
// 		return
// 	}
// 	sendLogs(conn, containerID, existingLogs)
// 	existingLogs.Close()

// 	options := types.ContainerLogsOptions{ShowStdout: true, Follow: true}
// 	logs, err := dockerClient.ContainerLogs(context.TODO(), containerID.Payload.ContainerID, options)
// 	if err != nil {
// 		logger.Logger.Error(fmt.Sprintf("Error streaming logs: %v", err))
// 		return
// 	}
// 	defer logs.Close()
// 	reader := bufio.NewReader(logs)
// 	for {
// 		line, err := reader.ReadBytes('\n')
// 		if err != nil && err != io.EOF {
// 			break
// 		}
// 		if len(line) > 0 {
// 			logString := extractLogMessage(line)
// 			message := structs.WsLogMessage{}
// 			message.Log.ContainerID = containerID.Payload.ContainerID
// 			message.Log.Line = logString
// 			logJSON, err := json.Marshal(message)
// 			if err != nil {
// 				logger.Logger.Warn(fmt.Sprintf("Error streaming logs: %v", err))
// 				break
// 			}
// 			if err := MuCon.Write(logJSON); err != nil {
// 				logger.Logger.Warn(fmt.Sprintf("Error streaming logs: %v", err))
// 				break
// 			}
// 		}
// 		if err == io.EOF {
// 			break
// 		}
// 	}
// }

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
	// get previous logs as one chunk
	options := types.ContainerLogsOptions{ShowStdout: true, Tail: "1000"}
	existingLogs, err := dockerClient.ContainerLogs(context.TODO(), containerID.Payload.ContainerID, options)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Error streaming previous logs: %v", err))
		return
	}
	allLogs, err := ioutil.ReadAll(existingLogs)
	existingLogs.Close()
	lastTimestamp, _ := extractTimestamp(getLastLogLine(allLogs))
	if err == nil && len(allLogs) > 0 {
		sendChunkedLogs(MuCon, containerID.Payload.ContainerID, allLogs)
	}
	// stream logs line-by-line (ongoing)
	options = types.ContainerLogsOptions{ShowStdout: true, Follow: true}
	streamingLogs, err := dockerClient.ContainerLogs(context.TODO(), containerID.Payload.ContainerID, options)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Error streaming logs: %v", err))
		return
	}
	defer streamingLogs.Close()
	sendLogs(MuCon, containerID.Payload.ContainerID, streamingLogs, lastTimestamp)
}

func sendChunkedLogs(MuCon *structs.MuConn, containerID string, logs []byte) {
	logString := extractLogMessage(logs)
	message := structs.WsLogMessage{}
	message.Log.ContainerID = containerID
	message.Log.Line = logString
	logJSON, err := json.Marshal(message)
	if err != nil {
		logger.Logger.Warn(fmt.Sprintf("Error sending chunked logs: %v", err))
		return
	}
	if err := MuCon.Write(logJSON); err != nil {
		logger.Logger.Warn(fmt.Sprintf("Error sending chunked logs: %v", err))
	}
}

func sendLogs(MuCon *structs.MuConn, containerID string, logs io.Reader, lastTimestamp time.Time) {
	reader := bufio.NewReader(logs)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			break
		}
		if len(line) > 0 {
			logTimestamp, err := extractTimestamp(string(line))
			if err == nil && logTimestamp.After(lastTimestamp) {
				logString := extractLogMessage(line)
				message := structs.WsLogMessage{}
				message.Log.ContainerID = containerID
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
		}
		if err == io.EOF {
			break
		}
	}
}