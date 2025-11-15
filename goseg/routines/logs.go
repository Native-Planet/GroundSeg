package routines

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
	"sort"
	"strconv"
	"strings"
	"time"

	// "io/ioutil"

	"sync"

	"github.com/docker/docker/api/types"
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
	for {
		// split legacy logs
		files, err := ioutil.ReadDir(logger.LogPath)
		if err != nil {
			// sleep after error
			zap.L().Error(fmt.Sprintf("failed to read logs directory: %v", err))
			time.Sleep(10 * time.Minute)
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
		// sleep after completed
		time.Sleep(10 * time.Minute)
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
	cli, err := dockerclient.New()
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
		line = strings.ReplaceAll(line, "\\", "\\\\")
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

// LogFile represents a structured log file with a date and part number.
type LogFile struct {
	Name      string
	Directory string
	Date      string
	Part      int
}

// Function to keep only the 10 most recent log files in a directory
func keepMostRecentFiles(dirPath string) error {
	// Read the directory
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Create a slice to store LogFile structs
	var logFiles []LogFile

	// Filter and parse log files
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".log") {
			parts := strings.Split(file.Name(), "-part-")
			if len(parts) == 2 {
				date := parts[0]
				partNumber, err := strconv.Atoi(strings.TrimSuffix(parts[1], ".log"))
				if err != nil {
					fmt.Printf("Failed to parse part number from file: %s\n", file.Name())
					continue
				}
				logFiles = append(logFiles, LogFile{
					Name:      file.Name(),
					Directory: dirPath,
					Date:      date,
					Part:      partNumber,
				})
			}
		}
	}

	// Sort log files first by Date, then by Part (both in descending order)
	sort.Slice(logFiles, func(i, j int) bool {
		if logFiles[i].Date == logFiles[j].Date {
			return logFiles[i].Part > logFiles[j].Part
		}
		return logFiles[i].Date > logFiles[j].Date
	})

	// Keep only the top 10 most recent files
	var recentFiles []string
	for i := 0; i < len(logFiles) && i < 10; i++ {
		recentFiles = append(recentFiles, filepath.Join(logFiles[i].Directory, logFiles[i].Name))
	}

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
