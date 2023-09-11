package logger

import (
	"log/slog"
	"io"
	"os"
	"fmt"
	"time"
	"sync"
)

var (
	logPath    string
	logFile    *os.File
	multiWriter io.Writer
	Logger     *slog.Logger
)


type MuMultiWriter struct {
	Writers []io.Writer
	Mu      sync.Mutex
}

func init() {
	basePath := os.Getenv("BASE_PATH")
	if basePath == "" {
		basePath = "/opt/nativeplanet/groundseg/"
	}
	logPath = basePath + "logs/"
    err := os.MkdirAll(logPath, 0755)
    if err != nil {
        panic(fmt.Sprintf("Failed to create log directory: %v", err))
    }
	logFile, err := os.OpenFile(sysLogfile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file: %v", err))
	}

	multiWriter = muMultiWriter(os.Stdout, logFile)
	Logger = slog.New(slog.NewJSONHandler(multiWriter, nil))
}

func sysLogfile() string {
	currentTime := time.Now()
	return fmt.Sprintf("%s%d-%02d.log", logPath, currentTime.Year(), currentTime.Month())
}

func muMultiWriter(writers ...io.Writer) *MuMultiWriter {
	return &MuMultiWriter{
		Writers: writers,
	}
}

func (m *MuMultiWriter) Write(p []byte) (n int, err error) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	var firstError error
	for _, w := range m.Writers {
		n, err := w.Write(p)
		if err != nil && firstError == nil {
			firstError = err
		}
		if n != len(p) && firstError == nil {
			firstError = io.ErrShortWrite
		}
	}
	return len(p), firstError
}
