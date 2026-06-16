package ws

import (
	"bufio"
	"encoding/json"
	"fmt"
	"groundseg/auth"
	"groundseg/docker"
	"groundseg/dockerclient"
	"groundseg/logger"
	"groundseg/structs"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type LogPayload struct {
	Type  string                `json:"type"`
	Token structs.WsTokenStruct `json:"token"`
}

func LogsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		setLogStreamCORS(w, r)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if strings.Contains(r.Header.Get("Accept"), "text/event-stream") || r.Method == http.MethodPost {
		LogsStreamHandler(w, r)
		return
	}

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
			conn.Close()
			break
		}
		// message type is text
		if messageType != 1 {
			zap.L().Error("log socket invalid message type")
			conn.Close()
			break
		}

		// manage payload
		var payload LogPayload
		if err := json.Unmarshal([]byte(p), &payload); err != nil {
			zap.L().Error(fmt.Sprintf("unmarshal log request error: %v", err))
			conn.Close()
			break
		}

		// verify session is authenticated
		if authed := auth.LogTokenCheck(payload.Token, r); !authed {
			zap.L().Info("log request not unauthenticated")
			conn.Close()
			break
		}
		if payload.Type == "system" {
			logHistory, err := logger.RetrieveSysLogHistory()
			if err != nil {
				zap.L().Error(fmt.Sprintf("failed to retrieve log history: %v", err))
				conn.Close()
				break
			}
			if err := conn.WriteMessage(1, logHistory); err != nil {
				zap.L().Error(fmt.Sprintf("error writing message: %v", err))
				conn.Close()
				break
			}
			zap.L().Info("log request authenticated")
			logger.SysLogSessions = append(logger.SysLogSessions, conn)
		} else {
			_, err := docker.FindContainer(payload.Type)
			if err != nil {
				zap.L().Error(fmt.Sprintf("log retrieval failed: %v", err))
				conn.Close()
				break
			}
			if _, exists := logger.DockerLogSessions[payload.Type]; !exists {
				logger.DockerLogSessions[payload.Type] = make(map[*websocket.Conn]bool)
			}
			logger.DockerLogSessions[payload.Type][conn] = false
		}
	}
}

func LogsStreamHandler(w http.ResponseWriter, r *http.Request) {
	setLogStreamCORS(w, r)
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var payload LogPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid log request", http.StatusBadRequest)
		return
	}

	if authed := auth.LogTokenCheck(payload.Token, r); !authed {
		zap.L().Info("log stream request unauthenticated")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	if err := writeLogStreamComment(w, "connected"); err != nil {
		return
	}
	flusher.Flush()

	if payload.Type == "system" {
		logHistory, logPath, logOffset, err := logger.RetrieveSysLogHistoryWithOffset()
		if err != nil {
			zap.L().Error(fmt.Sprintf("failed to retrieve log history: %v", err))
			return
		}
		if err := writeLogStreamEvent(w, logHistory); err != nil {
			return
		}
		flusher.Flush()
		streamSystemLogs(w, flusher, r, logPath, logOffset)
		return
	}

	if _, err := docker.FindContainer(payload.Type); err != nil {
		zap.L().Error(fmt.Sprintf("log stream retrieval failed: %v", err))
		return
	}
	streamDockerLogs(w, flusher, r, payload.Type)
}

func streamSystemLogs(w http.ResponseWriter, flusher http.Flusher, r *http.Request, filePath string, offset int64) {
	if filePath == "" {
		filePath = logger.SysLogfile()
	}
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			waitForSystemLogFile(w, flusher, r, filePath, offset)
			return
		}
		zap.L().Error(fmt.Sprintf("failed to open system log file %s: %v", filePath, err))
		return
	}
	defer func() {
		if file != nil {
			file.Close()
		}
	}()

	if info, err := file.Stat(); err == nil && offset > info.Size() {
		offset = 0
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		zap.L().Error(fmt.Sprintf("failed to seek system log file %s: %v", filePath, err))
		return
	}

	reader := bufio.NewReader(file)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			for {
				line, err := reader.ReadString('\n')
				if line != "" {
					if err := writeSystemLogLine(w, flusher, line); err != nil {
						return
					}
				}
				if err == nil {
					continue
				}
				if err != io.EOF {
					zap.L().Error(fmt.Sprintf("error reading system log file %s: %v", filePath, err))
					return
				}
				nextPath := logger.SysLogfile()
				if nextPath != filePath {
					if nextFile, openErr := os.Open(nextPath); openErr == nil {
						file.Close()
						file = nextFile
						filePath = nextPath
						reader = bufio.NewReader(file)
					}
				}
				break
			}
		}
	}
}

func waitForSystemLogFile(w http.ResponseWriter, flusher http.Flusher, r *http.Request, filePath string, offset int64) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if _, err := os.Stat(filePath); err == nil {
				streamSystemLogs(w, flusher, r, filePath, offset)
				return
			}
			filePath = logger.SysLogfile()
		}
	}
}

func writeSystemLogLine(w http.ResponseWriter, flusher http.Flusher, line string) error {
	line = strings.TrimRight(line, "\r\n")
	if strings.TrimSpace(line) == "" {
		return nil
	}
	logJSON := fmt.Appendf(nil, `{"type":"system","history":false,"log":%s}`, line)
	if err := writeLogStreamEvent(w, logJSON); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func streamDockerLogs(w http.ResponseWriter, flusher http.Flusher, r *http.Request, containerName string) {
	cli, err := dockerclient.New()
	if err != nil {
		zap.L().Error(fmt.Sprintf("failed to create Docker client: %v", err))
		return
	}
	defer cli.Close()

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
	}
	out, err := cli.ContainerLogs(r.Context(), containerName, options)
	if err != nil {
		zap.L().Error(fmt.Sprintf("failed to get logs for container %s: %v", containerName, err))
		return
	}
	defer out.Close()

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		line := ""
		if len(scanner.Text()) > 8 {
			line = scanner.Text()[8:]
		}
		logJSON, err := json.Marshal(map[string]any{
			"type":    containerName,
			"history": false,
			"log":     line,
		})
		if err != nil {
			zap.L().Error(fmt.Sprintf("failed to marshal log line for %s: %v", containerName, err))
			return
		}
		if err := writeLogStreamEvent(w, logJSON); err != nil {
			return
		}
		flusher.Flush()
	}
	if err := scanner.Err(); err != nil && r.Context().Err() == nil {
		zap.L().Error(fmt.Sprintf("error reading logs for %s: %v", containerName, err))
	}
}

func setLogStreamCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Vary", "Origin")
}

func writeLogStreamEvent(w http.ResponseWriter, data []byte) error {
	if _, err := fmt.Fprint(w, "event: log\n"); err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, "\n")
	return err
}

func writeLogStreamComment(w http.ResponseWriter, text string) error {
	_, err := fmt.Fprintf(w, ": %s\n\n", text)
	return err
}
