package logstream

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"groundseg/dockerclient"
	"groundseg/logger"
	"groundseg/structs"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	// "io/ioutil"

	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type DockerCancel struct {
	Container string
	Conn      *websocket.Conn
}

const (
	MaxChunkSize = 10 * 1024 * 1024 // 10MB
)

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

func OldLogsCleaner() {
	_ = OldLogsCleanerWithContext(context.Background())
}

func OldLogsCleanerWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// split legacy logs
			files, err := ioutil.ReadDir(logger.LogPath)
			if err != nil {
				zap.L().Error(fmt.Sprintf("failed to read logs directory: %v", err))
				continue
			}
			for _, file := range files {
				fn := file.Name()
				matched, err := regexp.MatchString(`^\d{4}-\d{2}\.log$`, filepath.Base(fn))
				if err != nil {
					zap.L().Error(fmt.Sprintf("regex failed: %v", err))
				}
				if matched {
					fullName := fmt.Sprintf("%s%s", logger.LogPath, fn)
					if err := splitLogFile(fullName); err != nil {
						zap.L().Error(fmt.Sprintf("failed to split legacy logfile %s: %v", fn, err))
						continue
					}
					if err := os.Remove(fullName); err != nil {
						zap.L().Error(fmt.Sprintf("failed to remove legacy logfile %s: %v", fn, err))
					}
				}
			}
			// clear logs
			if err := keepMostRecentFiles(logger.LogPath); err != nil {
				zap.L().Error(fmt.Sprintf("failed to clear old logs: %v", err))
			}
		}
	}
}

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
		for _, conn := range logger.SysLogSessions() {
			if err := conn.WriteMessage(1, logJSON); err != nil {
				zap.L().Error(fmt.Sprintf("error writing message: %v", err))
				conn.Close()
				logger.AddSysSessionToRemove(conn)
			}
		}
	}
}

func SysLogStreamerWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		logger.RemoveSysSessions()
		select {
		case <-ctx.Done():
			return nil
		case log := <-logger.SysLogChannel:
			var buffer bytes.Buffer
			err := json.Compact(&buffer, log)
			if err != nil {
				continue
			}
			escapedLog := buffer.Bytes()
			logJSON := []byte(fmt.Sprintf(`{"type":"system","history":false,"log":%s}`, escapedLog))
			for _, conn := range logger.SysLogSessions() {
				if err := conn.WriteMessage(1, logJSON); err != nil {
					zap.L().Error(fmt.Sprintf("error writing message: %v", err))
					conn.Close()
					logger.AddSysSessionToRemove(conn)
				}
			}
		}
	}
}

func DockerLogStreamer() {
	for {
		for container, sessionMap := range logger.DockerLogSessions() {
			for conn, live := range sessionMap {
				if !live {
					go streamToConn(container, conn)
					logger.SetDockerLogSessionLive(container, conn, true)
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func DockerLogStreamerWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		for container, sessionMap := range logger.DockerLogSessions() {
			for conn, live := range sessionMap {
				if !live {
					go streamToConnWithContext(ctx, container, conn)
					logger.SetDockerLogSessionLive(container, conn, true)
				}
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(1 * time.Second):
		}
	}
}

func DockerLogConnRemover() {
	for {
		c, _ := <-dockerLogCancelChannel
		logger.RemoveDockerLogSession(c.Container, c.Conn)
	}
}

func DockerLogConnRemoverWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case c := <-dockerLogCancelChannel:
			logger.RemoveDockerLogSession(c.Container, c.Conn)
		}
	}
}

func streamToConn(containerName string, conn *websocket.Conn) {
	streamToConnWithContext(context.Background(), containerName, conn)
}

func streamToConnWithContext(ctx context.Context, containerName string, conn *websocket.Conn) {
	if ctx == nil {
		ctx = context.Background()
	}
	defer func() {
		dockerLogCancelChannel <- DockerCancel{Container: containerName, Conn: conn}
	}()
	defer conn.Close()

	// Specify options to stream logs
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true, // Stream the logs
		Timestamps: true,
	}

	// Create a Docker client
	cli, err := dockerclient.New()
	if err != nil {
		zap.L().Error(fmt.Sprintf("failed to create Docker client: %v", err))
		return
	}
	defer cli.Close()

	out, err := cli.ContainerLogs(ctx, containerName, options)
	if err != nil {
		zap.L().Error(fmt.Sprintf("failed to get logs for container %s: %v", containerName, err))
		return
	}
	defer out.Close()

	// Read and print logs line by line
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		line := ""
		if len(scanner.Text()) > 8 {
			line = scanner.Text()[8:]
		}
		line = strings.ReplaceAll(line, "\\", "\\\\")
		logJSON := []byte(fmt.Sprintf(`{"type":"%s","history":false,"log":"%s"}`, containerName, line))
		if err := conn.WriteMessage(1, logJSON); err != nil {
			zap.L().Error(fmt.Sprintf("error writing message for %v: %v", containerName, err))
			return
		}
	}
	if err := scanner.Err(); err != nil {
		zap.L().Error(fmt.Sprintf("error reading logs: %v", err))
		return
	}
}

func splitLogFile(inputFile string) error {
	zap.L().Info(fmt.Sprintf("splitting legacy log file %s", inputFile))
	file, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	baseName := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	var (
		chunkFile   *os.File
		chunkWriter *bufio.Writer
		partNumber  int
		currentSize int
	)

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err.Error() != "EOF" {
			return err
		}

		if chunkFile == nil || currentSize+len(line) > MaxChunkSize {
			if chunkWriter != nil {
				chunkWriter.Flush()
				chunkFile.Close()
			}

			outputFileName := fmt.Sprintf("%s%s-part-%d.log", logger.LogPath, baseName, partNumber)
			chunkFile, err = os.Create(outputFileName)
			if err != nil {
				return err
			}
			chunkWriter = bufio.NewWriter(chunkFile)
			partNumber++
			currentSize = 0
		}

		n, err := chunkWriter.WriteString(line)
		if err != nil {
			return err
		}
		currentSize += n

		if err == io.EOF {
			break
		}

		if line == "" {
			break
		}
	}

	if chunkWriter != nil {
		chunkWriter.Flush()
		chunkFile.Close()
	}

	return nil
}

// Function to keep only the 10 most recent log files in a directory
func keepMostRecentFiles(dirPath string) error {
	// Read the directory
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %v", err)
	}

	recentFiles := logger.MostRecentPartedLogPaths(dirPath, files, 10)

	// Optionally, delete files that are not in recentFiles
	for _, file := range files {
		fullPath := filepath.Join(dirPath, file.Name())
		if !logContains(recentFiles, fullPath) {
			err := os.Remove(fullPath)
			if err != nil {
				fmt.Printf("Failed to delete file: %s\n", fullPath)
			}
		}
	}

	return nil
}

// Helper function to check if a string is in a slice
func logContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
