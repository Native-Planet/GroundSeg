package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func resetLoggerGlobalsForTest(t *testing.T) {
	t.Helper()
	originalLogPath := LogPath
	originalSysSessions := SysLogSessions
	originalToRemove := SysSessionsToRemove

	t.Cleanup(func() {
		LogPath = originalLogPath
		SysLogSessions = originalSysSessions
		SysSessionsToRemove = originalToRemove
	})
}

func drainSysLogChannel() {
	for {
		select {
		case <-SysLogChannel:
		default:
			return
		}
	}
}

func TestChanWriterWritePublishesBytes(t *testing.T) {
	drainSysLogChannel()

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
	case got := <-SysLogChannel:
		if !reflect.DeepEqual(got, payload) {
			t.Fatalf("unexpected channel payload: got %q want %q", got, payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SysLogChannel write")
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
	relativePath := makeLogPath()
	if relativePath != "/opt/nativeplanet/groundseg/logs/" && relativePath != "/media/data/logs/" {
		t.Fatalf("unexpected relative-path fallback result: %q", relativePath)
	}

	t.Setenv("GS_BASE_PATH", "/tmp/groundseg-logger-test")
	absolutePath := makeLogPath()
	if absolutePath != "/tmp/groundseg-logger-test/logs/" && absolutePath != "/media/data/logs/" {
		t.Fatalf("unexpected absolute-path result: %q", absolutePath)
	}
}

func TestRemoveSysSessionsDropsQueuedConnections(t *testing.T) {
	resetLoggerGlobalsForTest(t)

	a := &websocket.Conn{}
	b := &websocket.Conn{}
	c := &websocket.Conn{}
	SysLogSessions = []*websocket.Conn{a, b, c}
	SysSessionsToRemove = []*websocket.Conn{b}

	RemoveSysSessions()

	if !reflect.DeepEqual(SysLogSessions, []*websocket.Conn{a, c}) {
		t.Fatalf("unexpected remaining sessions: %+v", SysLogSessions)
	}
	if len(SysSessionsToRemove) != 0 {
		t.Fatalf("expected SysSessionsToRemove to be cleared, got %+v", SysSessionsToRemove)
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
