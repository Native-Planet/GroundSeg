package logger

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"groundseg/config"
	"groundseg/dockerclient"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggerRuntimeState struct {
	initMu        sync.Mutex
	loggerInitErr error
	loggerInitState loggerInitLifecycle

	sysLogPath   string
	pathMu       sync.RWMutex
	sysLogSinkMu sync.RWMutex
	sysLogSink   logStreamSink
}

var runtimeState = &loggerRuntimeState{
	loggerInitState: loggerInitNotInitialized,
	sysLogSink:      logStreamNoopSink{},
}

func getLoggerState() *loggerRuntimeState {
	return runtimeState
}

func setLoggerPath(path string) {
	state := getLoggerState()
	state.pathMu.Lock()
	defer state.pathMu.Unlock()
	state.sysLogPath = path
}

func loggerPath() string {
	state := getLoggerState()
	state.pathMu.RLock()
	defer state.pathMu.RUnlock()
	return state.sysLogPath
}

func setLogstreamSink(runtime logStreamSink) {
	state := getLoggerState()
	state.sysLogSinkMu.Lock()
	defer state.sysLogSinkMu.Unlock()
	if runtime == nil {
		state.sysLogSink = logStreamNoopSink{}
		return
	}
	state.sysLogSink = runtime
}

func getLogstreamSink() logStreamSink {
	state := getLoggerState()
	state.sysLogSinkMu.RLock()
	defer state.sysLogSinkMu.RUnlock()
	sink := state.sysLogSink
	if sink == nil {
		return logStreamNoopSink{}
	}
	return sink
}

type loggerRuntime struct {
	mkdirAllFn     func(string, os.FileMode) error
	pathResolverFn func() (string, error)
}

func defaultLoggerRuntime() loggerRuntime {
	return loggerRuntime{
		mkdirAllFn:     os.MkdirAll,
		pathResolverFn: makeLogPath,
	}
}

func sanitizeLoggerRuntime(runtime loggerRuntime) loggerRuntime {
	defaultRuntime := defaultLoggerRuntime()
	if runtime.mkdirAllFn == nil {
		runtime.mkdirAllFn = defaultRuntime.mkdirAllFn
	}
	if runtime.pathResolverFn == nil {
		runtime.pathResolverFn = defaultRuntime.pathResolverFn
	}
	return runtime
}

type logStreamNoopSink struct{}

func (logStreamNoopSink) PublishSystemLog(_ []byte) {}

type logStreamSink interface {
	PublishSystemLog([]byte)
}

type loggerInitLifecycle uint8

const (
	loggerInitNotInitialized loggerInitLifecycle = iota
	loggerInitInitializing
	loggerInitInitialized
)

const loggerFallbackLogPath = "/tmp/groundseg-logs/"

func LogPath() string {
	return loggerPath()
}

func SetLogPath(path string) {
	if path == "" {
		path = loggerFallbackLogPath
	}
	setLoggerPath(path)
}

func loggerInitErrState() error {
	state := getLoggerState()
	state.initMu.Lock()
	defer state.initMu.Unlock()
	return state.loggerInitErr
}

func setLoggerInitErr(err error) {
	state := getLoggerState()
	state.initMu.Lock()
	defer state.initMu.Unlock()
	state.loggerInitErr = err
}

func loggerInitStateValue() loggerInitLifecycle {
	state := getLoggerState()
	state.initMu.Lock()
	defer state.initMu.Unlock()
	return state.loggerInitState
}

func setLoggerInitState(stateValue loggerInitLifecycle) {
	state := getLoggerState()
	state.initMu.Lock()
	defer state.initMu.Unlock()
	state.loggerInitState = stateValue
}

