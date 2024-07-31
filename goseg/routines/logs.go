package routines

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"groundseg/logger"
	"groundseg/structs"
	"time"

	// "io/ioutil"

	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type DockerCancel struct {
	Container string
	Conn      *websocket.Conn
}

var (
	// zap
	logsMap                = make(map[*structs.MuConn]map[string]*structs.CtxWithCancel)
	dockerLogCancelChannel = make(chan DockerCancel, 100)
	wsLogMessagePool       = sync.Pool{
		New: func() interface{} {
			return new(structs.WsLogMessage)
		},
	}
)

// zap
func SysLogStreamer() {
	for {
		logger.RemoveSysSessions()
		log, _ := <-logger.SysLogChannel

		// cleanup log string
		var buffer bytes.Buffer
		err := json.Compact(&buffer, log)
		if err != nil {
			continue
		}
		escapedLog := buffer.Bytes()
		logJSON := []byte(fmt.Sprintf(`{"type":"system","history":false,"log":%s}`, escapedLog))
		if err != nil {
			continue
		}
		for _, conn := range logger.SysLogSessions {
			if err := conn.WriteMessage(1, logJSON); err != nil {
				zap.L().Error(fmt.Sprintf("error writing message: %v", err))
				conn.Close()
				logger.SysSessionsToRemove = append(logger.SysSessionsToRemove, conn)
			}
		}
	}
}

func DockerLogStreamer() {
	for {
		for container, sessionMap := range logger.DockerLogSessions {
			for conn, live := range sessionMap {
				if !live {
					go streamToConn(container, conn)
					logger.DockerLogSessions[container][conn] = true
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func DockerLogConnRemover() {
	for {
		c, _ := <-dockerLogCancelChannel
		if _, exists := logger.DockerLogSessions[c.Container]; exists {
			delete(logger.DockerLogSessions[c.Container], c.Conn)
			if len(logger.DockerLogSessions[c.Container]) == 0 {
				delete(logger.DockerLogSessions, c.Container)
			}
		}
	}
}

func streamToConn(container string, conn *websocket.Conn) {
	defer func() {
		dockerLogCancelChannel <- DockerCancel{Container: container, Conn: conn}
	}()
	defer conn.Close()
	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		zap.L().Error(fmt.Sprintf("failed to create Docker client: %w", err))
		return
	}
	defer cli.Close()

	// Set up the context
	ctx := context.Background()

	// Specify options to stream logs
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true, // Stream the logs
		Timestamps: true,
	}

	// Get the container logs as a stream
	out, err := cli.ContainerLogs(ctx, container, options)
	if err != nil {
		zap.L().Error(fmt.Sprintf("failed to get logs for container %s: %w", container, err))
		return
	}
	defer out.Close()

	// Read and print logs line by line
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		line := ""
		if len(scanner.Text()) > 8 {
			line = scanner.Text()[8:]
		}
		logJSON := []byte(fmt.Sprintf(`{"type":"%s","history":false,"log":"%s"}`, container, line))
		if err := conn.WriteMessage(1, logJSON); err != nil {
			zap.L().Error(fmt.Sprintf("error writing message for %v: %v", container, err))
			return
		}
	}
	if err := scanner.Err(); err != nil {
		zap.L().Error(fmt.Sprintf("error reading logs: %w", err))
		return
	}
}
