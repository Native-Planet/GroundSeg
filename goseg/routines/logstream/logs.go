package logstream

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"groundseg/dockerclient"
	"groundseg/internal/resource"
	"groundseg/logger"
	"groundseg/session"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

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

	maxLogCleanerConsecutiveFailures = 3
	maxDockerLogConsecutiveFailures  = 5
)

var legacyLogFileNamePattern = regexp.MustCompile(`^\d{4}-\d{2}\.log$`)

type LogstreamRuntime struct {
	sessionRuntime             session.LogstreamRuntime
	systemLogMessages          <-chan []byte
	dockerLogCancelChannel     chan DockerCancel
	systemLogParseFailureTotal uint64
}

func NewLogstreamRuntime(logRuntime session.LogstreamRuntime, systemLogMessages <-chan []byte) *LogstreamRuntime {
	if logRuntime == nil {
		logRuntime = session.LogstreamRuntimeState()
	}
	if systemLogMessages == nil && logRuntime != nil {
		systemLogMessages = logRuntime.SystemLogMessages()
	}
	return &LogstreamRuntime{
		sessionRuntime:         logRuntime,
		systemLogMessages:      systemLogMessages,
		dockerLogCancelChannel: make(chan DockerCancel, 100),
	}
}

func resolveLogstreamRuntime(runtime *LogstreamRuntime) *LogstreamRuntime {
	if runtime == nil {
		return NewLogstreamRuntime(session.LogstreamRuntimeState(), nil)
	}
	return runtime
}

const maxSystemLogParseWarnings = 5

func Configure(logRuntime session.LogstreamRuntime, systemLogMessages <-chan []byte) *LogstreamRuntime {
	return NewLogstreamRuntime(logRuntime, systemLogMessages)
}

// RunOldLogsCleanupLoop runs periodic log cleanup until context cancellation.
func RunOldLogsCleanupLoop() error {
	return RunOldLogsCleanupLoopWithContext(context.Background())
}

// RunOldLogsCleanupLoopWithContext runs periodic log cleanup until context cancellation.
func RunOldLogsCleanupLoopWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	failureCount := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := runLogCleanupCycle(); err != nil {
				failureCount++
				zap.L().Error(fmt.Sprintf("failed to clean up logs: %v", err))
				if failureCount >= maxLogCleanerConsecutiveFailures {
					return fmt.Errorf("log cleanup failed after %d consecutive attempts: %w", maxLogCleanerConsecutiveFailures, err)
				}
				continue
			}
			failureCount = 0
		}
	}
}

func runLogCleanupCycle() error {
	files, err := os.ReadDir(logger.LogPath())
	if err != nil {
		return fmt.Errorf("read logs directory %q: %w", logger.LogPath(), err)
	}
	var splitErrs []error
	for _, file := range files {
		fn := file.Name()
		matched := legacyLogFileNamePattern.MatchString(filepath.Base(fn))
		if !matched {
			continue
		}
		fullName := fmt.Sprintf("%s%s", logger.LogPath(), fn)
		if err := splitLogFile(fullName); err != nil {
			splitErrs = append(splitErrs, fmt.Errorf("split legacy logfile %s: %w", fn, err))
			continue
		}
		if err := os.Remove(fullName); err != nil {
			splitErrs = append(splitErrs, fmt.Errorf("remove legacy logfile %s: %w", fn, err))
		}
	}
	if err := keepMostRecentFiles(logger.LogPath()); err != nil {
		splitErrs = append(splitErrs, fmt.Errorf("clear old logs: %w", err))
	}
	return errors.Join(splitErrs...)
}

