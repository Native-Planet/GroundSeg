package logger

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

var (
	// zap
	SysLogChannel    = make(chan []byte, 100)
	LogSessions      = make(map[string][]*websocket.Conn)
	SessionsToRemove = make(map[string][]*websocket.Conn)
)

// File Writer
type FileWriter struct{}

func (fw FileWriter) Write(p []byte) (n int, err error) {
	// Open the file in append mode, create it if it doesn't exist
	f, err := os.OpenFile(SysLogfile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	// Write the byte slice to the file
	n, err = f.Write(p)
	return n, err
}

// Sync implements the zapcore.WriteSyncer interface for ConsoleWriter.
func (fw FileWriter) Sync() error {
	return nil
}

// Channel Writer
type ChanWriter struct{}

func (cw ChanWriter) Write(p []byte) (n int, err error) {
	SysLogChannel <- p
	return len(p), nil
}

func (cw ChanWriter) Sync() error {
	return nil
}

func RetrieveLogHistory(logType string) ([]byte, error) {
	switch logType {
	case "system":
		return systemHistory()
	default:
		return []byte{}, fmt.Errorf("Unknown log type: %s", logType)
	}
}

func systemHistory() ([]byte, error) {
	filePath := SysLogfile()
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return []byte{}, fmt.Errorf("Error opening file: %v", err)
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)
	var lines []string

	// Read each line and append it to the lines slice
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return []byte{}, fmt.Errorf("Error reading file:", err)
	}

	// Join the lines slice into a single string resembling a JavaScript array
	jsArray := fmt.Sprintf("[%s]", strings.Join(lines, ", "))
	fmt.Println(jsArray)

	// Print the JavaScript array string
	return []byte(fmt.Sprintf(`{"type":"system","history":true,"log":%s}`, jsArray)), nil
}
