package routines

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"groundseg/config"
	"groundseg/logger"
	"groundseg/structs"
	"io"

	// "io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

var (
	// zap
	logsMap          = make(map[*structs.MuConn]map[string]*structs.CtxWithCancel)
	wsLogMessagePool = sync.Pool{
		New: func() interface{} {
			return new(structs.WsLogMessage)
		},
	}
)

func FakeLogs() {
	for {
		zap.L().Info("fake log")
		time.Sleep(10 * time.Second)
	}
}

// zap
func SysLogStreamer() {
	sys := "system"
	for {
		logger.RemoveSessions(sys)
		log, _ := <-logger.SysLogChannel
		sessions, exists := logger.LogSessions[sys]
		if !exists {
			continue
		}
		// cleanup log string
		var buffer bytes.Buffer
		err := json.Compact(&buffer, log)
		if err != nil {
			continue
		}
		escapedLog := buffer.Bytes()
		logJSON := []byte(fmt.Sprintf(`{"type":"system","log":%s}`, escapedLog))
		if err != nil {
			continue
		}
		for _, conn := range sessions {
			if err := conn.WriteMessage(1, logJSON); err != nil {
				zap.L().Error(fmt.Sprintf("error writing message: %v", err))
				conn.Close()
				logger.SessionsToRemove[sys] = append(logger.SessionsToRemove[sys], conn)
			}
		}
	}
}

func cleanupLogsMap() {
	for MuCon, conMap := range logsMap {
		if len(conMap) == 0 {
			delete(logsMap, MuCon)
		} else {
			for containerID, ctxCancel := range conMap {
				select {
				case <-ctxCancel.Ctx.Done():
					delete(conMap, containerID)
				default:
					// context is not done yet
				}
			}
			if len(conMap) == 0 {
				delete(logsMap, MuCon)
			}
		}
	}
}

// manage log streams
func LogEvent() {
	for {
		event := <-config.LogsEventBus
		zap.L().Debug(fmt.Sprintf("Log action %v", event.Action))
		switch event.Action {
		case true:
			zap.L().Info(fmt.Sprintf("Starting logs for %v", event.ContainerID))
			ctx, cancel := context.WithCancel(context.Background())
			if _, exists := logsMap[event.MuCon]; !exists {
				logsMap[event.MuCon] = make(map[string]*structs.CtxWithCancel)
			}
			logsMap[event.MuCon][event.ContainerID] = &structs.CtxWithCancel{
				Ctx:    ctx,
				Cancel: cancel,
			}
			go streamLogs(ctx, event.MuCon, event.ContainerID)
		// cancel all log streams on ws break
		case false:
			if event.ContainerID == "all" {
				zap.L().Debug(fmt.Sprintf("Cancelling log stream for ws %v", event.ContainerID))
				if conMap, exists := logsMap[event.MuCon]; exists {
					for container, cancel := range conMap {
						cancel.Cancel()
						delete(logsMap[event.MuCon], container)
						cleanupLogsMap()
					}
				}
			} else {
				zap.L().Debug(fmt.Sprintf("Cancelling log stream for ws %v", event.ContainerID))
				if cancel, exists := logsMap[event.MuCon][event.ContainerID]; exists {
					cancel.Cancel()
					delete(logsMap[event.MuCon], event.ContainerID)
				}
			}
		default:
			zap.L().Warn(fmt.Sprintf("Unrecognized log request for %v -- %v", event.ContainerID, event.Action))
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
	bufReader := bufio.NewReader(reader)
	for {
		header := make([]byte, 8)
		_, err := bufReader.Read(header)
		if err == io.EOF {
			break
		}
		line, err := bufReader.ReadBytes('\n')
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
			zap.L().Error(fmt.Sprintf("Error streaming logs: %v", err))
			return
		}
		defer dockerClient.Close()
		options := types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Timestamps: true,
			Tail:       "1000",
		}
		existingLogs, err := dockerClient.ContainerLogs(ctx, containerID, options)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Error streaming previous logs: %v", err))
			return
		}
		defer existingLogs.Close()
		const logChunkSize = 4096
		buf := make([]byte, logChunkSize)
		for {
			n, err := existingLogs.Read(buf)
			if n > 0 {
				sendChunkedLogs(ctx, MuCon, containerID, buf[:n])
			}
			if err != nil {
				if err != io.EOF {
					zap.L().Error(fmt.Sprintf("Error reading log chunk: %v", err))
				}
				break
			}
		}
		lastTimestamp, _ := extractTimestamp(getLastLogLine(buf))
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
		streamingLogs, err := dockerClient.ContainerLogs(ctx, containerID, options)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Error streaming logs: %v", err))
			return
		}
		defer streamingLogs.Close()
		sendLogs(ctx, MuCon, containerID, streamingLogs, lastTimestamp)

	} else {
		err := tailLogs(ctx, MuCon, logger.SysLogfile())
		if err != nil {
			zap.L().Error(fmt.Sprintf("Error streaming system logs: %v", err))
		}
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
		message.Type = "log"
		logJSON, err := json.Marshal(message)
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Error sending chunked logs: %v", err))
			return
		}
		MuCon.Write(logJSON)
	}
}

// tail the syslog file then stream new changes
func tailLogs(ctx context.Context, MuCon *structs.MuConn, filename string) error {
	sendChunkedSysLogs(ctx, MuCon)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	err = watcher.Add(filename)
	if err != nil {
		return err
	}
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	file.Seek(0, os.SEEK_END)
	scanner := bufio.NewScanner(file)
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				for scanner.Scan() {
					sendSysLogs(ctx, MuCon, scanner.Text())
				}
				if err := scanner.Err(); err != nil {
					zap.L().Warn(fmt.Sprintf("Syslog scan error: %v", err))
				}
			}
		case err := <-watcher.Errors:
			return err
		}
	}
}

// send prev 500 lines of syslogs
func sendChunkedSysLogs(ctx context.Context, MuCon *structs.MuConn) {
	select {
	case <-ctx.Done():
		return
	default:
		message := structs.WsLogMessage{}
		message.Log.ContainerID = "system"
		message.Type = "log"
		logLines, err := logger.TailLogs(logger.SysLogfile(), 500)
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Error tailing logs: %v", err))
			return
		}
		message.Log.Line = strings.Join(logLines, "\n")
		logJSON, err := json.Marshal(message)
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Error sending chunked logs: %v", err))
			return
		}
		MuCon.Write(logJSON)
	}
}

// send an individual container log line
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
				message := wsLogMessagePool.Get().(*structs.WsLogMessage)
				message.Log.ContainerID = containerID
				message.Log.Line = logString
				message.Type = "log"
				logJSON, err := json.Marshal(message)
				if err != nil {
					zap.L().Warn(fmt.Sprintf("Error streaming logs: %v", err))
					break
				}
				MuCon.Write(logJSON)
				wsLogMessagePool.Put(message) // return message to the pool
			}
			if err == io.EOF {
				break
			}
		}
	}
}

// send individual system log line
func sendSysLogs(ctx context.Context, MuCon *structs.MuConn, line string) {
	select {
	case <-ctx.Done():
		return
	default:
		if len(line) > 0 {
			message := structs.WsLogMessage{}
			message.Log.ContainerID = "system"
			message.Log.Line = line
			message.Type = "log"
			logJSON, err := json.Marshal(message)
			if err != nil {
				zap.L().Warn(fmt.Sprintf("Error streaming logs: %v", err))
				return
			}
			MuCon.Write(logJSON)
		}
	}
}
