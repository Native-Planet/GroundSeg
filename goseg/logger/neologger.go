package logger

import (
	"os"

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
