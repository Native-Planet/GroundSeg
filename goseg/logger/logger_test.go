package logger

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type testSystemLogSink struct {
	ch chan []byte
}

func (sink *testSystemLogSink) PublishSystemLog(payload []byte) {
	sink.ch <- payload
}

func resetSystemLogSinkForTest(t *testing.T, ch chan []byte) {
	t.Helper()
	originalSink := sysLogSink
	configureSystemLogSink(&testSystemLogSink{ch: ch})
	t.Cleanup(func() {
		configureSystemLogSink(originalSink)
	})
}

func resetLoggerGlobalsForTest(t *testing.T) {
	t.Helper()
	originalLogPath := LogPath

	t.Cleanup(func() {
		LogPath = originalLogPath
	})
}

func drainSystemLogChannel(ch chan []byte) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

func TestChanWriterWritePublishesBytes(t *testing.T) {
	systemLogs := make(chan []byte, 2)
	resetSystemLogSinkForTest(t, systemLogs)
	drainSystemLogChannel(systemLogs)

	writer := ChanWriter{}
	payload := []byte(`{"msg":"hello"}`)
	n, err := writer.Write(payload)
	if err != nil {
		t.Fatalf("ChanWriter.Write returned error: %v", err)
	}
	if n != len(payload) {
		t.Fatalf("unexpected write length: got %d want %d", n, len(payload))
	}

	select {
	case got := <-systemLogs:
		if !reflect.DeepEqual(got, payload) {
			t.Fatalf("unexpected channel payload: got %q want %q", got, payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for system log sink write")
	}
}

func TestFileWriterWriteAppendsToCurrentLogFile(t *testing.T) {
	resetLoggerGlobalsForTest(t)
	LogPath = t.TempDir() + string(os.PathSeparator)

	writer := FileWriter{}
	first := []byte("first line\n")
	second := []byte("second line\n")

	if _, err := writer.Write(first); err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	if _, err := writer.Write(second); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	data, err := os.ReadFile(SysLogfile())
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if string(data) != string(first)+string(second) {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestSysLogfileRollsOverWhenCurrentPartIsFull(t *testing.T) {
	resetLoggerGlobalsForTest(t)
	LogPath = t.TempDir() + string(os.PathSeparator)

	now := time.Now()
	prefix := fmt.Sprintf("%d-%02d", now.Year(), now.Month())
	part0 := filepath.Join(LogPath, fmt.Sprintf("%s-part-0.log", prefix))
	if err := os.WriteFile(part0, []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to create part-0: %v", err)
	}
	const maxSize int64 = 10 * 1024 * 1024
	if err := os.Truncate(part0, maxSize); err != nil {
		t.Fatalf("failed to set part-0 size: %v", err)
	}

	got := SysLogfile()
	want := filepath.Join(LogPath, fmt.Sprintf("%s-part-1.log", prefix))
	if got != want {
		t.Fatalf("unexpected rollover target: got %q want %q", got, want)
	}
}

func TestPrevSysLogfileUsesPreviousMonth(t *testing.T) {
	resetLoggerGlobalsForTest(t)
	LogPath = t.TempDir() + string(os.PathSeparator)

	now := time.Now()
	year := now.Year()
	month := now.Month()
	if month == time.January {
		year--
		month = time.December
	} else {
		month--
	}
	want := fmt.Sprintf("%s%d-%02d.log", LogPath, year, month)

	if got := PrevSysLogfile(); got != want {
		t.Fatalf("unexpected previous logfile path: got %q want %q", got, want)
	}
}

func TestMakeLogPathHandlesRelativeAndAbsoluteBasePath(t *testing.T) {
	t.Setenv("GS_BASE_PATH", "relative/path")
	relativePath, err := makeLogPath()
	if err != nil {
		t.Fatalf("makeLogPath failed: %v", err)
	}
	if relativePath != "/opt/nativeplanet/groundseg/logs/" && relativePath != "/media/data/logs/" && relativePath != "/opt/nativeplanet/groundseg/logs" {
		t.Fatalf("unexpected relative-path fallback result: %q", relativePath)
	}

	t.Setenv("GS_BASE_PATH", "/tmp/groundseg-logger-test")
	absolutePath, err := makeLogPath()
	if err != nil {
		t.Fatalf("makeLogPath failed: %v", err)
	}
	if absolutePath != "/tmp/groundseg-logger-test/logs/" && absolutePath != "/media/data/logs/" && absolutePath != "/tmp/groundseg-logger-test/logs" {
		t.Fatalf("unexpected absolute-path result: %q", absolutePath)
	}
}

func resetLoggerInitForTest(t *testing.T) {
	t.Helper()
	originalErr := loggerInitErr
	originalState := loggerInitState
	originalMkdir := mkdirAllFn
	t.Cleanup(func() {
		loggerInitErr = originalErr
		loggerInitState = originalState
		mkdirAllFn = originalMkdir
	})
	loggerInitState = loggerInitNotInitialized
	loggerInitErr = nil
}

func TestInitializeReturnsErrorWhenLogDirectoryFails(t *testing.T) {
	resetLoggerInitForTest(t)
	mkdirAllFn = func(_ string, _ os.FileMode) error {
		return errors.New("permission denied")
	}

	err := Initialize()
	if err == nil {
		t.Fatalf("expected Initialize to return an error when log directory creation fails")
	}
	if !strings.Contains(err.Error(), "log path fallback failed") && !strings.Contains(err.Error(), "configured log path unavailable") {
		t.Fatalf("expected fallback-init error, got: %v", err)
	}
	if LogPath != loggerFallbackLogPath {
		t.Fatalf("expected fallback log path to be used, got %q", LogPath)
	}
}

func TestInitializeSucceedsWithConfiguredPath(t *testing.T) {
	resetLoggerInitForTest(t)
	tmpDir := filepath.Join(t.TempDir(), "groundseg-logs")
	t.Setenv("GS_BASE_PATH", tmpDir)
	mkdirAllFn = func(path string, _ os.FileMode) error {
		return nil
	}

	err := Initialize()
	if err != nil {
		t.Fatalf("expected Initialize to succeed with writable configured path, got: %v", err)
	}
}

func TestLoggerLevelFromArgs(t *testing.T) {
	if got, want := loggerLevelFromArgs([]string{}), zap.InfoLevel; got != want {
		t.Fatalf("expected default level %v, got %v", want, got)
	}
	if got, want := loggerLevelFromArgs([]string{"server", "dev"}), zap.DebugLevel; got != want {
		t.Fatalf("expected dev level %v, got %v", want, got)
	}
}

func TestBuildLoggerRespectsLevel(t *testing.T) {
	logger := buildLogger(zapcore.ErrorLevel)
	if logger == nil {
		t.Fatal("expected logger to be created")
	}
	if entry := logger.Check(zapcore.ErrorLevel, "err"); entry == nil {
		t.Fatal("expected error level logs to be enabled")
	}
	if entry := logger.Check(zapcore.DebugLevel, "dbg"); entry != nil {
		t.Fatalf("unexpected debug logs to be enabled with error-level logger")
	}
}

func TestRetrieveSysLogHistoryReturnsJSONEnvelope(t *testing.T) {
	resetLoggerGlobalsForTest(t)
	LogPath = t.TempDir() + string(os.PathSeparator)

	logFile := SysLogfile()
	content := strings.Join([]string{`"first"`, `"second"`}, "\n") + "\n"
	if err := os.WriteFile(logFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to seed system log file: %v", err)
	}

	data, err := RetrieveSysLogHistory()
	if err != nil {
		t.Fatalf("RetrieveSysLogHistory returned error: %v", err)
	}

	var payload struct {
		Type    string   `json:"type"`
		History bool     `json:"history"`
		Log     []string `json:"log"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("failed to unmarshal system log history payload: %v", err)
	}
	if payload.Type != "system" || !payload.History {
		t.Fatalf("unexpected envelope metadata: %+v", payload)
	}
	if !reflect.DeepEqual(payload.Log, []string{"first", "second"}) {
		t.Fatalf("unexpected history entries: %+v", payload.Log)
	}
}
