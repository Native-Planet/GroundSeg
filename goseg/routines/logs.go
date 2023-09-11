package routines

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"goseg/config"
	"goseg/logger"
	"goseg/structs"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var (
	logsMap = make(map[*structs.MuConn]map[string]context.CancelFunc)
)

// manage log streams
func LogEvent() {
    for {
        event := <-config.LogsEventBus
		logger.Logger.Info(fmt.Sprintf("New log request for %v",event.ContainerID))
        switch event.Action {
        case true:
            logger.Logger.Info(fmt.Sprintf("Starting logs for %v", event.ContainerID))
            ctx, cancel := context.WithCancel(context.Background())
            if _, exists := logsMap[event.MuCon]; !exists {
                logsMap[event.MuCon] = make(map[string]context.CancelFunc)
            }
            logsMap[event.MuCon][event.ContainerID] = cancel
            go streamLogs(ctx, event.MuCon, event.ContainerID)
        case false:
            logger.Logger.Info(fmt.Sprintf("Stopping logs for %v", event.ContainerID))
            if cancel, exists := logsMap[event.MuCon][event.ContainerID]; exists {
                cancel()
                delete(logsMap[event.MuCon], event.ContainerID)
            }
        default:
			logger.Logger.Warn(fmt.Sprintf("Unrecognized log request for %v -- %v",event.ContainerID,event.Action))
            continue
        }
    }
}

// log bytestream to string
func extractLogMessage(data []byte) string {
	if len(data) <= 8 {
		return ""
	}
	return string(data[8:])
}

// get the last line so we know when to start streaming
func getLastLogLine(logs []byte) string {
	sanitized := removeDockerHeaders(logs)
	lines := strings.Split(string(sanitized), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if len(lines[i]) > 0 {
			return lines[i]
		}
	}
	return ""
}

func extractTimestamp(logLine string) (time.Time, error) {
	layout := "2006-01-02T15:04:05.999999999Z"
	if len(logLine) < len(layout) {
		return time.Time{}, fmt.Errorf("log line too short")
	}
	timestampStr := logLine[:len(layout)]
	return time.Parse(layout, timestampStr)
}

// sanitize chunked log history
func removeDockerHeaders(logData []byte) string {
	var cleanedLogs strings.Builder
	reader := bytes.NewReader(logData)
	bufReader := bufio.NewReader(reader) // Wrap the bytes.Reader with bufio.Reader
	for {
		header := make([]byte, 8)
		_, err := bufReader.Read(header) // Use bufReader here
		if err == io.EOF {
			break
		}
		line, err := bufReader.ReadBytes('\n') // And here
		if err != nil && err != io.EOF {
			break
		}
		cleanedLogs.WriteString(string(line))
	}
	return cleanedLogs.String()
}

func streamLogs(ctx context.Context, MuCon *structs.MuConn, containerID string) {
	if containerID != "system" {
		dockerClient, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Error streaming logs: %v", err))
			return
		}
		defer dockerClient.Close()
		// get previous logs as one chunk
		options := types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Timestamps: true,
			Tail:       "1000",
		}
		existingLogs, err := dockerClient.ContainerLogs(ctx, containerID, options)  // Use ctx instead of context.TODO()
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Error streaming previous logs: %v", err))
			return
		}
		allLogs, err := ioutil.ReadAll(existingLogs)
		existingLogs.Close()
		// send chunked log history
		if err == nil && len(allLogs) > 0 {
			sendChunkedLogs(ctx, MuCon, containerID, allLogs)
		}
		lastTimestamp, _ := extractTimestamp(getLastLogLine(allLogs))
		skipForward := time.Millisecond
		adjustedTimestamp := lastTimestamp.Add(skipForward)
		sinceTimestamp := adjustedTimestamp.Format(time.RFC3339Nano)
		options = types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Timestamps: true,
			Follow:     true,
			Since:      sinceTimestamp,
		}
		streamingLogs, err := dockerClient.ContainerLogs(ctx, containerID, options)  // Use ctx instead of context.TODO()
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Error streaming logs: %v", err))
			return
		}
		defer streamingLogs.Close()
		sendLogs(ctx, MuCon, containerID, streamingLogs, lastTimestamp)
	} else {
		
	}
}

// send a big chunk of log history
func sendChunkedLogs(ctx context.Context, MuCon *structs.MuConn, containerID string, logs []byte) {
	select {
	case <-ctx.Done():
		return
	default:
		cleanedLogs := removeDockerHeaders(logs)
		message := structs.WsLogMessage{}
		message.Log.ContainerID = containerID
		message.Log.Line = cleanedLogs
		logJSON, err := json.Marshal(message)
		if err != nil {
			logger.Logger.Warn(fmt.Sprintf("Error sending chunked logs: %v", err))
			return
		}
		if err := MuCon.Write(logJSON); err != nil {
			logger.Logger.Warn(fmt.Sprintf("Error sending chunked logs: %v", err))
			return
		}
	}
}

// send an individual log line
func sendLogs(ctx context.Context, MuCon *structs.MuConn, containerID string, logs io.Reader, lastTimestamp time.Time) {
	reader := bufio.NewReader(logs)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			line, err := reader.ReadBytes('\n')
			if err != nil && err != io.EOF {
				break
			}
			if len(line) > 0 {
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
			if err == io.EOF {
				break
			}
		}
	}
}