func printStartupBanner() {
	fmt.Println("                                       !G#:\n                                   " +
		" .7G@@@^\n          .                       :J#@@@@P.\n     .75GB#BG57.                ~5&@@" +
		"@&Y^  \n    ?&@@@@@@@@@&J             !G@@@@B?. .^ \n   Y@@@@@@@@@@@@@J         :?B@@@@G!  :" +
		"Y&&:\n   @@@@@@@@@@@@@@B       ^Y&@@@&5^  ~P&@@@:\n   Y@@@@@@@@@@@@@J     !P&@@@#J.  7B@@@@G" +
		"! \n    ?#@@@@@@@@@&?   .7B@@@@G7  :J#@@@&5~   \n     .!YGBBBGY7.  :J#@@@&P~  ^5&@@@#J:  !?." +
		"\n                ^5&@@@#J:  !G@@@@G7. .?B@@:\n             .7G@@@@G7. :J#@@@&P~  ^Y#@@@&:\n" +
		"            .P&&&&G!   ~B&&&#5^   ~#&&&&&P. \n\nＮａｔｉｖｅ Ｐｌａｎｅｔ")
	fmt.Println(" ▄▄ • ▄▄▄        ▄• ▄▌ ▐ ▄ ·▄▄▄▄  .▄▄ · ▄▄▄ . ▄▄ • 𝐯𝟐!\n▐█ ▀ ▪▀▄ █·▪     █▪██▌•" +
		"█▌▐███▪ ██ ▐█ ▀. ▀▄.▀·▐█ ▀ ▪\n▄█ ▀█▄▐▀▀▄  ▄█▀▄ █▌▐█▌▐█▐▐▌▐█· ▐█▌▄▀▀▀█▄▐▀▀▪▄▄█ ▀█▄ 🪐\n▐█▄▪▐█▐" +
		"█•█▌▐█▌.▐▌▐█▄█▌██▐█▌██. ██ ▐█▄▪▐█▐█▄▄▌▐█▄▪▐█\n·▀▀▀▀ .▀  ▀ ▀█▄▀▪ ▀▀▀ ▀▀ █▪▀▀▀▀▀•  ▀▀▀▀  ▀▀▀" +
		" ·▀▀▀▀ (~)")
}

func loggerLevelFromArgs(args []string) zapcore.Level {
	for _, arg := range args {
		if arg == "dev" {
			return zap.DebugLevel
		}
	}
	return zap.InfoLevel
}

func resolveLoggerPath(runtime loggerRuntime) (string, error) {
	runtime = sanitizeLoggerRuntime(runtime)
	return runtime.pathResolverFn()
}

func makeLoggerCore(level zapcore.Level) zapcore.Core {
	// write logs to file
	fw := FileWriter{}
	fileWriteSyncer := zapcore.AddSync(fw)

	// stdout
	consoleWriteSyncer := zapcore.AddSync(os.Stdout)

	// channel
	cw := ChanWriter{}
	wsWriteSyncer := zapcore.AddSync(cw)

	// encoder config
	encoderConfig := zap.NewDevelopmentEncoderConfig()

	// encoder
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	// zap core
	return zapcore.NewTee(
		zapcore.NewCore(encoder, fileWriteSyncer, level),
		zapcore.NewCore(encoder, consoleWriteSyncer, level),
		zapcore.NewCore(encoder, wsWriteSyncer, level),
	)
}

func ConfigureLogstreamRuntime(runtime logStreamSink) {
	setLogstreamSink(runtime)
}

func buildLogger(level zapcore.Level) *zap.Logger {
	core := makeLoggerCore(level)
	return zap.Must(zap.New(core, zap.AddCaller()), nil)
}

func Debug(v string) {
	zap.L().Debug(v)
}

func Debugf(format string, args ...any) {
	zap.L().Debug(fmt.Sprintf(format, args...))
}

func Info(v string) {
	zap.L().Info(v)
}

func Infof(format string, args ...any) {
	zap.L().Info(fmt.Sprintf(format, args...))
}

func Warn(v string) {
	zap.L().Warn(v)
}

func Warnf(format string, args ...any) {
	zap.L().Warn(fmt.Sprintf(format, args...))
}

func Error(v string) {
	zap.L().Error(v)
}

func Errorf(format string, args ...any) {
	zap.L().Error(fmt.Sprintf(format, args...))
}

// File Writer
type FileWriter struct{}

func (fw FileWriter) Write(p []byte) (n int, err error) {
	fileName := SysLogfile()
	// Open the file in append mode, create it if it doesn't exist
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
	sink := getLogstreamSink()
	sink.PublishSystemLog(p)
	return len(p), nil
}

func (cw ChanWriter) Sync() error {
	return nil
}

func Initialize() error {
	runtime := defaultLoggerRuntime()
	return InitializeWithRuntime(runtime)
}