func SysLogStreamerWithRuntime(ctx context.Context, runtime *LogstreamRuntime) error {
	runtime = resolveLogstreamRuntime(runtime)
	if ctx == nil {
		ctx = context.Background()
	}
	writeFailure := func(conn *websocket.Conn, scope string, err error) {
		if conn == nil {
			return
		}
		if err != nil {
			zap.L().Error(fmt.Sprintf("%s: %v", scope, err))
		}
		if closeErr := conn.Close(); closeErr != nil {
			zap.L().Error(fmt.Sprintf("%s: failed to close websocket connection: %v", scope, closeErr))
		}
	}
	for {
		if runtime.sessionRuntime != nil {
			runtime.sessionRuntime.RemoveSysLogSessions()
		}
		select {
		case <-ctx.Done():
			return nil
		case log, ok := <-runtime.systemLogMessages:
			if !ok {
				return nil
			}
			var buffer bytes.Buffer
			if err := json.Compact(&buffer, log); err != nil {
				parseFailures := atomic.AddUint64(&runtime.systemLogParseFailureTotal, 1)
				if parseFailures <= maxSystemLogParseWarnings {
					sample := string(log)
					if len(sample) > 80 {
						sample = sample[:80] + "..."
					}
					zap.L().Warn(fmt.Sprintf("failed to compact system log payload (len=%d total=%d): %v sample=%q", len(log), parseFailures, err, sample))
				}
				continue
			}
			logJSON := []byte(fmt.Sprintf(`{"type":"system","history":false,"log":%s}`, buffer.Bytes()))
			if runtime.sessionRuntime == nil {
				continue
			}
			for _, conn := range runtime.sessionRuntime.SysLogSessions() {
				if err := conn.WriteMessage(1, logJSON); err != nil {
					writeFailure(conn, "system log websocket write failure", err)
					runtime.sessionRuntime.AddSysSessionToRemove(conn)
				}
			}
		}
	}
}

func DockerLogStreamerWithRuntime(ctx context.Context, runtime *LogstreamRuntime) error {
	runtime = resolveLogstreamRuntime(runtime)
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	streamErrs := make(chan error, 16)
	failures := 0
	recordFailure := func(err error) error {
		if err == nil {
			failures = 0
			return nil
		}
		failures++
		zap.L().Error(fmt.Sprintf("docker log stream failed: %v", err))
		if failures >= maxDockerLogConsecutiveFailures {
			return fmt.Errorf("docker log stream failed after %d consecutive attempts: %w", maxDockerLogConsecutiveFailures, err)
		}
		return nil
	}

	for {
		if runtime.sessionRuntime != nil {
			for container, sessionMap := range runtime.sessionRuntime.DockerLogSessions() {
				for conn, live := range sessionMap {
					if !live {
						streamConn := conn
						streamContainer := container
						go func(name string, streamConn *websocket.Conn) {
							streamErrs <- streamToConnWithRuntime(ctx, name, streamConn, runtime)
						}(streamContainer, streamConn)
						runtime.sessionRuntime.SetDockerLogSessionLive(container, conn, true)
					}
				}
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case err := <-streamErrs:
			if err := recordFailure(err); err != nil {
				return err
			}
		case <-ticker.C:
			if failures > 0 {
				failures = 0
			}
		}

		drain := true
		for {
			if !drain {
				break
			}
			select {
			case err := <-streamErrs:
				if err := recordFailure(err); err != nil {
					return err
				}
			default:
				drain = false
			}
		}
	}
}

func DockerLogConnRemoverWithRuntime(ctx context.Context, runtime *LogstreamRuntime) error {
	runtime = resolveLogstreamRuntime(runtime)
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case c := <-runtime.dockerLogCancelChannel:
			if runtime.sessionRuntime != nil {
				runtime.sessionRuntime.RemoveDockerLogSession(c.Container, c.Conn)
			}
		}
	}
}

func streamToConn(containerName string, conn *websocket.Conn) {
	if err := streamToConnWithRuntime(context.Background(), containerName, conn, nil); err != nil {
		zap.L().Warn(fmt.Sprintf("stream docker logs for %s: %v", containerName, err))
	}
}

func streamToConnWithRuntime(ctx context.Context, containerName string, conn *websocket.Conn, runtime *LogstreamRuntime) (err error) {
	runtime = resolveLogstreamRuntime(runtime)
	if ctx == nil {
		ctx = context.Background()
	}
	if conn == nil {
		return fmt.Errorf("missing websocket connection for %s", containerName)
	}
	defer func() {
		runtime.dockerLogCancelChannel <- DockerCancel{Container: containerName, Conn: conn}
	}()
	defer func() {
		err = resource.JoinCloseError(err, conn, "close docker log websocket connection")
	}()

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
	}

	cli, err := dockerclient.New()
	if err != nil {
		return fmt.Errorf("create docker client for %s: %w", containerName, err)
	}
	defer func() {
		err = resource.JoinCloseError(err, cli, "close docker client")
	}()

	out, err := cli.ContainerLogs(ctx, containerName, options)
	if err != nil {
		return fmt.Errorf("read docker logs for %s: %w", containerName, err)
	}
	defer func() {
		err = resource.JoinCloseError(err, out, "close docker log stream")
	}()

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		line := ""
		if len(scanner.Text()) > 8 {
			line = scanner.Text()[8:]
		}
		line = strings.ReplaceAll(line, "\\", "\\\\")
		logJSON := []byte(fmt.Sprintf(`{"type":"%s","history":false,"log":"%s"}`, containerName, line))
		if connWriteErr := conn.WriteMessage(1, logJSON); connWriteErr != nil {
			err = resource.JoinCloseError(
				fmt.Errorf("write docker log websocket message for %s: %w", containerName, connWriteErr),
				conn,
				"close docker log websocket connection",
			)
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read docker logs stream for %s: %w", containerName, err)
	}
	return nil
}

func splitLogFile(inputFile string) error {
	zap.L().Info(fmt.Sprintf("splitting legacy log file %s", inputFile))
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("open legacy log file %s: %w", inputFile, err)
	}
	defer func() {
		err = resource.JoinCloseError(err, file, "close legacy log file")
	}()

	reader := bufio.NewReader(file)

	baseName := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	var (
		chunkFile   *os.File
		chunkWriter *bufio.Writer
		partNumber  int
		currentSize int
	)
	closeChunk := func() error {
		if chunkWriter == nil || chunkFile == nil {
			return nil
		}
		if flushErr := chunkWriter.Flush(); flushErr != nil {
			return fmt.Errorf("flush split log file chunk %s-part-%d.log: %w", baseName, partNumber, flushErr)
		}
		if closeErr := chunkFile.Close(); closeErr != nil {
			return fmt.Errorf("close split log file chunk %s-part-%d.log: %w", baseName, partNumber, closeErr)
		}
		chunkWriter = nil
		chunkFile = nil
		return nil
	}
	defer func() {
		if chunkCloseErr := closeChunk(); chunkCloseErr != nil {
			err = errors.Join(err, chunkCloseErr)
		}
	}()

	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return fmt.Errorf("read legacy log file %s: %w", inputFile, readErr)
		}

		if chunkFile == nil || currentSize+len(line) > MaxChunkSize {
			if chunkWriter != nil {
				if err := closeChunk(); err != nil {
					return err
				}
			}

			outputFileName := fmt.Sprintf("%s%s-part-%d.log", logger.LogPath(), baseName, partNumber)
			chunkFile, err = os.Create(outputFileName)
			if err != nil {
				return fmt.Errorf("create split log file %s: %w", outputFileName, err)
			}
			chunkWriter = bufio.NewWriter(chunkFile)
			partNumber++
			currentSize = 0
		}

		n, writeErr := chunkWriter.WriteString(line)
		if writeErr != nil {
			return fmt.Errorf("write split log file %s: %w", inputFile, writeErr)
		}
		currentSize += n

		if errors.Is(readErr, io.EOF) {
			break
		}

		if line == "" {
			break
		}
	}

	if chunkWriter != nil {
		if err := closeChunk(); err != nil {
			return err
		}
	}

	return nil
}

// Function to keep only the 10 most recent log files in a directory
func keepMostRecentFiles(dirPath string) error {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}
	fileInfos, err := dirEntriesToFileInfos(files)
	if err != nil {
		return fmt.Errorf("failed to convert log directory entries to file info: %w", err)
	}

	recentFiles := logger.MostRecentPartedLogPaths(dirPath, fileInfos, 10)

	for _, file := range files {
		fullPath := filepath.Join(dirPath, file.Name())
		if !logContains(recentFiles, fullPath) {
			if err := os.Remove(fullPath); err != nil {
				return fmt.Errorf("failed to delete file %s: %w", fullPath, err)
			}
		}
	}

	return nil
}

func logContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func dirEntriesToFileInfos(entries []os.DirEntry) ([]os.FileInfo, error) {
	fileInfos := make([]os.FileInfo, 0, len(entries))
	for _, entry := range entries {
		fileInfo, err := entry.Info()
		if err != nil {
			return nil, err
		}
		fileInfos = append(fileInfos, fileInfo)
	}
	return fileInfos, nil
}