func InitializeWithRuntime(runtime loggerRuntime) error {
	runtime = sanitizeLoggerRuntime(runtime)
	state := getLoggerState()
	state.initMu.Lock()
	defer state.initMu.Unlock()
	switch state.loggerInitState {
	case loggerInitInitializing:
		return state.loggerInitErr
	case loggerInitInitialized:
		return nil
	}

	state.loggerInitState = loggerInitInitializing
	state.loggerInitErr = nil
	storagePath, err := resolveLoggerPath(runtime)
	if err != nil {
		state.loggerInitErr = err
	} else {
		setLoggerPath(storagePath)
	}
	if LogPath() == "" {
		SetLogPath(loggerFallbackLogPath)
	}
	err = runtime.mkdirAllFn(LogPath(), 0755)
	zap.ReplaceGlobals(buildLogger(loggerLevelFromArgs(os.Args[1:])))
	printStartupBanner()
	if err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
		fmt.Print("\n\n.・。.・゜✭・.・✫・゜・。..・。.・゜✭・.・✫・゜・。.\n")
		fmt.Print("Please run GroundSeg as root!  \n    /) /)\n   ( . . )" +
			"\n   (  >< )\n Love, Native Planet\n")
		fmt.Print(".・。.・゜✭・.・✫・゜・。..・。.・゜✭・.・✫・゜・。.\n\n")
		SetLogPath(loggerFallbackLogPath)
		if mkErr := runtime.mkdirAllFn(LogPath(), 0755); mkErr != nil {
			fmt.Printf("Failed to create fallback log directory: %v\n", mkErr)
			state.loggerInitErr = fmt.Errorf("log path fallback failed: %w", mkErr)
		} else {
			state.loggerInitErr = fmt.Errorf("configured log path unavailable, using fallback: %w", err)
		}
	}
	zap.L().Info("Starting GroundSeg")
	zap.L().Info("Urbit is love <3")
	if state.loggerInitErr == nil {
		state.loggerInitState = loggerInitInitialized
	} else {
		state.loggerInitState = loggerInitNotInitialized
	}
	return state.loggerInitErr
}

func SysLogfile() string {
	currentTime := time.Now()
	curMonthYear := fmt.Sprintf("%d-%02d", currentTime.Year(), currentTime.Month())
	count := 0
	for {
		// check if y-m-part-n.log exists
		fn := fmt.Sprintf("%s%s-part-%v.log", LogPath(), curMonthYear, count)
		// file doesn't exist, use this
		file, err := os.Stat(fn)
		if err != nil {
			return fn
		}
		// check if already 10MB
		const maxSize int64 = 10 * 1024 * 1024
		if file.Size() >= maxSize {
			count = count + 1
			continue
		}
		return fn
	}
}

func PrevSysLogfile() string {
	currentTime := time.Now()
	year := currentTime.Year()
	month := currentTime.Month()
	if month == time.January {
		year = year - 1
		month = time.December
	} else {
		month = month - 1
	}
	return fmt.Sprintf("%s%d-%02d.log", LogPath(), year, month)
}

func makeLogPath() (string, error) {
	return config.GetStoragePath("logs")
}

func getDockerLogs(name string) ([]byte, error) {
	cli, err := dockerclient.New()
	if err != nil {
		return []byte{}, fmt.Errorf("create docker client: %w", err)
	}

	options := container.LogsOptions{ShowStdout: true, ShowStderr: true, Timestamps: true}
	logs, err := cli.ContainerLogs(context.Background(), name, options)
	if err != nil {
		return []byte{}, fmt.Errorf("read docker logs for %s: %w", name, err)
	}
	defer logs.Close()

	var logEntries []string
	scanner := bufio.NewScanner(logs)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 8 {
			line = line[8:]
		}
		jsonLine, err := json.Marshal(line)
		if err != nil {
			return []byte{}, fmt.Errorf("marshal docker log line for %s: %w", name, err)
		}
		logEntries = append(logEntries, string(jsonLine))
	}
	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return []byte{}, fmt.Errorf("Error reading docker logs: %w", err)
	}
	jsArray := fmt.Sprintf("[%s]", strings.Join(logEntries, ", "))

	// Print the JavaScript array string
	return []byte(fmt.Sprintf(`{"type":"%s","history":true,"log":%s}`, name, jsArray)), nil
}

func RetrieveSysLogHistory() ([]byte, error) {
	filePath := SysLogfile()
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return []byte{}, fmt.Errorf("Error opening file: %w", err)
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
		return []byte{}, fmt.Errorf("Error reading file: %w", err)
	}

	// Join the lines slice into a single string resembling a JavaScript array
	jsArray := fmt.Sprintf("[%s]", strings.Join(lines, ", "))
	fmt.Println(jsArray)

	// Print the JavaScript array string
	return []byte(fmt.Sprintf(`{"type":"system","history":true,"log":%s}`, jsArray)), nil
}
